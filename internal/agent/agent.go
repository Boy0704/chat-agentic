package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"agent-service/internal/skill"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

type Agent struct {
	client   *openai.Client
	model    string
	registry *skill.Registry
	logger   *slog.Logger
}

func New(client *openai.Client, model string, registry *skill.Registry, logger *slog.Logger) *Agent {
	return &Agent{
		client:   client,
		model:    model,
		registry: registry,
		logger:   logger,
	}
}

type RunInput struct {
	Message   string
	History   []openai.ChatCompletionMessage
	AppContext map[string]any
}

type RunResult struct {
	Reply      string
	SkillsUsed []string
	Usage      openai.Usage
}

// Run sends a message to the LLM and returns the final reply (non-streaming).
func (a *Agent) Run(ctx context.Context, input RunInput) (RunResult, error) {
	messages := buildMessages(input)
	tools := a.registry.ToOpenAITools()
	var skillsUsed []string
	var totalUsage openai.Usage

	for {
		resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    a.model,
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			return RunResult{}, fmt.Errorf("llm call: %w", err)
		}

		totalUsage.PromptTokens += resp.Usage.PromptTokens
		totalUsage.CompletionTokens += resp.Usage.CompletionTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		if choice.FinishReason != openai.FinishReasonToolCalls {
			return RunResult{
				Reply:      choice.Message.Content,
				SkillsUsed: skillsUsed,
				Usage:      totalUsage,
			}, nil
		}

		messages, skillsUsed = a.executeToolCalls(ctx, messages, choice.Message.ToolCalls, skillsUsed, input.AppContext, nil)
	}
}

// RunStream sends a message to the LLM and streams events to eventCh.
// The channel is closed when the stream completes or encounters an error.
func (a *Agent) RunStream(ctx context.Context, input RunInput, eventCh chan<- Event) {
	defer close(eventCh)

	messages := buildMessages(input)
	tools := a.registry.ToOpenAITools()
	var skillsUsed []string
	messageID := uuid.New().String()

	for {
		stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:    a.model,
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			eventCh <- Event{Type: EventTypeError, Error: err.Error()}
			return
		}

		var (
			fullContent  strings.Builder
			toolCalls    []openai.ToolCall
			finishReason openai.FinishReason
		)

		for {
			chunk, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				stream.Close()
				eventCh <- Event{Type: EventTypeError, Error: err.Error()}
				return
			}
			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			// Stream text tokens to client
			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				eventCh <- Event{Type: EventTypeToken, Content: choice.Delta.Content}
			}

			// Accumulate tool calls — they arrive in fragments across chunks
			for _, tc := range choice.Delta.ToolCalls {
				accumulateToolCall(&toolCalls, tc)
			}
		}
		stream.Close()

		// Add full assistant turn to history
		messages = append(messages, openai.ChatCompletionMessage{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   fullContent.String(),
			ToolCalls: toolCalls,
		})

		if finishReason != openai.FinishReasonToolCalls {
			eventCh <- Event{
				Type:       EventTypeDone,
				MessageID:  messageID,
				SkillsUsed: skillsUsed,
			}
			return
		}

		// Execute tool calls and continue the loop
		messages, skillsUsed = a.executeToolCalls(ctx, messages, toolCalls, skillsUsed, input.AppContext, eventCh)
	}
}

// executeToolCalls runs each tool call, appends results to messages, and returns updated state.
// When eventCh is non-nil, it emits skill_start and skill_result events.
func (a *Agent) executeToolCalls(
	ctx context.Context,
	messages []openai.ChatCompletionMessage,
	toolCalls []openai.ToolCall,
	skillsUsed []string,
	appCtx map[string]any,
	eventCh chan<- Event,
) ([]openai.ChatCompletionMessage, []string) {
	for _, tc := range toolCalls {
		skillName := tc.Function.Name
		skillsUsed = append(skillsUsed, skillName)

		if eventCh != nil {
			eventCh <- Event{Type: EventTypeSkillStart, Skill: skillName}
		}
		a.logger.InfoContext(ctx, "executing skill", "skill", skillName)

		var params map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			params = map[string]any{}
		}

		result, err := a.registry.Execute(ctx, skillName, params, appCtx)

		var toolContent string
		if err != nil {
			toolContent = fmt.Sprintf("Error: %s", err.Error())
			a.logger.ErrorContext(ctx, "skill error", "skill", skillName, "error", err)
		} else {
			b, _ := json.Marshal(result)
			toolContent = string(b)
			if eventCh != nil {
				eventCh <- Event{Type: EventTypeSkillResult, Skill: skillName, Summary: result.Summary}
			}
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			ToolCallID: tc.ID,
			Content:    toolContent,
		})
	}
	return messages, skillsUsed
}

// accumulateToolCall merges a streaming tool call delta into the accumulator slice.
// Tool calls arrive fragmented across chunks — Index tells us which slot to merge into.
func accumulateToolCall(toolCalls *[]openai.ToolCall, delta openai.ToolCall) {
	idx := 0
	if delta.Index != nil {
		idx = *delta.Index
	}
	for len(*toolCalls) <= idx {
		*toolCalls = append(*toolCalls, openai.ToolCall{})
	}
	tc := &(*toolCalls)[idx]
	if delta.ID != "" {
		tc.ID = delta.ID
	}
	if delta.Type != "" {
		tc.Type = delta.Type
	}
	if delta.Function.Name != "" {
		tc.Function.Name = delta.Function.Name
	}
	tc.Function.Arguments += delta.Function.Arguments
}

func buildMessages(input RunInput) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(input.History)+2)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: buildSystemPrompt(input.AppContext),
	})
	messages = append(messages, input.History...)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input.Message,
	})
	return messages
}

func buildSystemPrompt(appCtx map[string]any) string {
	base := "Kamu adalah asisten AI yang membantu pengguna dengan informasi bisnis. " +
		"Gunakan tools yang tersedia untuk mendapatkan data yang akurat. " +
		"Jawab dalam Bahasa Indonesia dengan ringkas dan jelas."
	if len(appCtx) > 0 {
		ctxJSON, _ := json.Marshal(appCtx)
		base += fmt.Sprintf("\n\nKonteks aplikasi: %s", string(ctxJSON))
	}
	return base
}

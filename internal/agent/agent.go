package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"agent-service/internal/skill"

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

func (a *Agent) Run(ctx context.Context, input RunInput) (RunResult, error) {
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

		for _, toolCall := range choice.Message.ToolCalls {
			skillName := toolCall.Function.Name
			skillsUsed = append(skillsUsed, skillName)

			var params map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
				return RunResult{}, fmt.Errorf("parse tool args for %s: %w", skillName, err)
			}

			a.logger.InfoContext(ctx, "executing skill", "skill", skillName)

			result, err := a.registry.Execute(ctx, skillName, params, input.AppContext)

			var toolContent string
			if err != nil {
				toolContent = fmt.Sprintf("Error menjalankan skill: %s", err.Error())
				a.logger.ErrorContext(ctx, "skill error", "skill", skillName, "error", err)
			} else {
				b, _ := json.Marshal(result)
				toolContent = string(b)
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: toolCall.ID,
				Content:    toolContent,
			})
		}
	}
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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-service/internal/agent"
	"agent-service/internal/session"
	"agent-service/internal/skill"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	agent      *agent.Agent
	sessions   *session.Store
	registry   *skill.Registry
	llmBaseURL string
}

func NewHandler(a *agent.Agent, s *session.Store, r *skill.Registry, llmBaseURL string) *Handler {
	return &Handler{agent: a, sessions: s, registry: r, llmBaseURL: llmBaseURL}
}

type ChatRequest struct {
	SessionID  string         `json:"session_id"`
	Message    string         `json:"message" binding:"required"`
	AppContext  map[string]any `json:"context"`
}

type ChatResponse struct {
	SessionID  string   `json:"session_id"`
	MessageID  string   `json:"message_id"`
	Reply      string   `json:"reply"`
	SkillsUsed []string `json:"skills_used"`
	Usage      any      `json:"usage"`
}

func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	history, err := h.sessions.Get(c.Request.Context(), req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load session"})
		return
	}

	result, err := h.agent.Run(c.Request.Context(), agent.RunInput{
		Message:    req.Message,
		History:    history,
		AppContext:  req.AppContext,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.sessions.Append(c.Request.Context(), req.SessionID, req.Message, result.Reply); err != nil {
		c.Header("X-Session-Warning", "failed to save session")
	}

	c.JSON(http.StatusOK, ChatResponse{
		SessionID:  req.SessionID,
		MessageID:  uuid.New().String(),
		Reply:      result.Reply,
		SkillsUsed: result.SkillsUsed,
		Usage:      result.Usage,
	})
}

func (h *Handler) ChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	history, err := h.sessions.Get(c.Request.Context(), req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load session"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	eventCh := make(chan agent.Event, 16)

	go h.agent.RunStream(c.Request.Context(), agent.RunInput{
		Message:    req.Message,
		History:    history,
		AppContext:  req.AppContext,
	}, eventCh)

	var (
		fullReply  strings.Builder
		skillsUsed []string
	)

	c.Stream(func(w io.Writer) bool {
		event, ok := <-eventCh
		if !ok {
			return false
		}

		if event.Type == agent.EventTypeToken {
			fullReply.WriteString(event.Content)
		}
		if event.Type == agent.EventTypeDone {
			skillsUsed = event.SkillsUsed
		}

		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		return true
	})

	// Save to session after stream ends.
	// Use a fresh context — the request context is already cancelled at this point.
	if fullReply.Len() > 0 {
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.sessions.Append(saveCtx, req.SessionID, req.Message, fullReply.String())
	}
	_ = skillsUsed
}

type skillInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Required    []string `json:"required_params"`
}

func (h *Handler) ListSkills(c *gin.Context) {
	manifests := h.registry.List()
	skills := make([]skillInfo, 0, len(manifests))
	for _, m := range manifests {
		skills = append(skills, skillInfo{
			Name:        m.Name,
			Description: m.Description,
			Required:    m.Parameters.Required,
		})
	}
	c.JSON(http.StatusOK, gin.H{"skills": skills})
}

func (h *Handler) GetSession(c *gin.Context) {
	rows, err := h.sessions.GetHistory(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id": c.Param("id"),
		"messages":   rows,
	})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	if err := h.sessions.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete session"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Health(c *gin.Context) {
	checks := map[string]string{}
	status := "ok"
	httpStatus := http.StatusOK

	// DB check
	pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := h.sessions.Ping(pingCtx); err != nil {
		checks["db"] = "error: " + err.Error()
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["db"] = "ok"
	}

	// LLM reachability check (quick HTTP probe — any HTTP response counts as reachable)
	if h.llmBaseURL != "" {
		llmCtx, llmCancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer llmCancel()
		req, _ := http.NewRequestWithContext(llmCtx, http.MethodGet, h.llmBaseURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			checks["llm"] = "unreachable"
			if status == "ok" {
				status = "degraded"
			}
		} else {
			resp.Body.Close()
			checks["llm"] = "ok"
		}
	}

	c.JSON(httpStatus, gin.H{
		"status":  status,
		"version": "1.0.0",
		"checks":  checks,
	})
}

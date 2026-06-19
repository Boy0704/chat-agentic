package api

import (
	"net/http"

	"agent-service/internal/agent"
	"agent-service/internal/session"
	"agent-service/internal/skill"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	agent    *agent.Agent
	sessions *session.Store
	registry *skill.Registry
}

func NewHandler(a *agent.Agent, s *session.Store, r *skill.Registry) *Handler {
	return &Handler{agent: a, sessions: s, registry: r}
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
		Message:   req.Message,
		History:   history,
		AppContext: req.AppContext,
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
	c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "1.0.0"})
}

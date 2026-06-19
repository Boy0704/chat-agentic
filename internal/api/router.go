package api

import (
	"log/slog"
	"time"

	"agent-service/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(h *Handler, cfg config.ServerConfig, logger *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(SlogLogger(logger))
	r.Use(corsMiddleware(cfg.CORS))

	if cfg.MaxBodyBytes > 0 {
		r.Use(BodySizeLimit(cfg.MaxBodyBytes))
	}

	r.GET("/health", h.Health)

	v1 := r.Group("/api/v1", AuthMiddleware(cfg.APIKey))

	if cfg.RateLimit.Enabled && cfg.RateLimit.RequestsPerMinute > 0 {
		rl := newIPRateLimiter(cfg.RateLimit.RequestsPerMinute)
		v1.Use(RateLimitMiddleware(rl))
	}

	v1.POST("/chat", h.Chat)
	v1.POST("/chat/stream", h.ChatStream)
	v1.GET("/sessions/:id", h.GetSession)
	v1.DELETE("/sessions/:id", h.DeleteSession)
	v1.GET("/skills", h.ListSkills)

	return r
}

func corsMiddleware(cfg config.CORSConfig) gin.HandlerFunc {
	origins := cfg.AllowOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Type"},
		AllowCredentials: len(origins) == 1 && origins[0] == "*" == false,
		MaxAge:           12 * time.Hour,
	})
}

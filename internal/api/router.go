package api

import "github.com/gin-gonic/gin"

func SetupRouter(h *Handler, apiKey string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/health", h.Health)

	v1 := r.Group("/api/v1", AuthMiddleware(apiKey))
	{
		v1.POST("/chat", h.Chat)
		v1.POST("/chat/stream", h.ChatStream)
		v1.GET("/sessions/:id", h.GetSession)
		v1.DELETE("/sessions/:id", h.DeleteSession)
		v1.GET("/skills", h.ListSkills)
	}

	return r
}

package apikey

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers API key routes on the given group (expects JWT Auth middleware).
func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.DELETE("/:id", h.Delete)
}

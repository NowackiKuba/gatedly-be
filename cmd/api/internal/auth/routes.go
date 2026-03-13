package auth

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts auth routes on the given group (no auth required).
func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.Refresh)
}

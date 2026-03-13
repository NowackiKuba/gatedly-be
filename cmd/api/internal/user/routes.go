package user

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts user routes on the given group. Expects group to use Auth middleware.
func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.GET("/me", h.GetMe)
	g.PATCH("/me", h.UpdateMe)
}

package evaluation

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers evaluation routes on the given group.
// Group path should be /evaluation so: POST /api/v1/evaluation, POST /api/v1/evaluation/batch.
func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("", h.Evaluate)
	g.POST("/batch", h.EvaluateBatch)
}

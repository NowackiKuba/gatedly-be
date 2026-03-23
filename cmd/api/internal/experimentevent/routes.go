package experimentevent

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("", h.Create)
	g.GET("", h.GetByExperimentID)
}

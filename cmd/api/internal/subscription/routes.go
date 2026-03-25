package subscription

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/me", h.GetMe)
	g.GET("/:id", h.GetByID)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

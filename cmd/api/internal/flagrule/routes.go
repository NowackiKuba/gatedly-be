package flagrule

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("", h.Create)
	g.GET("", h.ListByFlag)
	g.GET("/:id", h.GetByID)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

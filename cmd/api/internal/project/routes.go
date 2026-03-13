package project

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("/", h.Create)
	g.GET("/", h.ListForUser)
	g.GET("/:id", h.GetByID)
	g.GET("/slug/:slug", h.GetBySlug)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

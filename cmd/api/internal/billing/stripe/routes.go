package billing

import "github.com/gin-gonic/gin"

// RegisterRoutes registers billing routes on the given group (expects JWT Auth middleware).
func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.POST("/checkout", h.Checkout)
	g.POST("/portal", h.Portal)
	g.GET("/usage", h.Usage)
	g.GET("/payment-method", h.GetPaymentMethod)
	g.GET("/invoices", h.GetInvoices)
	g.POST("/cancel", h.Cancel)
}

// RegisterWebhookRoute registers the Stripe webhook on the root router group (no auth).
func RegisterWebhookRoute(g *gin.RouterGroup, wh *WebhookHandler) {
	g.POST("/webhooks/stripe", func(c *gin.Context) {
		wh.Handle(c.Writer, c.Request)
	})
}

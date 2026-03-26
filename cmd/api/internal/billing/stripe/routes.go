package billing

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
		// Read the raw request body before Gin middleware can consume it
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		// Restore the body so the webhook handler can read it
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		wh.Handle(c.Writer, c.Request)
	})
}

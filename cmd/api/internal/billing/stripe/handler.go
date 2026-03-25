package billing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v84"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/cmd/api/internal/packet"
	"toggly.com/m/cmd/api/internal/subscription"
	"toggly.com/m/pkg/response"
)

// Handler exposes billing endpoints.
type Handler struct {
	svc         Service
	packetSvc   packet.Service
	subSvc      subscription.Service
	successURL  string
	cancelURL   string
}

// NewHandler returns a new billing Handler.
func NewHandler(
	svc Service,
	packetSvc packet.Service,
	subSvc subscription.Service,
	successURL string,
	cancelURL string,
) *Handler {
	return &Handler{
		svc:        svc,
		packetSvc:  packetSvc,
		subSvc:     subSvc,
		successURL: successURL,
		cancelURL:  cancelURL,
	}
}

// CreateCheckoutRequest is the body for POST /billing/checkout.
type CreateCheckoutRequest struct {
	PacketID uuid.UUID `json:"packetId"`
}

// CreateCheckoutResponse is returned after a session is created.
type CreateCheckoutResponse struct {
	URL string `json:"url"`
}

// Checkout creates a Stripe Checkout Session for the given packet.
// POST /billing/checkout
func (h *Handler) Checkout(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	var req CreateCheckoutRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if req.PacketID == uuid.Nil {
		response.Error(c.Writer, response.BadRequest("packetId is required"))
		c.Abort()
		return
	}

	p, err := h.packetSvc.GetById(c.Request.Context(), req.PacketID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	mode := string(stripe.CheckoutSessionModeSubscription)
	quantity := int64(1)

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(mode),
		SuccessURL: stripe.String(h.successURL),
		CancelURL:  stripe.String(h.cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(p.StripePriceID),
				Quantity: &quantity,
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": userIDStr,
			},
		},
		ClientReferenceID: stripe.String(userIDStr),
	}

	if p.TrialDays > 0 {
		trialDays := int64(p.TrialDays)
		params.SubscriptionData.TrialPeriodDays = &trialDays
	}

	sess, err := h.svc.CreateCheckoutSession(c.Request.Context(), params)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusCreated, CreateCheckoutResponse{URL: sess.URL})
}

// Portal creates a Stripe Billing Portal Session for the authenticated user.
// POST /billing/portal
func (h *Handler) Portal(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c.Writer, response.Unauthorized("invalid user context"))
		c.Abort()
		return
	}

	sub, err := h.subSvc.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(sub.StripeCustomerID),
		ReturnURL: stripe.String(h.cancelURL),
	}

	sess, err := h.svc.CreatePortalSession(c.Request.Context(), params)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusCreated, CreateCheckoutResponse{URL: sess.URL})
}

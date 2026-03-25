package billing

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v84"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/cmd/api/internal/packet"
	"toggly.com/m/cmd/api/internal/subscription"
	"toggly.com/m/pkg/response"
)

// Handler exposes billing endpoints.
type Handler struct {
	svc        Service
	packetSvc  packet.Service
	subSvc     subscription.Service
	db         *gorm.DB
	successURL string
	cancelURL  string
}

// NewHandler returns a new billing Handler.
func NewHandler(
	svc Service,
	packetSvc packet.Service,
	subSvc subscription.Service,
	db *gorm.DB,
	successURL string,
	cancelURL string,
) *Handler {
	return &Handler{
		svc:        svc,
		packetSvc:  packetSvc,
		subSvc:     subSvc,
		db:         db,
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

// isNotFound returns true when err is a 404 AppError.
func isNotFound(err error) bool {
	var appErr *response.AppError
	return errors.As(err, &appErr) && appErr.Status == 404
}

// usageStat holds used/limit for a single resource.
type usageStat struct {
	Used  int  `json:"used"`
	Limit *int `json:"limit"`
}

// BillingUsageResponse is returned by GET /billing/usage.
type BillingUsageResponse struct {
	Evaluations  usageStat `json:"evaluations"`
	Flags        usageStat `json:"flags"`
	Projects     usageStat `json:"projects"`
	Environments usageStat `json:"environments"`
}

// Usage returns the current resource usage for the authenticated user.
// GET /billing/usage
func (h *Handler) Usage(c *gin.Context) {
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

	// Count projects owned by user.
	var projectCount int64
	h.db.WithContext(c.Request.Context()).
		Model(&domain.Project{}).
		Where("owner_id = ?", userID).
		Count(&projectCount)

	// Count flags across all user's projects.
	var flagCount int64
	h.db.WithContext(c.Request.Context()).
		Model(&domain.Flag{}).
		Joins("JOIN projects ON projects.id = flags.project_id").
		Where("projects.owner_id = ? AND projects.deleted_at IS NULL AND flags.deleted_at IS NULL", userID).
		Count(&flagCount)

	// Count environments across all user's projects.
	var envCount int64
	h.db.WithContext(c.Request.Context()).
		Model(&domain.Environment{}).
		Joins("JOIN projects ON projects.id = environments.project_id").
		Where("projects.owner_id = ? AND projects.deleted_at IS NULL AND environments.deleted_at IS NULL", userID).
		Count(&envCount)

	freeFlagLimit := 3
	freeProjLimit := 1
	freeEnvLimit := 2
	freeEvalLimit := 10000

	out := BillingUsageResponse{
		Evaluations:  usageStat{Used: 0, Limit: &freeEvalLimit},
		Flags:        usageStat{Used: int(flagCount), Limit: &freeFlagLimit},
		Projects:     usageStat{Used: int(projectCount), Limit: &freeProjLimit},
		Environments: usageStat{Used: int(envCount), Limit: &freeEnvLimit},
	}

	// Overlay limits from subscription packet if one exists.
	sub, err := h.subSvc.GetByUserID(c.Request.Context(), userID)
	if err == nil && sub != nil && sub.Packet != nil {
		l := sub.Packet.Limits
		out.Evaluations.Used = sub.EvaluationsUsed
		evalLimit := l.MonthlyEvaluations
		out.Evaluations.Limit = &evalLimit
		flagLimit := l.MaxFlags
		out.Flags.Limit = &flagLimit
		projLimit := l.MaxProjects
		out.Projects.Limit = &projLimit
		envLimit := l.MaxEnvironments
		out.Environments.Limit = &envLimit
	}

	response.JSON(c.Writer, http.StatusOK, out)
}

// PaymentMethodResponse is returned by GET /billing/payment-method.
type PaymentMethodResponse struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int64  `json:"expMonth"`
	ExpYear  int64  `json:"expYear"`
}

// GetPaymentMethod returns the default payment method for the authenticated user.
// GET /billing/payment-method
func (h *Handler) GetPaymentMethod(c *gin.Context) {
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
		if isNotFound(err) {
			response.JSON(c.Writer, http.StatusOK, nil)
			return
		}
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	pm, err := h.svc.GetDefaultPaymentMethod(c.Request.Context(), sub.StripeCustomerID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if pm == nil || pm.Card == nil {
		response.JSON(c.Writer, http.StatusOK, nil)
		return
	}

	response.JSON(c.Writer, http.StatusOK, PaymentMethodResponse{
		Brand:    string(pm.Card.Brand),
		Last4:    pm.Card.Last4,
		ExpMonth: pm.Card.ExpMonth,
		ExpYear:  pm.Card.ExpYear,
	})
}

// InvoiceResponse is a simplified invoice returned by GET /billing/invoices.
type InvoiceResponse struct {
	ID               string `json:"id"`
	AmountDue        int64  `json:"amountDue"`
	AmountPaid       int64  `json:"amountPaid"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	HostedInvoiceURL string `json:"hostedInvoiceUrl"`
	InvoicePDF       string `json:"invoicePdf"`
	CreatedAt        string `json:"createdAt"`
}

// GetInvoices returns the last 24 invoices for the authenticated user.
// GET /billing/invoices
func (h *Handler) GetInvoices(c *gin.Context) {
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
		if isNotFound(err) {
			response.JSON(c.Writer, http.StatusOK, []InvoiceResponse{})
			return
		}
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	invoices, err := h.svc.ListInvoices(c.Request.Context(), sub.StripeCustomerID, 24)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	out := make([]InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, InvoiceResponse{
			ID:               inv.ID,
			AmountDue:        inv.AmountDue,
			AmountPaid:       inv.AmountPaid,
			Currency:         string(inv.Currency),
			Status:           string(inv.Status),
			HostedInvoiceURL: inv.HostedInvoiceURL,
			InvoicePDF:       inv.InvoicePDF,
			CreatedAt:        time.Unix(inv.Created, 0).UTC().Format(time.RFC3339),
		})
	}

	response.JSON(c.Writer, http.StatusOK, out)
}

// Cancel sets CancelAtPeriodEnd=true on the Stripe subscription.
// POST /billing/cancel
func (h *Handler) Cancel(c *gin.Context) {
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
		if isNotFound(err) {
			response.JSON(c.Writer, http.StatusOK, nil)
			return
		}
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	if err := h.svc.CancelSubscription(c.Request.Context(), sub.StripeID); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	cancelAtEnd := true
	sub.CancelAtCurrentPeriodEnd = &cancelAtEnd
	if err := h.subSvc.Update(c.Request.Context(), sub); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusOK, nil)
}

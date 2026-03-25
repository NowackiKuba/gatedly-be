package subscription

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

type CreateRequest struct {
	UserID                 uuid.UUID              `json:"userId"`
	PacketID               uuid.UUID              `json:"packetId"`
	Tier                   domain.BillingTier     `json:"tier"`
	Amount                 int                    `json:"amount"`
	Currency               string                 `json:"currency"`
	Interval               domain.BillingInterval `json:"interval"`
	Status                 string                 `json:"status"`
	IsTrial                bool                   `json:"isTrial"`
	TrialStartsAt          *time.Time             `json:"trialStartsAt"`
	TrialEndsAt            *time.Time             `json:"trialEndsAt"`
	CurrentPeriodStartedAt time.Time              `json:"currentPeriodStartedAt"`
	CurrentPeriodEndsAt    time.Time              `json:"currentPeriodEndsAt"`
	StripeID               string                 `json:"stripeId"`
	StripeCustomerID       string                 `json:"stripeCustomerId"`
	StripeMetadata         domain.JSONMap         `json:"stripeMetadata"`
}

type UpdateRequest struct {
	PacketID                 *uuid.UUID              `json:"packetId"`
	Tier                     *domain.BillingTier     `json:"tier"`
	Amount                   *int                    `json:"amount"`
	Currency                 *string                 `json:"currency"`
	Interval                 *domain.BillingInterval `json:"interval"`
	Status                   *string                 `json:"status"`
	IsTrial                  *bool                   `json:"isTrial"`
	TrialStartsAt            *time.Time              `json:"trialStartsAt"`
	TrialEndsAt              *time.Time              `json:"trialEndsAt"`
	CurrentPeriodStartedAt   *time.Time              `json:"currentPeriodStartedAt"`
	CurrentPeriodEndsAt      *time.Time              `json:"currentPeriodEndsAt"`
	CancelAtCurrentPeriodEnd *bool                   `json:"cancelAtCurrentPeriodEnd"`
	CanceledAt               *time.Time              `json:"canceledAt"`
	CancellationReason       *string                 `json:"cancellationReason"`
	EvaluationsUsed          *int                    `json:"evaluationsUsed"`
	EvaluationsResetAt       *time.Time              `json:"evaluationsResetAt"`
	StripeCustomerID         *string                 `json:"stripeCustomerId"`
	StripeMetadata           domain.JSONMap          `json:"stripeMetadata"`
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func (h *Handler) Create(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	var req CreateRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if req.StripeID == "" {
		response.Error(c.Writer, response.BadRequest("stripeId is required"))
		c.Abort()
		return
	}

	s := &domain.Subscription{
		UserID:                 req.UserID,
		PacketID:               req.PacketID,
		Tier:                   req.Tier,
		Amount:                 req.Amount,
		Currency:               req.Currency,
		Interval:               req.Interval,
		Status:                 req.Status,
		IsTrial:                req.IsTrial,
		TrialStartsAt:          req.TrialStartsAt,
		TrialEndsAt:            req.TrialEndsAt,
		CurrentPeriodStartedAt: req.CurrentPeriodStartedAt,
		CurrentPeriodEndsAt:    req.CurrentPeriodEndsAt,
		StripeID:               req.StripeID,
		StripeCustomerID:       req.StripeCustomerID,
		StripeMetadata:         req.StripeMetadata,
	}
	if err := h.svc.Create(c.Request.Context(), s); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusCreated, s)
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid subscription id"))
		c.Abort()
		return
	}
	s, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, s)
}

func (h *Handler) GetMe(c *gin.Context) {
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
	s, err := h.svc.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, s)
}

func (h *Handler) List(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	filters := Filters{
		Limit:        parseIntDefault(c.Query("limit"), 20),
		Offset:       parseIntDefault(c.Query("offset"), 0),
		OrderBy:      c.DefaultQuery("orderBy", "asc"),
		OrderByField: c.DefaultQuery("orderByField", "id"),
	}

	page, err := h.svc.List(c.Request.Context(), filters)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, page)
}

func (h *Handler) Update(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid subscription id"))
		c.Abort()
		return
	}

	var req UpdateRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	if req.PacketID != nil {
		existing.PacketID = *req.PacketID
	}
	if req.Tier != nil {
		existing.Tier = *req.Tier
	}
	if req.Amount != nil {
		existing.Amount = *req.Amount
	}
	if req.Currency != nil {
		existing.Currency = *req.Currency
	}
	if req.Interval != nil {
		existing.Interval = *req.Interval
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.IsTrial != nil {
		existing.IsTrial = *req.IsTrial
	}
	if req.TrialStartsAt != nil {
		existing.TrialStartsAt = req.TrialStartsAt
	}
	if req.TrialEndsAt != nil {
		existing.TrialEndsAt = req.TrialEndsAt
	}
	if req.CurrentPeriodStartedAt != nil {
		existing.CurrentPeriodStartedAt = *req.CurrentPeriodStartedAt
	}
	if req.CurrentPeriodEndsAt != nil {
		existing.CurrentPeriodEndsAt = *req.CurrentPeriodEndsAt
	}
	if req.CancelAtCurrentPeriodEnd != nil {
		existing.CancelAtCurrentPeriodEnd = req.CancelAtCurrentPeriodEnd
	}
	if req.CanceledAt != nil {
		existing.CanceledAt = req.CanceledAt
	}
	if req.CancellationReason != nil {
		existing.CancellationReason = req.CancellationReason
	}
	if req.EvaluationsUsed != nil {
		existing.EvaluationsUsed = *req.EvaluationsUsed
	}
	if req.EvaluationsResetAt != nil {
		existing.EvaluationsResetAt = req.EvaluationsResetAt
	}
	if req.StripeCustomerID != nil {
		existing.StripeCustomerID = *req.StripeCustomerID
	}
	if req.StripeMetadata != nil {
		existing.StripeMetadata = req.StripeMetadata
	}

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, existing)
}

func (h *Handler) Delete(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid subscription id"))
		c.Abort()
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusNoContent, nil)
}

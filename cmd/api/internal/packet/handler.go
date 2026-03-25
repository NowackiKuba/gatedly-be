package packet

import (
	"net/http"

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
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	IsPopular       bool                   `json:"isPopular"`
	Tier            domain.BillingTier     `json:"tier"`
	Amount          int                    `json:"amount"`
	Currency        string                 `json:"currency"`
	Features        domain.JSONMap         `json:"features"`
	StripeProductID string                 `json:"stripeProductId"`
	StripePriceID   string                 `json:"stripePriceId"`
	Interval        domain.BillingInterval `json:"interval"`
	TrialDays       int                    `json:"trialDays"`
	Limits          domain.BillingLimits   `json:"limits"`
	StripeMetadata  domain.JSONMap         `json:"stripeMetadata"`
}

type UpdateRequest struct {
	Name            *string                 `json:"name"`
	Description     *string                 `json:"description"`
	IsPopular       *bool                   `json:"isPopular"`
	Tier            *domain.BillingTier     `json:"tier"`
	Amount          *int                    `json:"amount"`
	Currency        *string                 `json:"currency"`
	Features        domain.JSONMap          `json:"features"`
	StripeProductID *string                 `json:"stripeProductId"`
	StripePriceID   *string                 `json:"stripePriceId"`
	Interval        *domain.BillingInterval `json:"interval"`
	TrialDays       *int                    `json:"trialDays"`
	Limits          *domain.BillingLimits   `json:"limits"`
	StripeMetadata  domain.JSONMap          `json:"stripeMetadata"`
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
	if req.Name == "" {
		response.Error(c.Writer, response.BadRequest("name is required"))
		c.Abort()
		return
	}

	p := &domain.Packet{
		Name:            req.Name,
		Description:     req.Description,
		IsPopular:       req.IsPopular,
		Tier:            req.Tier,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Features:        req.Features,
		StripeProductID: req.StripeProductID,
		StripePriceID:   req.StripePriceID,
		Interval:        req.Interval,
		TrialDays:       req.TrialDays,
		Limits:          req.Limits,
		StripeMetadata:  req.StripeMetadata,
	}
	if err := h.svc.Create(c.Request.Context(), p); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusCreated, p)
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid packet id"))
		c.Abort()
		return
	}
	p, err := h.svc.GetById(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, p)
}

func (h *Handler) GetAll(c *gin.Context) {
	list, err := h.svc.GetAll(c.Request.Context())
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, list)
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
		response.Error(c.Writer, response.BadRequest("invalid packet id"))
		c.Abort()
		return
	}

	var req UpdateRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	existing, err := h.svc.GetById(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.IsPopular != nil {
		existing.IsPopular = *req.IsPopular
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
	if req.Features != nil {
		existing.Features = req.Features
	}
	if req.StripeProductID != nil {
		existing.StripeProductID = *req.StripeProductID
	}
	if req.StripePriceID != nil {
		existing.StripePriceID = *req.StripePriceID
	}
	if req.Interval != nil {
		existing.Interval = *req.Interval
	}
	if req.TrialDays != nil {
		existing.TrialDays = *req.TrialDays
	}
	if req.Limits != nil {
		existing.Limits = *req.Limits
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
		response.Error(c.Writer, response.BadRequest("invalid packet id"))
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

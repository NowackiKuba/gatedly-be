package experiments

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

type CreateRequest struct {
	FlagID            uuid.UUID                 `json:"flagId"`
	Name              string                    `json:"name"`
	Status            domain.ExperimentStatus   `json:"status"`
	TrafficPercentage *int                      `json:"trafficPercentage"`
	Variants          domain.ExperimentVariants `json:"variants"`
	MinimumSampleSize *int                      `json:"minimumSampleSize"`
	ScheduledAt       *time.Time                `json:"scheduledAt"`
}

type UpdateRequest struct {
	Name              *string                   `json:"name"`
	Status            *domain.ExperimentStatus  `json:"status"`
	TrafficPercentage *int                      `json:"trafficPercentage"`
	Variants          domain.ExperimentVariants `json:"variants"`
	MinimumSampleSize *int                      `json:"minimumSampleSize"`
	WinnerVariant     *string                   `json:"winnerVariant"`
	ScheduledAt       *time.Time                `json:"scheduledAt"`
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest

	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	e := &domain.Experiment{
		FlagID:      req.FlagID,
		Name:        req.Name,
		Status:      req.Status,
		Variants:    req.Variants,
		ScheduledAt: req.ScheduledAt,
	}

	if req.TrafficPercentage != nil {
		e.TrafficPercentage = *req.TrafficPercentage
	} else {
		e.TrafficPercentage = 100
	}

	if req.MinimumSampleSize != nil {
		e.MinimumSampleSize = req.MinimumSampleSize
	}

	if err := h.svc.Create(c.Request.Context(), e); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusCreated, e)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))

	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid id"))
		c.Abort()
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), id)

	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	var req UpdateRequest

	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.TrafficPercentage != nil {
		existing.TrafficPercentage = *req.TrafficPercentage
	}
	if req.Variants != nil {
		existing.Variants = req.Variants
	}
	if req.MinimumSampleSize != nil {
		existing.MinimumSampleSize = req.MinimumSampleSize
	}
	if req.WinnerVariant != nil {
		existing.WinnerVariant = req.WinnerVariant
	}
	if req.ScheduledAt != nil {
		existing.ScheduledAt = req.ScheduledAt
	}

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusOK, existing)
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))

	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid id"))
		c.Abort()
		return
	}

	e, err := h.svc.GetByID(c.Request.Context(), id)

	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusOK, e)
}

func (h *Handler) GetByFlagID(c *gin.Context) {
	flagIDParam := c.Query("flagId")

	if flagIDParam == "" {
		response.Error(c.Writer, response.BadRequest("flag id is required"))
		c.Abort()
		return
	}

	flagID, err := uuid.Parse(flagIDParam)

	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag id"))
		c.Abort()
		return
	}

	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 20)
	orderBy := c.DefaultQuery("orderBy", "desc")
	orderByField := c.DefaultQuery("orderByField", "createdAt")

	filters := Filters{
		Limit:        limit,
		Offset:       offset,
		OrderBy:      orderBy,
		OrderByField: orderByField,
	}

	page, err := h.svc.GetByFlagID(c.Request.Context(), filters, flagID)

	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusOK, page)
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

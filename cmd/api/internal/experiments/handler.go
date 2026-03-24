package experiments

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/response"
)

// FlexibleTime handles multiple timestamp formats
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleTime
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	if str == "" || str == "null" {
		ft.Time = time.Time{}
		return nil
	}

	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, str); err == nil {
		ft.Time = t
		return nil
	}

	// Try format without timezone (YYYY-MM-DDTHH:MM)
	if t, err := time.Parse("2006-01-02T15:04", str); err == nil {
		ft.Time = t
		return nil
	}

	// Try format with just date (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", str); err == nil {
		ft.Time = t
		return nil
	}

	return &time.ParseError{
		Layout: time.RFC3339,
		Value:  str,
	}
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

type CreateRequest struct {
	FlagID            uuid.UUID                 `json:"flagId"`
	EnvironmentID     uuid.UUID                 `json:"environmentId"`
	Name              string                    `json:"name"`
	Status            domain.ExperimentStatus   `json:"status"`
	TrafficPercentage *int                      `json:"trafficPercentage"`
	Variants          domain.ExperimentVariants `json:"variants"`
	MinimumSampleSize *int                      `json:"minimumSampleSize"`
	ScheduledAt       *FlexibleTime             `json:"scheduledAt"`
}

type UpdateRequest struct {
	Name              *string                   `json:"name"`
	Status            *domain.ExperimentStatus  `json:"status"`
	TrafficPercentage *int                      `json:"trafficPercentage"`
	Variants          domain.ExperimentVariants `json:"variants"`
	MinimumSampleSize *int                      `json:"minimumSampleSize"`
	WinnerVariant     *string                   `json:"winnerVariant"`
	ScheduledAt       *FlexibleTime             `json:"scheduledAt"`
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest

	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	if req.EnvironmentID == uuid.Nil {
		response.Error(c.Writer, response.BadRequest("environmentId is required"))
		c.Abort()
		return
	}

	e := &domain.Experiment{
		FlagID:        req.FlagID,
		EnvironmentID: req.EnvironmentID,
		Name:          req.Name,
		Status:        req.Status,
		Variants:      req.Variants,
	}

	if req.ScheduledAt != nil {
		e.ScheduledAt = &req.ScheduledAt.Time
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
		existing.ScheduledAt = &req.ScheduledAt.Time
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
	environmentIDParam := c.Query("environmentId")

	if flagIDParam == "" || environmentIDParam == "" {
		response.Error(c.Writer, response.BadRequest("flagId and environmentId are required"))
		c.Abort()
		return
	}

	flagID, err := uuid.Parse(flagIDParam)

	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag id"))
		c.Abort()
		return
	}

	environmentID, err := uuid.Parse(environmentIDParam)

	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid environment id"))
		c.Abort()
		return
	}

	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)
	orderBy := c.DefaultQuery("orderBy", "desc")
	orderByField := c.DefaultQuery("orderByField", "created_at")

	filters := Filters{
		Limit:        limit,
		Offset:       offset,
		OrderBy:      orderBy,
		OrderByField: orderByField,
	}

	page, err := h.svc.GetByFlagID(c.Request.Context(), filters, flagID, environmentID)

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

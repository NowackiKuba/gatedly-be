package experimentevent

import (
	"fmt"
	"net/http"
	"strconv"

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
	ExperimentID uuid.UUID                  `json:"experimentId"`
	UserID       string                     `json:"userId"`
	Variant      string                     `json:"variant"`
	EventType    domain.ExperimentEventType `json:"eventType"`
	Metadata     domain.JSONMap             `json:"metadata"`
}

type VariantEventBreakdown struct {
	Variant        string `json:"variant"`
	Impressions    int    `json:"impressions"`
	Conversions    int    `json:"conversions"`
	CustomEvents   int    `json:"customEvents"`
	ConversionRate int    `json:"conversionRate"`
	UniqueUsers    int    `json:"uniqueUsers"`
}

type SummaryResponse struct {
	ExperimentID     string                `json:"experimentId"`
	TotalImpressions int                   `json:"totalImpressions"`
	TotalConversions int                   `json:"totalConversions"`
	TotalCustom      int                   `json:"totalCustom"`
	ConversionRate   int                   `json:"conversionRate"`
	VariantBreakdown VariantEventBreakdown `json:"variantBreakdown"`
	UnqiueUsers      int                   `json:"uniqueUsers"`
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	if req.ExperimentID == uuid.Nil {
		response.Error(c.Writer, response.BadRequest("experimentId is required"))
		c.Abort()
		return
	}
	if req.UserID == "" {
		response.Error(c.Writer, response.BadRequest("userId is required"))
		c.Abort()
		return
	}
	if req.Variant == "" {
		response.Error(c.Writer, response.BadRequest("variant is required"))
		c.Abort()
		return
	}
	if req.EventType == "" {
		response.Error(c.Writer, response.BadRequest("eventType is required"))
		c.Abort()
		return
	}

	event := &domain.ExperimentEvent{
		ExperimentID: req.ExperimentID,
		UserID:       req.UserID,
		Variant:      req.Variant,
		EventType:    req.EventType,
		Metadata:     req.Metadata,
	}

	if err := h.svc.Create(c.Request.Context(), event); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusCreated, event)
}

func (h *Handler) GetByExperimentID(c *gin.Context) {
	experimentIDParam := c.Param("id")
	if experimentIDParam == "" {
		response.Error(c.Writer, response.BadRequest("experimentId is required"))
		c.Abort()
		return
	}

	experimentID, err := uuid.Parse(experimentIDParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid experimentId"))
		c.Abort()
		return
	}

	fmt.Printf("LIMIT: %s", c.Query("limit"))
	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)
	orderBy := c.DefaultQuery("orderBy", "DESC")
	orderByField := c.DefaultQuery("orderByField", "created_at")

	var eventType *domain.ExperimentEventType
	if et := c.Query("eventType"); et != "" {
		t := domain.ExperimentEventType(et)
		eventType = &t
	}

	filters := Filters{
		Limit:        limit,
		Offset:       offset,
		OrderBy:      orderBy,
		OrderByField: orderByField,
		EventType:    eventType,
	}

	page, err := h.svc.GetByExperimentID(c.Request.Context(), filters, experimentID)
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

func (h *Handler) GetExperimentEventsSummary(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))

	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid event id"))
		c.Abort()
		return
	}

	hasMore := true
	limit := 100
	offset := 0

	var events []domain.ExperimentEvent

	for hasMore {
		res, err := h.svc.GetByExperimentID(c.Request.Context(), Filters{
			Limit:        limit,
			Offset:       offset,
			OrderBy:      "desc",
			OrderByField: "created_at",
		}, id)

		if err != nil {
			response.Error(c.Writer, err)
			c.Abort()
			return
		}

		events = append(events, res.Data...)

		hasMore = res.Page.HasNextPage

		offset += limit
	}

	// aggregate counts across all fetched events
	variantMap := make(map[string]*VariantEventBreakdown)
	uniqueUsers := make(map[string]struct{})

	for _, e := range events {
		uniqueUsers[e.UserID] = struct{}{}

		if _, ok := variantMap[e.Variant]; !ok {
			variantMap[e.Variant] = &VariantEventBreakdown{Variant: e.Variant}
		}
		v := variantMap[e.Variant]

		variantUniqueKey := e.Variant + "|" + e.UserID
		_ = variantUniqueKey // tracked per-variant below via a separate set if needed

		switch e.EventType {
		case domain.ExperimentEventImpression:
			v.Impressions++
		case domain.ExperimentEventConversion:
			v.Conversions++
		case domain.ExperimentEventCustom:
			v.CustomEvents++
		}
	}

	totalImpressions := 0
	totalConversions := 0
	totalCustom := 0

	// build per-variant unique user sets
	variantUsers := make(map[string]map[string]struct{})
	for _, e := range events {
		if _, ok := variantUsers[e.Variant]; !ok {
			variantUsers[e.Variant] = make(map[string]struct{})
		}
		variantUsers[e.Variant][e.UserID] = struct{}{}
	}

	for variant, v := range variantMap {
		if v.Impressions > 0 {
			v.ConversionRate = (v.Conversions * 100) / v.Impressions
		}
		v.UniqueUsers = len(variantUsers[variant])
		totalImpressions += v.Impressions
		totalConversions += v.Conversions
		totalCustom += v.CustomEvents
	}

	// flatten variant breakdown (return first variant; extend to slice if needed)
	var breakdown VariantEventBreakdown
	for _, v := range variantMap {
		breakdown = *v
		break
	}

	conversionRate := 0
	if totalImpressions > 0 {
		conversionRate = (totalConversions * 100) / totalImpressions
	}

	result := SummaryResponse{
		ExperimentID:     id.String(),
		TotalImpressions: totalImpressions,
		TotalConversions: totalConversions,
		TotalCustom:      totalCustom,
		ConversionRate:   conversionRate,
		VariantBreakdown: breakdown,
		UnqiueUsers:      len(uniqueUsers),
	}

	response.JSON(c.Writer, http.StatusOK, result)
}

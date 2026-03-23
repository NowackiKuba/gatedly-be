package experimentevent

import (
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
	experimentIDParam := c.Query("experimentId")
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

	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)
	orderBy := c.DefaultQuery("orderBy", "desc")
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

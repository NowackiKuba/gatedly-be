package experimentevent

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
	ExperimentID uuid.UUID                  `json:"experimentId"`
	Variant      string                     `json:"variant"`
	EventType    domain.ExperimentEventType `json:"eventType"`
	Metadata     domain.JSONMap             `json:"metadata"`
}

func (h *Handler) Create(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	uID, err := uuid.Parse(userIDStr)

	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid user id"))
		c.Abort()
		return
	}

	var req CreateRequest

	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	event := &domain.ExperimentEvent{
		ExperimentID: req.ExperimentID,
		Variant:      req.Variant,
		EventType:    req.EventType,
		Metadata:     req.Metadata,
		UserID:       uID,
	}

	if err := h.svc.Create(c.Request.Context(), event); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusCreated, event)
}

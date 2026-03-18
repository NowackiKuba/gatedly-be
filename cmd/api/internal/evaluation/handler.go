package evaluation

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/analytics"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc *Service
	analyticsRollups *analytics.Service
}

func NewHandler(svc *Service, analyticsRollups *analytics.Service) *Handler {
	return &Handler{svc: svc, analyticsRollups: analyticsRollups}
}

// EvaluateRequest is the body for single-flag evaluation. EnvID comes from API key (context).
type EvaluateRequest struct {
	FlagKey    string         `json:"flagKey"`
	UserID     string         `json:"userId"`
	Attributes map[string]any `json:"attributes"`
}

// EvaluateBatchRequest is the body for batch evaluation. EnvID comes from API key (context).
type EvaluateBatchRequest struct {
	FlagKeys   []string       `json:"flagKeys"`
	UserID     string         `json:"userId"`
	Attributes map[string]any `json:"attributes"`
}

func (h *Handler) Evaluate(c *gin.Context) {
	envID, ok := middleware.EnvironmentIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing environment context"))
		c.Abort()
		return
	}

	ctx := c.Request.Context()
	now := time.Now().UTC()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var projectID uuid.UUID
	if h.analyticsRollups != nil {
		if pid, err := h.analyticsRollups.GetProjectIDByEnvironmentID(ctx, envID); err == nil {
			projectID = pid
		}
	}

	recordUsage := func(isError bool) {
		if h.analyticsRollups == nil || projectID == uuid.Nil {
			return
		}
		errorsDelta := 0
		if isError {
			errorsDelta = 1
		}
		_ = h.analyticsRollups.IncrementApiUsageDaily(ctx, projectID, date, 1, errorsDelta)
	}

	var req EvaluateRequest
	if err := response.Decode(c.Request, &req); err != nil {
		recordUsage(true)
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if req.FlagKey == "" {
		recordUsage(true)
		response.Error(c.Writer, response.BadRequest("flagKey is required"))
		c.Abort()
		return
	}

	result, err := h.svc.Evaluate(ctx, envID, req.FlagKey, req.UserID, req.Attributes)
	if err != nil {
		recordUsage(true)
		response.Error(c.Writer, response.NotFound(err.Error()))
		c.Abort()
		return
	}
	recordUsage(false)
	response.JSON(c.Writer, http.StatusOK, result)
}

func (h *Handler) EvaluateBatch(c *gin.Context) {
	envID, ok := middleware.EnvironmentIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing environment context"))
		c.Abort()
		return
	}

	ctx := c.Request.Context()
	now := time.Now().UTC()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var projectID uuid.UUID
	if h.analyticsRollups != nil {
		if pid, err := h.analyticsRollups.GetProjectIDByEnvironmentID(ctx, envID); err == nil {
			projectID = pid
		}
	}

	recordUsage := func(isError bool) {
		if h.analyticsRollups == nil || projectID == uuid.Nil {
			return
		}
		errorsDelta := 0
		if isError {
			errorsDelta = 1
		}
		_ = h.analyticsRollups.IncrementApiUsageDaily(ctx, projectID, date, 1, errorsDelta)
	}

	var req EvaluateBatchRequest
	if err := response.Decode(c.Request, &req); err != nil {
		recordUsage(true)
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if len(req.FlagKeys) == 0 {
		recordUsage(true)
		response.Error(c.Writer, response.BadRequest("flagKeys is required and must not be empty"))
		c.Abort()
		return
	}

	results := h.svc.EvaluateBatch(ctx, envID, req.FlagKeys, req.UserID, req.Attributes)
	recordUsage(false)
	response.JSON(c.Writer, http.StatusOK, results)
}

package evaluation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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

	var req EvaluateRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if req.FlagKey == "" {
		response.Error(c.Writer, response.BadRequest("flagKey is required"))
		c.Abort()
		return
	}

	result, err := h.svc.Evaluate(c.Request.Context(), envID, req.FlagKey, req.UserID, req.Attributes)
	if err != nil {
		response.Error(c.Writer, response.NotFound(err.Error()))
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, result)
}

func (h *Handler) EvaluateBatch(c *gin.Context) {
	envID, ok := middleware.EnvironmentIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing environment context"))
		c.Abort()
		return
	}

	var req EvaluateBatchRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if len(req.FlagKeys) == 0 {
		response.Error(c.Writer, response.BadRequest("flagKeys is required and must not be empty"))
		c.Abort()
		return
	}

	results := h.svc.EvaluateBatch(c.Request.Context(), envID, req.FlagKeys, req.UserID, req.Attributes)
	response.JSON(c.Writer, http.StatusOK, results)
}

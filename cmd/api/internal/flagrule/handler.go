package flagrule

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
	FlagID        uuid.UUID       `json:"flagId"`
	EnvironmentID uuid.UUID       `json:"environmentId"`
	Enabled       bool            `json:"enabled"`
	RolloutPct    int             `json:"rolloutPct"`
	AllowList     domain.StringArray `json:"allowList"`
	DenyList      domain.StringArray `json:"denyList"`
	Conditions    domain.ConditionGroup `json:"conditions"`
	UpdatedBy     uuid.UUID       `json:"updatedBy"`
}

func (h *Handler) Create(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}
	updatedBy, err := uuid.Parse(userIDStr)
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

	rule := &domain.FlagRule{
		FlagID:        req.FlagID,
		EnvironmentID: req.EnvironmentID,
		Enabled:       req.Enabled,
		RolloutPct:    req.RolloutPct,
		AllowList:     req.AllowList,
		DenyList:      req.DenyList,
		Conditions:    req.Conditions,
		UpdatedBy:     updatedBy,
	}
	if rule.AllowList == nil {
		rule.AllowList = domain.StringArray{}
	}
	if rule.DenyList == nil {
		rule.DenyList = domain.StringArray{}
	}

	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusCreated, rule)
}

func (h *Handler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag rule id"))
		c.Abort()
		return
	}
	rule, err := h.svc.GetById(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, rule)
}

func (h *Handler) ListByFlag(c *gin.Context) {
	flagIDParam := c.Query("flagId")
	if flagIDParam == "" {
		response.Error(c.Writer, response.BadRequest("flagId is required"))
		c.Abort()
		return
	}
	flagID, err := uuid.Parse(flagIDParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag id"))
		c.Abort()
		return
	}
	list, err := h.svc.GetByFlagId(c.Request.Context(), flagID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if list == nil {
		list = []domain.FlagRule{}
	}
	response.JSON(c.Writer, http.StatusOK, list)
}

type UpdateRequest struct {
	Enabled    *bool                  `json:"enabled"`
	RolloutPct *int                   `json:"rolloutPct"`
	AllowList  *domain.StringArray    `json:"allowList"`
	DenyList   *domain.StringArray    `json:"denyList"`
	Conditions *domain.ConditionGroup `json:"conditions"`
}

func (h *Handler) Update(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}
	updatedBy, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid user id"))
		c.Abort()
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag rule id"))
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

	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.RolloutPct != nil {
		existing.RolloutPct = *req.RolloutPct
	}
	if req.AllowList != nil {
		existing.AllowList = *req.AllowList
	}
	if req.DenyList != nil {
		existing.DenyList = *req.DenyList
	}
	if req.Conditions != nil {
		existing.Conditions = *req.Conditions
	}
	existing.UpdatedBy = updatedBy

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, existing)
}

func (h *Handler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag rule id"))
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

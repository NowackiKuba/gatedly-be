package flagrule

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/analytics"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc Service
	analyticsRollups *analytics.Service
}

func NewHandler(svc Service, analyticsRollups *analytics.Service) *Handler {
	return &Handler{svc: svc, analyticsRollups: analyticsRollups}
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

	// Activity feed (non-blocking on failures).
	if h.analyticsRollups != nil {
		if projectID, err := h.analyticsRollups.GetProjectIDByEnvironmentID(c.Request.Context(), rule.EnvironmentID); err == nil {
			envID := rule.EnvironmentID
			flagID := rule.FlagID
			ruleID := rule.ID
			actorID := updatedBy
			evt := &domain.AnalyticsActivityEvent{
				ProjectID:     projectID,
				EnvironmentID: &envID,
				FlagID:        &flagID,
				RuleID:        &ruleID,
				EventType:     "RULE_CREATED",
				ActorID:       &actorID,
				OccurredAt:    time.Now().UTC(),
				Payload: domain.JSONMap{
					"enabled":    rule.Enabled,
					"rolloutPct": rule.RolloutPct,
					"allowList":  rule.AllowList,
					"denyList":   rule.DenyList,
					"conditions": rule.Conditions,
				},
			}
			_ = h.analyticsRollups.CreateActivityEvent(c.Request.Context(), evt)
		}
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
	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)
	orderBy := c.DefaultQuery("orderBy", "asc")
	orderByField := c.DefaultQuery("orderByField", "id")

	filters := Filters{
		Limit:        limit,
		Offset:       offset,
		OrderBy:      orderBy,
		OrderByField: orderByField,
	}

	page, err := h.svc.GetByFlagId(c.Request.Context(), filters, flagID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, page)
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
	if existing == nil {
		response.Error(c.Writer, response.NotFound("flag rule not found"))
		c.Abort()
		return
	}

	// Capture old values for PATCH activity payload.
	oldEnabled := existing.Enabled
	oldRolloutPct := existing.RolloutPct
	oldAllowList := existing.AllowList
	oldDenyList := existing.DenyList
	oldConditions := existing.Conditions

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

	// Activity feed (non-blocking on failures). Create one event per changed field.
	if h.analyticsRollups != nil {
		if projectID, err := h.analyticsRollups.GetProjectIDByEnvironmentID(c.Request.Context(), existing.EnvironmentID); err == nil {
			envID := existing.EnvironmentID
			flagID := existing.FlagID
			ruleID := existing.ID
			actorID := updatedBy

			now := time.Now().UTC()
			createEvt := func(eventType string, payload domain.JSONMap) {
				evt := &domain.AnalyticsActivityEvent{
					ProjectID:     projectID,
					EnvironmentID: &envID,
					FlagID:        &flagID,
					RuleID:        &ruleID,
					EventType:     eventType,
					ActorID:       &actorID,
					OccurredAt:    now,
					Payload:       payload,
				}
				_ = h.analyticsRollups.CreateActivityEvent(c.Request.Context(), evt)
			}

			if req.Enabled != nil {
				eventType := "RULE_ENABLED"
				if !existing.Enabled {
					eventType = "RULE_DISABLED"
				}
				createEvt(eventType, domain.JSONMap{
					"field": "enabled",
					"old":   oldEnabled,
					"new":   existing.Enabled,
				})
			}
			if req.RolloutPct != nil {
				createEvt("ROLLOUT_CHANGED", domain.JSONMap{
					"field": "rolloutPct",
					"old":   oldRolloutPct,
					"new":   existing.RolloutPct,
				})
			}
			if req.AllowList != nil {
				createEvt("ALLOW_LIST_UPDATED", domain.JSONMap{
					"field": "allowList",
					"old":   oldAllowList,
					"new":   existing.AllowList,
				})
			}
			if req.DenyList != nil {
				createEvt("DENY_LIST_UPDATED", domain.JSONMap{
					"field": "denyList",
					"old":   oldDenyList,
					"new":   existing.DenyList,
				})
			}
			if req.Conditions != nil {
				createEvt("CONDITIONS_UPDATED", domain.JSONMap{
					"field":      "conditions",
					"old":        oldConditions,
					"new":        existing.Conditions,
				})
			}
		}
	}

	response.JSON(c.Writer, http.StatusOK, existing)
}

func (h *Handler) Delete(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}
	actorID, err := uuid.Parse(userIDStr)
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

	existing, err := h.svc.GetById(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	if existing == nil {
		response.Error(c.Writer, response.NotFound("flag rule not found"))
		c.Abort()
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	// Activity feed (non-blocking on failures).
	if h.analyticsRollups != nil {
		if projectID, err := h.analyticsRollups.GetProjectIDByEnvironmentID(c.Request.Context(), existing.EnvironmentID); err == nil {
			envID := existing.EnvironmentID
			flagID := existing.FlagID
			ruleID := existing.ID
			evt := &domain.AnalyticsActivityEvent{
				ProjectID:     projectID,
				EnvironmentID: &envID,
				FlagID:        &flagID,
				RuleID:        &ruleID,
				EventType:     "RULE_DELETED",
				ActorID:       &actorID,
				OccurredAt:    time.Now().UTC(),
				Payload:       domain.JSONMap{},
			}
			_ = h.analyticsRollups.CreateActivityEvent(c.Request.Context(), evt)
		}
	}

	response.JSON(c.Writer, http.StatusNoContent, nil)
}

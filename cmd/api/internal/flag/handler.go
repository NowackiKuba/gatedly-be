package flag

import (
	"net/http"
	"strconv"

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
	ProjectID   uuid.UUID `json:"projectId"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
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

	f := &domain.Flag{
		ProjectID:   req.ProjectID,
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := h.svc.Create(c.Request.Context(), f); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusCreated, f)
}

func (h *Handler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag id"))
		c.Abort()
		return
	}
	f, err := h.svc.GetById(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, f)
}

func (h *Handler) GetByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Error(c.Writer, response.BadRequest("key is required"))
		c.Abort()
		return
	}
	f, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, f)
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

func (h *Handler) ListByProject(c *gin.Context) {
	projectIDParam := c.Query("projectId")
	if projectIDParam == "" {
		response.Error(c.Writer, response.BadRequest("projectId is required"))
		c.Abort()
		return
	}
	projectID, err := uuid.Parse(projectIDParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid project id"))
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

	page, err := h.svc.GetByProjectId(c.Request.Context(), filters, projectID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, page)
}

type UpdateRequest struct {
	Key         *string `json:"key"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *Handler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid flag id"))
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

	if req.Key != nil {
		existing.Key = *req.Key
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}

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
		response.Error(c.Writer, response.BadRequest("invalid flag id"))
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

package environment

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
	ProjectID uuid.UUID `json:"projectId"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Color     string    `json:"color"`
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

	env := &domain.Environment{
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Slug:      req.Slug,
		Color:     req.Color,
	}
	if env.Color == "" {
		env.Color = "#6366f1"
	}

	if err := h.svc.Create(c.Request.Context(), env); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusCreated, env)
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
	orderByField := c.DefaultQuery("orderByField", "created_at")

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

func (h *Handler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid environment id"))
		c.Abort()
		return
	}
	env, err := h.svc.GetById(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, env)
}

type UpdateRequest struct {
	Name  *string `json:"name"`
	Slug  *string `json:"slug"`
	Color *string `json:"color"`
}

func (h *Handler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid environment id"))
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

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Slug != nil {
		existing.Slug = *req.Slug
	}
	if req.Color != nil {
		existing.Color = *req.Color
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
		response.Error(c.Writer, response.BadRequest("invalid environment id"))
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

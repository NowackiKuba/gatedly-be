package apikey

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// CreateRequest is the body for POST /api-keys.
type CreateRequest struct {
	EnvironmentID uuid.UUID `json:"environmentId"`
	Name          string   `json:"name"`
}

// CreateResponse includes the plaintext key only on create (never stored or returned again).
type CreateResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Prefix        string    `json:"prefix"`
	EnvironmentID uuid.UUID `json:"environmentId"`
	CreatedAt     string    `json:"createdAt"`
	Key           string    `json:"key"` // plaintext — only shown once
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
	if req.Name == "" {
		response.Error(c.Writer, response.BadRequest("name is required"))
		c.Abort()
		return
	}

	k, plainKey, err := h.svc.Generate(c.Request.Context(), req.EnvironmentID, req.Name)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	resp := CreateResponse{
		ID:            k.ID,
		Name:          k.Name,
		Prefix:        k.Prefix,
		EnvironmentID: k.EnvironmentID,
		CreatedAt:     k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Key:           plainKey,
	}
	response.JSON(c.Writer, http.StatusCreated, resp)
}

func (h *Handler) List(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	envIDParam := c.Query("environmentId")
	if envIDParam == "" {
		response.Error(c.Writer, response.BadRequest("environmentId is required"))
		c.Abort()
		return
	}
	environmentID, err := uuid.Parse(envIDParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid environment id"))
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

	page, err := h.svc.List(c.Request.Context(), filters, environmentID)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, page)
}

func (h *Handler) Delete(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid api key id"))
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

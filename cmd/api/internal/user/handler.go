package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/pkg/response"
)

// Handler handles user HTTP requests.
type Handler struct {
	svc *Service
}

// NewHandler returns a new user handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetMe returns the current user's profile (requires auth).
func (h *Handler) GetMe(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid user id"))
		c.Abort()
		return
	}
	u, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, u)
}

// UpdateMeRequest is the body for PATCH /me.
type UpdateMeRequest struct {
	Name string `json:"name"`
}

// UpdateMe updates the current user's profile (requires auth).
func (h *Handler) UpdateMe(c *gin.Context) {
	userIDStr, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		response.Error(c.Writer, response.Unauthorized("missing user context"))
		c.Abort()
		return
	}
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid user id"))
		c.Abort()
		return
	}
	var req UpdateMeRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	u, err := h.svc.UpdateProfile(c.Request.Context(), id, req.Name)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, u)
}

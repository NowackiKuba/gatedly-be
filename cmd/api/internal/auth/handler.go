package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"toggly.com/m/pkg/response"
)

// Handler handles auth HTTP requests.
type Handler struct {
	svc *Service
}

// NewHandler returns a new auth handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRequest is the body for POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// Register creates a new user and returns tokens.
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	resp, err := h.svc.Register(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusCreated, resp)
}

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates and returns tokens.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	resp, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, resp)
}

// RefreshRequest is the body for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// Refresh returns new tokens using a valid refresh token.
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	resp, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}
	response.JSON(c.Writer, http.StatusOK, resp)
}

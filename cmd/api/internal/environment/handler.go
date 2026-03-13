package environment

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

type CreateRequest struct {
	ProjectID uuid.UUID `json:"projectId" gorm:"type:uuid;not null;uniqueIndex:idx_env_project_slug"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Slug      string    `json:"slug" gorm:"size:255;not null;uniqueIndex:idx_env_project_slug"`
	Color     string    `json:"color" gorm:"size:7;not null;default:#6366f1"`
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest

	if err := response.Decode(c.Request, &req); err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	env := domain.Environment{
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Slug:      req.Slug,
		Color:     req.Color,
	}

	err := h.svc.Create(c.Request.Context(), &env)

	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusCreated, nil)

}

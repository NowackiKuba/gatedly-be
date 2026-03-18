package analytics

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func parseRangeDays(v string) (int, error) {
	if v == "" {
		return 14, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	if n != 7 && n != 14 && n != 30 {
		return 0, nil
	}
	return n, nil
}

// GetProjectAnalytics serves GET /api/v1/projects/:id/analytics?rangeDays=7|14|30
func (h *Handler) GetProjectAnalytics(c *gin.Context) {
	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c.Writer, response.BadRequest("invalid project id"))
		c.Abort()
		return
	}

	rangeDays, err := strconv.Atoi(c.Query("rangeDays"))
	if err != nil {
		// allow empty → default below
		rangeDays, _ = parseRangeDays(c.Query("rangeDays"))
	}
	if rangeDays == 0 {
		rangeDays, _ = parseRangeDays("")
	}
	if rangeDays != 7 && rangeDays != 14 && rangeDays != 30 {
		response.Error(c.Writer, response.BadRequest("rangeDays must be 7, 14, or 30"))
		c.Abort()
		return
	}

	// Auth already populated userID in context, but this endpoint only needs Auth middleware,
	// not the user ID itself. Keeping the dependency in case we want actor-based filtering later.
	_, _ = middleware.UserIDFromContext(c.Request.Context())

	resp, err := h.svc.GetProjectAnalytics(c.Request.Context(), projectID, rangeDays)
	if err != nil {
		response.Error(c.Writer, err)
		c.Abort()
		return
	}

	response.JSON(c.Writer, http.StatusOK, resp)
}


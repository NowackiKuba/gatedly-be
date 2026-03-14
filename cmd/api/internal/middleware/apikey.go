package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
)

const environmentIDContextKey contextKey = "environmentID"

// EnvironmentIDFromContext returns the environment ID stored by APIKeyAuth middleware.
func EnvironmentIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(environmentIDContextKey)
	if v == nil {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// APIKeyVerifier is implemented by apikey.Service. Defined here to avoid import cycle.
type APIKeyVerifier interface {
	Verify(ctx context.Context, rawKey string) (*domain.APIKey, error)
}

// APIKeyAuth reads X-API-Key header, verifies it via the verifier, and injects environmentID into context.
// On failure returns 401 with { "status": 401, "message": "invalid api key" }.
func APIKeyAuth(verifier APIKeyVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawKey := c.GetHeader("X-API-Key")
		if rawKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "invalid api key"})
			c.Abort()
			return
		}

		k, err := verifier.Verify(c.Request.Context(), rawKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "invalid api key"})
			c.Abort()
			return
		}
		if k == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "invalid api key"})
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), environmentIDContextKey, k.EnvironmentID))
		c.Next()
	}
}

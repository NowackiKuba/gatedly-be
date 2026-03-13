package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// UserIDFromContext returns the user ID string stored in the request context by Auth middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(userIDContextKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// responseWriter wraps gin.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	gin.ResponseWriter
	status  int
	written int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	w.written += n
	return n, err
}

// Logger logs method, path, status, duration and optional bytes written via slog.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		wr := &responseWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = wr

		c.Next()

		latency := time.Since(start)
		log.Info("request",
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", wr.status),
			slog.Duration("latency", latency),
			slog.String("client_ip", clientIP),
		)
	}
}

// Recoverer catches panics, logs them with slog, and returns 500.
func Recoverer(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					slog.Any("panic", err),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

// Claims holds JWT standard and custom claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID    string `json:"userId"`
	TokenType string `json:"tokenType"` // "access" or "refresh"; Auth middleware accepts only "access"
}

// Auth validates the Bearer JWT and injects userID into context. Returns 401 on missing/invalid token.
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || len(auth) < 8 || auth[:7] != "Bearer " {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenStr := auth[7:]

		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid || claims.UserID == "" || claims.TokenType == "refresh" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), userIDContextKey, claims.UserID))
		c.Next()
	}
}

// CORS sets Access-Control-Allow-* headers and responds 204 to OPTIONS preflight.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

package server

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/auth"
	"toggly.com/m/cmd/api/internal/config"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/cmd/api/internal/user"
	"toggly.com/m/pkg/logger"
)

// Init creates the router, wires user and auth modules, and starts the HTTP server.
func Init(db *gorm.DB, cfg *config.Config) {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}
	slog := logger.New(env)

	router := gin.New()
	router.Use(middleware.Logger(slog), middleware.Recoverer(slog), middleware.CORS())

	// User repo/service/handler (shared with auth)
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	// Auth service/handler (uses user repo)
	authSvc := auth.NewService(userRepo, cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	authHandler := auth.NewHandler(authSvc)

	v1 := router.Group("/api/v1")
	auth.RegisterRoutes(v1.Group("/auth"), authHandler)

	usersGroup := v1.Group("/users")
	usersGroup.Use(middleware.Auth(cfg.JWT.Secret))
	user.RegisterRoutes(usersGroup, userHandler)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}
	slog.Info("server listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		log.Fatal(err)
	}
}

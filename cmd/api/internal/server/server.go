package server

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/analytics"
	"toggly.com/m/cmd/api/internal/apikey"
	"toggly.com/m/cmd/api/internal/auth"
	"toggly.com/m/cmd/api/internal/config"
	"toggly.com/m/cmd/api/internal/environment"
	"toggly.com/m/cmd/api/internal/evaluation"
	"toggly.com/m/cmd/api/internal/experimentevent"
	"toggly.com/m/cmd/api/internal/experiments"
	"toggly.com/m/cmd/api/internal/flag"
	"toggly.com/m/cmd/api/internal/flagrule"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/cmd/api/internal/project"
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
	slog.Info("server init (api routes)", "version", "1")

	router := gin.New()
	// Disable redirects so CORS headers are always applied (redirects can bypass middleware).
	router.RedirectTrailingSlash = false
	// CORS must run first so headers are set on every response (including 401/404).
	router.Use(middleware.CORS(), middleware.Logger(slog), middleware.Recoverer(slog))

	// User repo/service/handler (shared with auth)
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	envRepo := environment.NewRepository(db)
	envSvc := environment.NewService(envRepo)
	envHandler := environment.NewHandler(envSvc)

	// Auth service/handler (uses user repo)
	authSvc := auth.NewService(userRepo, cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	authHandler := auth.NewHandler(authSvc)

	projRepo := project.NewRepository(db)
	projSvc := project.NewService(projRepo)
	projHandler := project.NewHandler(projSvc)

	analyticsSvc := analytics.NewService(db)
	analyticsHandler := analytics.NewHandler(analyticsSvc)

	flagRepo := flag.NewRepository(db)
	flagSvc := flag.NewService(flagRepo)
	flagHandler := flag.NewHandler(flagSvc)

	flagRuleRepo := flagrule.NewRepository(db)
	evalSvc := evaluation.New(flagRuleRepo, analyticsSvc)
	flagRuleSvc := flagrule.NewService(flagRuleRepo, evalSvc.InvalidateCache)
	flagRuleHandler := flagrule.NewHandler(flagRuleSvc, analyticsSvc)

	apiKeyRepo := apikey.NewRepository(db)
	apiKeySvc := apikey.NewService(apiKeyRepo)
	apiKeyHandler := apikey.NewHandler(apiKeySvc)

	experimentsRepo := experiments.NewRepository(db)
	experimentsSvc := experiments.NewService(experimentsRepo)
	experimentsHandler := experiments.NewHandler(experimentsSvc)

	experimentEventRepo := experimentevent.NewRepository(db)
	experimentEventSvc := experimentevent.NewService(experimentEventRepo)
	experimentEventHandler := experimentevent.NewHandler(experimentEventSvc)

	evalHandler := evaluation.NewHandler(evalSvc, analyticsSvc)

	// Health + auth + user routes (explicit paths to avoid group path issues)
	v1 := router.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)

	usersGroup := v1.Group("/users")
	usersGroup.Use(middleware.Auth(cfg.JWT.Secret))
	user.RegisterRoutes(usersGroup, userHandler)

	envGroup := v1.Group("/environments")
	envGroup.Use(middleware.Auth(cfg.JWT.Secret))
	environment.RegisterRoutes(envGroup, envHandler)

	projGroup := v1.Group("/projects")
	projGroup.Use(middleware.Auth(cfg.JWT.Secret))
	project.RegisterRoutes(projGroup, projHandler)
	// Analytics endpoint (protected by Auth middleware, same as other project routes).
	projGroup.GET("/:id/analytics", analyticsHandler.GetProjectAnalytics)

	flagGroup := v1.Group("/flags")
	flagGroup.Use(middleware.Auth(cfg.JWT.Secret))
	flag.RegisterRoutes(flagGroup, flagHandler)

	flagRuleGroup := v1.Group("/flag-rules")
	flagRuleGroup.Use(middleware.Auth(cfg.JWT.Secret))
	flagrule.RegisterRoutes(flagRuleGroup, flagRuleHandler)

	apiKeysGroup := v1.Group("/api-keys")
	apiKeysGroup.Use(middleware.Auth(cfg.JWT.Secret))
	apikey.RegisterRoutes(apiKeysGroup, apiKeyHandler)

	experimentGroup := v1.Group("/experiments")
	experimentGroup.Use(middleware.Auth(cfg.JWT.Secret))
	experiments.RegisterRoutes(experimentGroup, experimentsHandler)

	evalGroup := v1.Group("/evaluation")
	evalGroup.Use(middleware.APIKeyAuth(apiKeySvc))
	evaluation.RegisterRoutes(evalGroup, evalHandler)

	experimentEventGroup := v1.Group("/experiment-events")
	experimentEventGroup.Use(middleware.APIKeyAuth(apiKeySvc))
	experimentEventGroup.POST("", experimentEventHandler.Create)

	experimentEventAuthGroup := v1.Group("/experiment-events")
	experimentEventAuthGroup.Use(middleware.Auth(cfg.JWT.Secret))
	experimentEventAuthGroup.GET("/:id", experimentEventHandler.GetByExperimentID)
	experimentEventAuthGroup.GET("/:id/summary", experimentEventHandler.GetExperimentEventsSummary)
	//

	// Log all registered API routes (Gin's Routes() can be incomplete with groups)
	routes := []struct{ method, path string }{
		{"GET", "/api/v1/health"},
		{"POST", "/api/v1/auth/register"},
		{"POST", "/api/v1/auth/login"},
		{"POST", "/api/v1/auth/refresh"},
		{"GET", "/api/v1/users/me"},
		{"PATCH", "/api/v1/users/me"},
	}
	for _, r := range routes {
		slog.Info("route", "method", r.method, "path", r.path)
	}

	router.NoRoute(func(c *gin.Context) {
		slog.Warn("404", "method", c.Request.Method, "path", c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{"error": "not found", "path": c.Request.URL.Path})
	})

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

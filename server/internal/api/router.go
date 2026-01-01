package api

import (
	"net/http"

	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"stackyn/server/internal/infra"
	"stackyn/server/internal/services"
)

// Router sets up the HTTP router with all routes and middleware
func Router(logger *zap.Logger, config *infra.Config, pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	// CORS middleware - allow all origins for development
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggingMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60))

	// Initialize log services (nil for now - wire up when ready)
	// TODO: Initialize log persistence and container log services
	var logPersistence LogPersistenceService
	var containerLogs ContainerLogService
	
	// Initialize plan enforcement service
	planEnforcement := services.NewPlanEnforcementService(logger)
	
	// Initialize billing service
	billingService := services.NewBillingService(logger)
	
	// Initialize constraints service (MVP constraints)
	maxBuildTimeMinutes := 15 // MVP: 15 minute max build time
	constraintsService := services.NewConstraintsService(logger, maxBuildTimeMinutes)
	
	// Initialize handlers
	handlers := NewHandlers(logger, logPersistence, containerLogs, planEnforcement, billingService, constraintsService)
	
	// Initialize email service
	emailService := services.NewEmailService(logger, config.Email.ResendAPIKey, config.Email.FromEmail)
	
	// Initialize repositories (use pool directly)
	otpRepo := NewOTPRepo(pool, logger)
	userRepo := NewUserRepo(pool, logger)
	
	// Initialize OTP service
	otpService := services.NewOTPService(logger, otpRepo, emailService)
	
	// Initialize JWT service
	jwtService := services.NewJWTService(config.JWT.Secret, logger)
	
	// Initialize auth handlers
	authHandlers := NewAuthHandlers(logger, otpService, jwtService, userRepo, otpRepo)

	// Health check
	r.Get("/health", handlers.HealthCheck)

	// Auth routes (no auth required)
	r.Route("/api/auth", func(r chi.Router) {
		// OTP authentication endpoints
		r.Post("/send-otp", authHandlers.SendOTP)
		r.Post("/verify-otp", authHandlers.VerifyOTP)
		r.Post("/login", authHandlers.Login)
		
		// Legacy Firebase endpoint (for compatibility)
		r.Post("/verify-token", handlers.VerifyToken)
		
		// Update user profile (requires auth)
		r.With(AuthMiddleware(jwtService, logger)).Post("/update-profile", authHandlers.UpdateUserProfile)
	})

	// User routes
	r.Route("/api/user", func(r chi.Router) {
		r.Get("/me", handlers.GetUserProfile)
	})

	// Apps routes - /api/apps (for listing)
	r.Get("/api/apps", handlers.ListApps)

	// Apps routes - /api/v1/apps (for CRUD operations)
	r.Route("/api/v1/apps", func(r chi.Router) {
		r.Get("/{id}", handlers.GetAppByID)
		r.Post("/", handlers.CreateApp)
		r.Delete("/{id}", handlers.DeleteApp)
		r.Post("/{id}/redeploy", handlers.RedeployApp)
		r.Get("/{id}/deployments", handlers.GetAppDeployments)
		r.Get("/{id}/env", handlers.GetEnvVars)
		r.Post("/{id}/env", handlers.CreateEnvVar)
		r.Delete("/{id}/env/{key}", handlers.DeleteEnvVar)
		
		// Log endpoints
		r.Get("/{id}/logs/build", handlers.GetBuildLogs)
		r.Get("/{id}/logs/runtime", handlers.GetRuntimeLogs)
		r.Get("/{id}/logs/runtime/stream", handlers.StreamRuntimeLogs)
	})

	// Deployments routes
	r.Route("/api/v1/deployments", func(r chi.Router) {
		r.Get("/{id}", handlers.GetDeploymentByID)
		r.Get("/{id}/logs", handlers.GetDeploymentLogs)
	})

	// Billing webhooks routes
	r.Route("/api/webhooks", func(r chi.Router) {
		r.Post("/lemon-squeezy", handlers.HandleLemonSqueezyWebhook)
	})

	return r
}

func loggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}


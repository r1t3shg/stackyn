package api

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"stackyn/server/internal/infra"
	"stackyn/server/internal/services"
)

// Wrapper types for adapter compatibility

// logPersistenceAdapter adapts services.LogPersistenceService to handlers.LogPersistenceService interface
type logPersistenceAdapter struct {
	service *services.LogPersistenceService
}

func (a *logPersistenceAdapter) PersistLog(ctx context.Context, entry LogEntry) error {
	// Convert handlers.LogEntry to services.LogEntry
	serviceEntry := services.LogEntry{
		AppID:        entry.AppID,
		BuildJobID:   entry.BuildJobID,
		DeploymentID: entry.DeploymentID,
		LogType:      string(entry.LogType),
		Timestamp:    entry.Timestamp,
		Content:      entry.Content,
		Size:         entry.Size,
	}
	return a.service.PersistLog(ctx, serviceEntry)
}

func (a *logPersistenceAdapter) PersistLogStream(ctx context.Context, entry LogEntry, reader io.Reader) error {
	// Convert LogEntry to services.LogEntry
	serviceEntry := services.LogEntry{
		AppID:        entry.AppID,
		BuildJobID:   entry.BuildJobID,
		DeploymentID: entry.DeploymentID,
		LogType:      entry.LogType, // LogEntry.LogType is already string
		Timestamp:    entry.Timestamp,
		Content:      entry.Content,
		Size:         entry.Size,
	}
	return a.service.PersistLogStream(ctx, serviceEntry, reader)
}

func (a *logPersistenceAdapter) GetLogs(ctx context.Context, appID string, logType LogType, limit int, offset int) ([]LogEntry, error) {
	serviceLogs, err := a.service.GetLogs(ctx, appID, string(logType), limit, offset)
	if err != nil {
		return nil, err
	}
	// Convert services.LogEntry to LogEntry
	logs := make([]LogEntry, len(serviceLogs))
	for i, serviceLog := range serviceLogs {
		logs[i] = LogEntry{
			AppID:        serviceLog.AppID,
			BuildJobID:   serviceLog.BuildJobID,
			DeploymentID: serviceLog.DeploymentID,
			LogType:      serviceLog.LogType, // Already string
			Timestamp:    serviceLog.Timestamp,
			Content:      serviceLog.Content,
			Size:         serviceLog.Size,
		}
	}
	return logs, nil
}

func (a *logPersistenceAdapter) GetLogsByDeploymentID(ctx context.Context, appID string, deploymentID string) (string, error) {
	return a.service.GetLogsByDeploymentID(ctx, appID, deploymentID)
}

func (a *logPersistenceAdapter) GetLogsByBuildJobID(ctx context.Context, appID string, buildJobID string) (string, error) {
	return a.service.GetLogsByBuildJobID(ctx, appID, buildJobID)
}

func (a *logPersistenceAdapter) GetLatestBuildLogs(ctx context.Context, appID string) (string, error) {
	return a.service.GetLatestBuildLogs(ctx, appID)
}

func (a *logPersistenceAdapter) DeleteOldLogs(ctx context.Context, appID string, before time.Time) error {
	return a.service.DeleteOldLogs(ctx, appID, before)
}

type planRepoInterfaceWrapper struct {
	repo *PlanRepo
}

func (w *planRepoInterfaceWrapper) GetPlanByID(ctx context.Context, planID string) (interface{}, error) {
	return w.repo.GetPlanByID(ctx, planID)
}

func (w *planRepoInterfaceWrapper) GetPlanByName(ctx context.Context, planName string) (interface{}, error) {
	return w.repo.GetPlanByName(ctx, planName)
}

func (w *planRepoInterfaceWrapper) GetDefaultPlan(ctx context.Context) (interface{}, error) {
	return w.repo.GetDefaultPlan(ctx)
}

type subscriptionRepoInterfaceWrapper struct {
	repo *SubscriptionRepo
}

func (w *subscriptionRepoInterfaceWrapper) GetSubscriptionByUserID(ctx context.Context, userID string) (interface{}, error) {
	return w.repo.GetSubscriptionByUserID(ctx, userID)
}

// Router sets up the HTTP router with all routes and middleware
func Router(logger *zap.Logger, config *infra.Config, pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	// CORS middleware - allow frontend origins
	// Use AllowOriginFunc to support staging subdomains dynamically
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			// Allow specific origins
			allowedOrigins := []string{
				"https://dev.stackyn.com",
				"https://console.dev.stackyn.com",
				"http://localhost:3000",
				"http://localhost:3001",
				"http://localhost:5173",
			}
			
			// Check exact matches first
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			
			// Allow any dev.stackyn.com subdomain
			if strings.HasSuffix(origin, ".dev.stackyn.com") || origin == "https://dev.stackyn.com" {
				return true
			}
			
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Link", "Content-Length"},
		AllowCredentials: true, // Allow credentials for JWT tokens
		MaxAge:           300,
		Debug:            false,
	}))

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggingMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(70 * time.Second)) // Slightly less than HTTP server timeout (75s)

	// Initialize log persistence service
	// Use /app/logs to match the volume mount in docker-compose (shared with deploy-worker)
	logStorageDir := "/app/logs"
	if err := os.MkdirAll(logStorageDir, 0755); err != nil {
		logger.Warn("Failed to create log storage directory", zap.Error(err), zap.String("dir", logStorageDir))
		// Continue without log persistence if directory creation fails
		logStorageDir = ""
	}
	
	var logPersistence LogPersistenceService
	if logStorageDir != "" {
		usePostgres := false // TODO: Make configurable
		maxStoragePerAppMB := int64(100) // Default: 100 MB per app
		// Create the service and wrap it to match the interface
		logPersistenceService := services.NewLogPersistenceService(logger, logStorageDir, usePostgres, maxStoragePerAppMB)
		logPersistence = &logPersistenceAdapter{service: logPersistenceService}
		logger.Info("Log persistence service initialized", zap.String("storage_dir", logStorageDir))
	} else {
		logger.Warn("Log persistence service not initialized - log storage directory unavailable")
	}
	
	var containerLogs ContainerLogService
	
	// Initialize plan enforcement service
	planEnforcement := services.NewPlanEnforcementService(logger)
	
	// Initialize billing service
	billingService := services.NewBillingService(logger)
	
	// Initialize constraints service (MVP constraints)
	maxBuildTimeMinutes := 15 // MVP: 15 minute max build time
	constraintsService := services.NewConstraintsService(logger, maxBuildTimeMinutes)
	
	// Initialize email service
	emailService := services.NewEmailService(logger, config.Email.ResendAPIKey, config.Email.FromEmail)
	
	// Initialize task enqueue service for triggering builds/deployments
	taskEnqueue, err := services.NewTaskEnqueueService(config.Redis.Addr, config.Redis.Password, logger, planEnforcement)
	if err != nil {
		logger.Error("Failed to initialize task enqueue service", zap.Error(err))
		// Continue without task enqueue - deployments will need to be triggered manually
		taskEnqueue = nil
	}
	
	// Initialize repositories (use pool directly)
	otpRepo := NewOTPRepo(pool, logger)
	userRepo := NewUserRepo(pool, logger)
	appRepo := NewAppRepo(pool, logger)
	
	// Initialize plan-related repositories
	planRepo := NewPlanRepo(pool, logger)
	subscriptionRepo := NewSubscriptionRepo(pool, logger)
	userPlanRepo := NewUserPlanRepo(pool, logger)
	
	// Create adapters for plan enforcement service
	// Use type assertion wrappers to match adapter interface
	planRepoAdapter := services.NewPlanRepoAdapter(&planRepoInterfaceWrapper{repo: planRepo}, logger)
	subscriptionRepoAdapter := services.NewSubscriptionRepoAdapter(&subscriptionRepoInterfaceWrapper{repo: subscriptionRepo}, logger)
	
	// Wire up plan enforcement service with repositories
	planEnforcement.SetRepositories(planRepoAdapter, subscriptionRepoAdapter, userPlanRepo)
	
	// Initialize OTP service
	otpService := services.NewOTPService(logger, otpRepo, emailService)
	
	// Initialize JWT service
	jwtService := services.NewJWTService(config.JWT.Secret, logger)
	
	// Initialize deployment repository
	deploymentRepo := NewDeploymentRepo(pool, logger)

	// Initialize environment variables repository
	envVarRepo := NewEnvVarRepo(pool, logger)

	// Initialize deployment service for verification (optional - can be nil)
	// Note: Deployment service requires Docker client, which may not be available in API server
	// For now, we'll pass nil and handlers will return service unavailable if called
	// TODO: Initialize deployment service if Docker is available in API server
	// deploymentSvc, err := services.NewDeploymentService(config.Docker.Host, logger, nil, config.Traefik.NetworkName)
	// var deploymentService handlers.DeploymentService
	// if err != nil {
	// 	logger.Warn("Failed to initialize deployment service in API server", zap.Error(err))
	// 	deploymentService = nil
	// } else {
	// 	deploymentService = deploymentSvc
	// }

	// Initialize handlers with appRepo, deploymentRepo, envVarRepo, userRepo, planRepo, userPlanRepo and task enqueue service
	// WebSocket removed - DB is single source of truth
	handlers := NewHandlers(logger, logPersistence, containerLogs, planEnforcement, billingService, constraintsService, appRepo, deploymentRepo, envVarRepo, userRepo, planRepo, userPlanRepo, taskEnqueue, nil, nil)

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
		
		// Password reset endpoints
		r.Post("/forgot-password", authHandlers.ForgotPassword)
		r.Post("/reset-password", authHandlers.ResetPassword)
		
		// Legacy Firebase endpoint (for compatibility)
		r.Post("/verify-token", handlers.VerifyToken)
		
		// Update user profile (requires auth)
		r.With(AuthMiddleware(jwtService, userRepo, logger)).Post("/update-profile", authHandlers.UpdateUserProfile)
	})

	// User routes
	r.Route("/api/user", func(r chi.Router) {
		r.Get("/me", handlers.GetUserProfile)
	})

	// Apps routes - /api/apps (for listing) - requires authentication
	r.With(AuthMiddleware(jwtService, userRepo, logger)).Get("/api/apps", handlers.ListApps)

	// Apps routes - /api/v1/apps (for CRUD operations) - requires authentication
	r.Route("/api/v1/apps", func(r chi.Router) {
		// Apply authentication middleware to all routes
		r.Use(AuthMiddleware(jwtService, userRepo, logger))
		
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
		
		// Verification endpoint
		r.Get("/{id}/verify", handlers.VerifyDeployment)
	})

	// Deployments routes - requires authentication
	r.Route("/api/v1/deployments", func(r chi.Router) {
		// Apply authentication middleware to all routes
		r.Use(AuthMiddleware(jwtService, userRepo, logger))
		
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
			// No WebSocket endpoints - removed for DB-based status updates

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


package services

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// PlanRepository interface for plan data access
type PlanRepository interface {
	GetPlanByID(ctx context.Context, planID string) (*PlanData, error)
	GetPlanByName(ctx context.Context, planName string) (*PlanData, error)
	GetDefaultPlan(ctx context.Context) (*PlanData, error)
}

// SubscriptionRepository interface for subscription data access
type SubscriptionRepository interface {
	GetSubscriptionByUserID(ctx context.Context, userID string) (*SubscriptionData, error)
}

// UserPlanRepository interface for getting user's plan_id
type UserPlanRepository interface {
	GetUserPlanID(ctx context.Context, userID string) (string, error)
}

// PlanData represents plan information for plan enforcement
type PlanData struct {
	ID             string
	Name           string
	MaxRAMMB       int
	MaxApps        int
	PriorityBuilds bool
}

// SubscriptionData represents subscription information
type SubscriptionData struct {
	Plan   string
	Status string
}

// PlanEnforcementService enforces plan-based limits
type PlanEnforcementService struct {
	logger           *zap.Logger
	planRepo         PlanRepository
	subscriptionRepo SubscriptionRepository
	userPlanRepo     UserPlanRepository

	// In-memory tracking for concurrent builds and RAM usage
	// In production, this should be in Redis or database
	buildCounts   map[string]int // userID -> concurrent build count
	ramUsage      map[string]int // userID -> RAM usage in MB
	buildCountsMu sync.RWMutex
	ramUsageMu    sync.RWMutex
}

// NewPlanEnforcementService creates a new plan enforcement service
func NewPlanEnforcementService(logger *zap.Logger) *PlanEnforcementService {
	return &PlanEnforcementService{
		logger:      logger,
		buildCounts: make(map[string]int),
		ramUsage:    make(map[string]int),
	}
}

// NewPlanEnforcementServiceWithRepos creates a new plan enforcement service with repositories
func NewPlanEnforcementServiceWithRepos(logger *zap.Logger, planRepo PlanRepository, subscriptionRepo SubscriptionRepository, userPlanRepo UserPlanRepository) *PlanEnforcementService {
	return &PlanEnforcementService{
		logger:           logger,
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		userPlanRepo:     userPlanRepo,
		buildCounts:      make(map[string]int),
		ramUsage:         make(map[string]int),
	}
}

// SetRepositories sets the repositories after service creation
func (s *PlanEnforcementService) SetRepositories(planRepo PlanRepository, subscriptionRepo SubscriptionRepository, userPlanRepo UserPlanRepository) {
	s.planRepo = planRepo
	s.subscriptionRepo = subscriptionRepo
	s.userPlanRepo = userPlanRepo
}

// PlanLimits represents the limits for a plan
type PlanLimits struct {
	MaxApps             int
	MaxRAMMB            int
	MaxConcurrentBuilds int
	QueuePriority       int // Higher number = higher priority
}

// GetPlanLimits gets the limits for a user's plan
func (s *PlanEnforcementService) GetPlanLimits(ctx context.Context, userID string) (*PlanLimits, error) {
	// If repositories are not set, fall back to default free plan limits
	if s.planRepo == nil {
		s.logger.Debug("Plan repository not set, using default free plan limits", zap.String("user_id", userID))
		return &PlanLimits{
			MaxApps:             3,
			MaxRAMMB:            1024, // 1 GB
			MaxConcurrentBuilds: 1,
			QueuePriority:       1, // Low priority
		}, nil
	}

	// Try to get plan from subscription first
	// Check both "active" and "trial" subscriptions (trial users should have plan limits enforced)
	var plan *PlanData
	if s.subscriptionRepo != nil {
		sub, err := s.subscriptionRepo.GetSubscriptionByUserID(ctx, userID)
		if err == nil && sub != nil && (sub.Status == "active" || sub.Status == "trial") {
			// Use the plan from the subscription (works for both "active" and "trial" status)
			plan, err = s.planRepo.GetPlanByName(ctx, sub.Plan)
			if err == nil && plan != nil {
				s.logger.Debug("Retrieved plan from subscription",
					zap.String("user_id", userID),
					zap.String("plan_name", plan.Name),
					zap.String("subscription_status", sub.Status),
				)
				return s.planDataToLimits(plan), nil
			}
		}
	}

	// If no subscription, try to get plan_id from user table
	if plan == nil && s.userPlanRepo != nil {
		planID, err := s.userPlanRepo.GetUserPlanID(ctx, userID)
		if err == nil && planID != "" {
			// Get plan by ID
			plan, err = s.planRepo.GetPlanByID(ctx, planID)
			if err == nil && plan != nil {
				s.logger.Debug("Retrieved plan from user plan_id",
					zap.String("user_id", userID),
					zap.String("plan_name", plan.Name),
				)
				return s.planDataToLimits(plan), nil
			}
		}
	}

	// If still no plan, get default free plan
	if plan == nil {
		defaultPlan, err := s.planRepo.GetDefaultPlan(ctx)
		if err == nil && defaultPlan != nil {
			plan = defaultPlan
			s.logger.Debug("Using default free plan",
				zap.String("user_id", userID),
				zap.String("plan_name", plan.Name),
			)
			return s.planDataToLimits(plan), nil
		}
	}

	// Fall back to hardcoded default free plan limits
	s.logger.Warn("Failed to retrieve plan, using hardcoded default limits", zap.String("user_id", userID))
	return &PlanLimits{
		MaxApps:             3,
		MaxRAMMB:            1024, // 1 GB
		MaxConcurrentBuilds: 1,
		QueuePriority:       1, // Low priority
	}, nil
}

// planDataToLimits converts PlanData to PlanLimits
func (s *PlanEnforcementService) planDataToLimits(plan *PlanData) *PlanLimits {
	queuePriority := 1 // Default low priority
	if plan.PriorityBuilds {
		queuePriority = 10 // High priority
	}

	maxApps := plan.MaxApps
	if maxApps == 0 {
		maxApps = 3 // Default
	}

	maxRAMMB := plan.MaxRAMMB
	if maxRAMMB == 0 {
		maxRAMMB = 1024 // Default 1 GB
	}

	return &PlanLimits{
		MaxApps:             maxApps,
		MaxRAMMB:            maxRAMMB,
		MaxConcurrentBuilds: 1, // Can be made configurable per plan later
		QueuePriority:       queuePriority,
	}
}

// CheckMaxApps checks if user can create another app (accounts for the new app being created)
func (s *PlanEnforcementService) CheckMaxApps(ctx context.Context, userID string, currentAppCount int) error {
	limits, err := s.GetPlanLimits(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get plan limits for max apps check",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get plan limits: %w", err)
	}

	s.logger.Info("Checking max apps limit",
		zap.String("user_id", userID),
		zap.Int("current_app_count", currentAppCount),
		zap.Int("max_apps", limits.MaxApps),
	)

	// Check if creating a new app would exceed the limit
	// currentAppCount is the count BEFORE creating the new app
	// So we check if currentAppCount + 1 > maxApps
	if currentAppCount+1 > limits.MaxApps {
		s.logger.Warn("Max apps limit exceeded",
			zap.String("user_id", userID),
			zap.Int("current_app_count", currentAppCount),
			zap.Int("max_apps", limits.MaxApps),
		)
		return &PlanLimitError{
			Limit:   "max_apps",
			Current: currentAppCount,
			Max:     limits.MaxApps,
			UserID:  userID,
			Message: fmt.Sprintf("You have reached the maximum number of apps (%d) for your plan. Please upgrade your plan to create more apps.", limits.MaxApps),
		}
	}

	s.logger.Debug("Max apps check passed",
		zap.String("user_id", userID),
		zap.Int("current_app_count", currentAppCount),
		zap.Int("max_apps", limits.MaxApps),
	)

	return nil
}

// CheckMaxRAM checks if user has enough RAM quota for requested RAM
func (s *PlanEnforcementService) CheckMaxRAM(ctx context.Context, userID string, requestedRAMMB int) error {
	limits, err := s.GetPlanLimits(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get plan limits: %w", err)
	}

	s.ramUsageMu.RLock()
	currentRAM := s.ramUsage[userID]
	s.ramUsageMu.RUnlock()

	if currentRAM+requestedRAMMB > limits.MaxRAMMB {
		return &PlanLimitError{
			Limit:     "max_ram",
			Current:   currentRAM,
			Requested: requestedRAMMB,
			Max:       limits.MaxRAMMB,
			UserID:    userID,
			Message:   fmt.Sprintf("Insufficient RAM quota. You are using %d MB of %d MB. Requested %d MB would exceed your plan limit. Please upgrade your plan or reduce resource usage.", currentRAM, limits.MaxRAMMB, requestedRAMMB),
		}
	}

	return nil
}

// CheckMaxConcurrentBuilds checks if user can start another build
func (s *PlanEnforcementService) CheckMaxConcurrentBuilds(ctx context.Context, userID string) error {
	limits, err := s.GetPlanLimits(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get plan limits: %w", err)
	}

	s.buildCountsMu.RLock()
	currentBuilds := s.buildCounts[userID]
	s.buildCountsMu.RUnlock()

	if currentBuilds >= limits.MaxConcurrentBuilds {
		return &PlanLimitError{
			Limit:   "max_concurrent_builds",
			Current: currentBuilds,
			Max:     limits.MaxConcurrentBuilds,
			UserID:  userID,
			Message: fmt.Sprintf("You have reached the maximum number of concurrent builds (%d) for your plan. Please wait for current builds to complete or upgrade your plan.", limits.MaxConcurrentBuilds),
		}
	}

	return nil
}

// GetQueuePriority gets the queue priority for a user based on their plan
func (s *PlanEnforcementService) GetQueuePriority(ctx context.Context, userID string) (int, error) {
	limits, err := s.GetPlanLimits(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get plan limits: %w", err)
	}

	return limits.QueuePriority, nil
}

// IncrementBuildCount increments the concurrent build count for a user
func (s *PlanEnforcementService) IncrementBuildCount(ctx context.Context, userID string) error {
	s.buildCountsMu.Lock()
	defer s.buildCountsMu.Unlock()

	s.buildCounts[userID]++

	s.logger.Debug("Incremented build count",
		zap.String("user_id", userID),
		zap.Int("current_builds", s.buildCounts[userID]),
	)

	return nil
}

// DecrementBuildCount decrements the concurrent build count for a user
func (s *PlanEnforcementService) DecrementBuildCount(ctx context.Context, userID string) error {
	s.buildCountsMu.Lock()
	defer s.buildCountsMu.Unlock()

	if s.buildCounts[userID] > 0 {
		s.buildCounts[userID]--
	}

	s.logger.Debug("Decremented build count",
		zap.String("user_id", userID),
		zap.Int("current_builds", s.buildCounts[userID]),
	)

	return nil
}

// IncrementRAMUsage increments the RAM usage for a user
func (s *PlanEnforcementService) IncrementRAMUsage(ctx context.Context, userID string, ramMB int) error {
	s.ramUsageMu.Lock()
	defer s.ramUsageMu.Unlock()

	s.ramUsage[userID] += ramMB

	s.logger.Debug("Incremented RAM usage",
		zap.String("user_id", userID),
		zap.Int("ram_mb", ramMB),
		zap.Int("total_ram_mb", s.ramUsage[userID]),
	)

	return nil
}

// DecrementRAMUsage decrements the RAM usage for a user
func (s *PlanEnforcementService) DecrementRAMUsage(ctx context.Context, userID string, ramMB int) error {
	s.ramUsageMu.Lock()
	defer s.ramUsageMu.Unlock()

	if s.ramUsage[userID] >= ramMB {
		s.ramUsage[userID] -= ramMB
	} else {
		s.ramUsage[userID] = 0
	}

	s.logger.Debug("Decremented RAM usage",
		zap.String("user_id", userID),
		zap.Int("ram_mb", ramMB),
		zap.Int("total_ram_mb", s.ramUsage[userID]),
	)

	return nil
}

// GetCurrentUsage gets the current usage for a user
func (s *PlanEnforcementService) GetCurrentUsage(ctx context.Context, userID string) (currentBuilds int, currentRAMMB int, err error) {
	s.buildCountsMu.RLock()
	currentBuilds = s.buildCounts[userID]
	s.buildCountsMu.RUnlock()

	s.ramUsageMu.RLock()
	currentRAMMB = s.ramUsage[userID]
	s.ramUsageMu.RUnlock()

	return currentBuilds, currentRAMMB, nil
}

// PlanLimitError represents a plan limit violation
type PlanLimitError struct {
	Limit     string
	Current   int
	Requested int // Only set for RAM checks
	Max       int
	UserID    string
	Message   string
}

func (e *PlanLimitError) Error() string {
	return e.Message
}

// IsPlanLimitError checks if an error is a PlanLimitError
func IsPlanLimitError(err error) bool {
	_, ok := err.(*PlanLimitError)
	return ok
}

// GetPlanLimitError extracts PlanLimitError from error
func GetPlanLimitError(err error) (*PlanLimitError, bool) {
	planErr, ok := err.(*PlanLimitError)
	return planErr, ok
}

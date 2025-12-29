package services

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// PlanEnforcementService enforces plan-based limits
type PlanEnforcementService struct {
	logger *zap.Logger
	// TODO: Add database repository when DB is connected
	// userRepo UserRepository
	// planRepo PlanRepository
	
	// In-memory tracking for concurrent builds and RAM usage
	// In production, this should be in Redis or database
	buildCounts   map[string]int           // userID -> concurrent build count
	ramUsage      map[string]int           // userID -> RAM usage in MB
	buildCountsMu sync.RWMutex
	ramUsageMu    sync.RWMutex
}

// NewPlanEnforcementService creates a new plan enforcement service
func NewPlanEnforcementService(logger *zap.Logger) *PlanEnforcementService {
	return &PlanEnforcementService{
		logger:     logger,
		buildCounts: make(map[string]int),
		ramUsage:    make(map[string]int),
	}
}

// PlanLimits represents the limits for a plan
type PlanLimits struct {
	MaxApps            int
	MaxRAMMB           int
	MaxConcurrentBuilds int
	QueuePriority      int // Higher number = higher priority
}

// GetPlanLimits gets the limits for a user's plan
// TODO: Fetch from database when DB is connected
func (s *PlanEnforcementService) GetPlanLimits(ctx context.Context, userID string) (*PlanLimits, error) {
	// TODO: Query database for user's plan
	// For now, return default limits
	// In production, this would be:
	// user, err := s.userRepo.GetByID(ctx, userID)
	// if err != nil {
	//     return nil, err
	// }
	// plan, err := s.planRepo.GetByID(ctx, user.PlanID)
	// if err != nil {
	//     return nil, err
	// }
	// return &PlanLimits{
	//     MaxApps: plan.MaxApps,
	//     MaxRAMMB: plan.MaxRAMMB,
	//     MaxConcurrentBuilds: plan.MaxConcurrentBuilds,
	//     QueuePriority: plan.QueuePriority,
	// }, nil

	// Default plan limits (free tier)
	return &PlanLimits{
		MaxApps:            3,
		MaxRAMMB:           1024, // 1 GB
		MaxConcurrentBuilds: 1,
		QueuePriority:      1, // Low priority
	}, nil
}

// CheckMaxApps checks if user has reached max apps limit
func (s *PlanEnforcementService) CheckMaxApps(ctx context.Context, userID string, currentAppCount int) error {
	limits, err := s.GetPlanLimits(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get plan limits: %w", err)
	}

	if currentAppCount >= limits.MaxApps {
		return &PlanLimitError{
			Limit:     "max_apps",
			Current:   currentAppCount,
			Max:       limits.MaxApps,
			UserID:    userID,
			Message:   fmt.Sprintf("You have reached the maximum number of apps (%d) for your plan. Please upgrade your plan to create more apps.", limits.MaxApps),
		}
	}

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
			Limit:     "max_concurrent_builds",
			Current:   currentBuilds,
			Max:       limits.MaxConcurrentBuilds,
			UserID:    userID,
			Message:   fmt.Sprintf("You have reached the maximum number of concurrent builds (%d) for your plan. Please wait for current builds to complete or upgrade your plan.", limits.MaxConcurrentBuilds),
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


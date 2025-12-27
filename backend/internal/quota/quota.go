// Package quota provides quota checking and enforcement for user plans.
// Quotas are enforced per user across all their apps, not per app.
package quota

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"mvp-be/internal/plans"
)

// Service handles quota validation and enforcement
type Service struct {
	db *sql.DB
}

// NewService creates a new quota service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// QuotaCheck represents the result of a quota check
type QuotaCheck struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// UserQuota represents a user's current quota usage
type UserQuota struct {
	PlanName     plans.PlanName `json:"plan_name"`
	Plan         plans.Plan     `json:"plan"`
	AppCount     int            `json:"app_count"`
	TotalRAMMB   int            `json:"total_ram_mb"`
	TotalDiskMB  int            `json:"total_disk_mb"`
}

// GetUserQuota retrieves a user's current quota usage
func (s *Service) GetUserQuota(ctx context.Context, userID string) (*UserQuota, error) {
	// Get user's plan
	var planNameStr string
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(plan, 'free') FROM users WHERE id = $1", userID).Scan(&planNameStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get user plan: %w", err)
	}

	planName := plans.PlanName(planNameStr)
	plan := plans.GetPlan(planName)

	// Count user's apps
	var appCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM apps WHERE user_id = $1", userID).Scan(&appCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count apps: %w", err)
	}

	// Calculate total RAM and disk usage across all apps
	// For now, we'll use default values per app since we don't track per-app resources yet
	// TODO: Track actual RAM/disk usage per app from Docker containers
	// For now, assume each app uses a default amount
	defaultRAMPerApp := 64  // MB per app (will be configurable per app later)
	defaultDiskPerApp := 256 // MB per app (will be configurable per app later)

	totalRAMMB := appCount * defaultRAMPerApp
	totalDiskMB := appCount * defaultDiskPerApp

	return &UserQuota{
		PlanName:    planName,
		Plan:        plan,
		AppCount:    appCount,
		TotalRAMMB:  totalRAMMB,
		TotalDiskMB: totalDiskMB,
	}, nil
}

// CheckAppCreation checks if a user can create a new app
func (s *Service) CheckAppCreation(ctx context.Context, userID string) (*QuotaCheck, error) {
	quota, err := s.GetUserQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check app count limit
	if quota.AppCount >= quota.Plan.MaxApps {
		return &QuotaCheck{
			Allowed: false,
			Reason:  fmt.Sprintf("App limit reached. Your %s plan allows up to %d app(s). Please upgrade your plan or delete an existing app.", quota.Plan.DisplayName, quota.Plan.MaxApps),
		}, nil
	}

	// Check RAM limit (with new app)
	newAppRAM := 64 // Default RAM per app
	if quota.TotalRAMMB+newAppRAM > quota.Plan.MaxRAMMB {
		return &QuotaCheck{
			Allowed: false,
			Reason:  fmt.Sprintf("RAM limit exceeded. Your %s plan allows %d MB total RAM. Current usage: %d MB. New app would require %d MB more.", quota.Plan.DisplayName, quota.Plan.MaxRAMMB, quota.TotalRAMMB, newAppRAM),
		}, nil
	}

	// Check disk limit (with new app)
	newAppDisk := 256 // Default disk per app
	if quota.TotalDiskMB+newAppDisk > quota.Plan.MaxDiskMB {
		return &QuotaCheck{
			Allowed: false,
			Reason:  fmt.Sprintf("Disk limit exceeded. Your %s plan allows %d MB total disk. Current usage: %d MB. New app would require %d MB more.", quota.Plan.DisplayName, quota.Plan.MaxDiskMB, quota.TotalDiskMB, newAppDisk),
		}, nil
	}

	return &QuotaCheck{Allowed: true}, nil
}

// CheckFeature checks if a user's plan supports a specific feature
func (s *Service) CheckFeature(ctx context.Context, userID string, feature string) (*QuotaCheck, error) {
	quota, err := s.GetUserQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	var hasFeature bool
	var featureName string

	switch feature {
	case "auto_deploy":
		hasFeature = quota.Plan.AutoDeploy
		featureName = "Auto Deploy"
	case "health_checks":
		hasFeature = quota.Plan.HealthChecks
		featureName = "Health Checks"
	case "logs":
		hasFeature = quota.Plan.Logs
		featureName = "Application Logs"
	case "zero_downtime":
		hasFeature = quota.Plan.ZeroDowntime
		featureName = "Zero-Downtime Deployments"
	case "workers":
		hasFeature = quota.Plan.Workers
		featureName = "Background Workers"
	case "priority_builds":
		hasFeature = quota.Plan.PriorityBuilds
		featureName = "Priority Builds"
	case "always_on":
		hasFeature = quota.Plan.AlwaysOn
		featureName = "Always-On Apps"
	default:
		return &QuotaCheck{
			Allowed: false,
			Reason:  fmt.Sprintf("Unknown feature: %s", feature),
		}, nil
	}

	if !hasFeature {
		return &QuotaCheck{
			Allowed: false,
			Reason:  fmt.Sprintf("%s is not available on your %s plan. Please upgrade to access this feature.", featureName, quota.Plan.DisplayName),
		}, nil
	}

	return &QuotaCheck{Allowed: true}, nil
}

// ValidatePlanName validates that a plan name is valid
func ValidatePlanName(planName string) error {
	if !plans.IsValidPlan(plans.PlanName(planName)) {
		return errors.New("invalid plan name")
	}
	return nil
}


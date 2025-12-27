// Package plans defines pricing plans and their limits/features.
// Plans are code-based configurations that define resource quotas and feature flags.
package plans

// PlanName represents a pricing plan identifier
type PlanName string

const (
	PlanFree    PlanName = "free"
	PlanStarter PlanName = "starter"
	PlanBuilder PlanName = "builder"
	PlanPro     PlanName = "pro"
)

// Plan defines a pricing plan with its limits and features
type Plan struct {
	Name        PlanName `json:"name"`
	DisplayName string    `json:"display_name"`
	Price       int       `json:"price"` // Price in cents per month

	// Resource Limits (per user, not per app)
	MaxRAMMB      int `json:"max_ram_mb"`      // Total RAM across all apps
	MaxDiskMB     int `json:"max_disk_mb"`     // Total disk across all apps
	MaxApps       int `json:"max_apps"`        // Maximum number of apps

	// Feature Flags
	AlwaysOn         bool `json:"always_on"`          // Apps stay running
	AutoDeploy      bool `json:"auto_deploy"`        // Automatic deployments on git push
	HealthChecks    bool `json:"health_checks"`       // Health check monitoring
	Logs            bool `json:"logs"`               // Access to application logs
	ZeroDowntime    bool `json:"zero_downtime"`      // Zero-downtime deployments
	Workers         bool `json:"workers"`            // Background workers support
	PriorityBuilds  bool `json:"priority_builds"`    // Priority in build queue
	ManualDeployOnly bool `json:"manual_deploy_only"` // Only manual deployments allowed
}

// Plans contains all available pricing plans
var Plans = map[PlanName]Plan{
	PlanFree: {
		Name:            PlanFree,
		DisplayName:     "Free",
		Price:           0, // $0/month
		MaxRAMMB:        128,
		MaxDiskMB:       512, // Minimal default
		MaxApps:         1,
		AlwaysOn:        false,
		AutoDeploy:      false,
		HealthChecks:    false,
		Logs:            false,
		ZeroDowntime:    false,
		Workers:         false,
		PriorityBuilds: false,
		ManualDeployOnly: true,
	},
	PlanStarter: {
		Name:            PlanStarter,
		DisplayName:     "Starter",
		Price:           500, // $5/month
		MaxRAMMB:        256,
		MaxDiskMB:       1024, // Higher than Free
		MaxApps:         1,
		AlwaysOn:        true,
		AutoDeploy:      false,
		HealthChecks:    false,
		Logs:            false,
		ZeroDowntime:    false,
		Workers:         false,
		PriorityBuilds: false,
		ManualDeployOnly: true,
	},
	PlanBuilder: {
		Name:            PlanBuilder,
		DisplayName:     "Builder",
		Price:           1500, // $15/month
		MaxRAMMB:        512,
		MaxDiskMB:       2048, // Medium
		MaxApps:         3,
		AlwaysOn:        true,
		AutoDeploy:      true,
		HealthChecks:    true,
		Logs:            true,
		ZeroDowntime:    false,
		Workers:         false,
		PriorityBuilds: false,
		ManualDeployOnly: false,
	},
	PlanPro: {
		Name:            PlanPro,
		DisplayName:     "Pro",
		Price:           2900, // $29/month
		MaxRAMMB:        1024, // 1GB
		MaxDiskMB:       5120, // Highest
		MaxApps:         5,
		AlwaysOn:        true,
		AutoDeploy:      true,
		HealthChecks:    true,
		Logs:            true,
		ZeroDowntime:    true,
		Workers:         true,
		PriorityBuilds:  true,
		ManualDeployOnly: false,
	},
}

// GetPlan returns a plan by name, or the Free plan if not found
func GetPlan(name PlanName) Plan {
	if plan, ok := Plans[name]; ok {
		return plan
	}
	return Plans[PlanFree] // Default to Free plan
}

// GetDefaultPlan returns the default plan (Free)
func GetDefaultPlan() Plan {
	return Plans[PlanFree]
}

// IsValidPlan checks if a plan name is valid
func IsValidPlan(name PlanName) bool {
	_, ok := Plans[name]
	return ok
}


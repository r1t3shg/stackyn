package services

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"
)

// PlanRepoAdapter adapts api.PlanRepo to services.PlanRepository interface
type PlanRepoAdapter struct {
	planRepo interface {
		GetPlanByID(ctx context.Context, planID string) (interface{}, error)
		GetPlanByName(ctx context.Context, planName string) (interface{}, error)
		GetDefaultPlan(ctx context.Context) (interface{}, error)
	}
	logger *zap.Logger
}

// NewPlanRepoAdapter creates a new plan repo adapter
func NewPlanRepoAdapter(planRepo interface {
	GetPlanByID(ctx context.Context, planID string) (interface{}, error)
	GetPlanByName(ctx context.Context, planName string) (interface{}, error)
	GetDefaultPlan(ctx context.Context) (interface{}, error)
}, logger *zap.Logger) PlanRepository {
	return &PlanRepoAdapter{
		planRepo: planRepo,
		logger:   logger,
	}
}

// GetPlanByID retrieves a plan by ID and converts to PlanData
func (a *PlanRepoAdapter) GetPlanByID(ctx context.Context, planID string) (*PlanData, error) {
	plan, err := a.planRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	return a.convertToPlanData(plan)
}

// GetPlanByName retrieves a plan by name and converts to PlanData
func (a *PlanRepoAdapter) GetPlanByName(ctx context.Context, planName string) (*PlanData, error) {
	plan, err := a.planRepo.GetPlanByName(ctx, planName)
	if err != nil {
		return nil, err
	}
	return a.convertToPlanData(plan)
}

// GetDefaultPlan retrieves the default plan and converts to PlanData
func (a *PlanRepoAdapter) GetDefaultPlan(ctx context.Context) (*PlanData, error) {
	plan, err := a.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return nil, err
	}
	return a.convertToPlanData(plan)
}

// convertToPlanData converts the plan interface to PlanData using reflection
func (a *PlanRepoAdapter) convertToPlanData(plan interface{}) (*PlanData, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan is nil")
	}

	v := reflect.ValueOf(plan)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("plan is not a struct: %T", plan)
	}

	planData := &PlanData{}

	// Extract fields using reflection
	if f := v.FieldByName("ID"); f.IsValid() && f.Kind() == reflect.String {
		planData.ID = f.String()
	}
	if f := v.FieldByName("Name"); f.IsValid() && f.Kind() == reflect.String {
		planData.Name = f.String()
	}
	if f := v.FieldByName("MaxRAMMB"); f.IsValid() && f.Kind() == reflect.Int {
		planData.MaxRAMMB = int(f.Int())
	}
	if f := v.FieldByName("MaxApps"); f.IsValid() && f.Kind() == reflect.Int {
		planData.MaxApps = int(f.Int())
	}
	if f := v.FieldByName("PriorityBuilds"); f.IsValid() && f.Kind() == reflect.Bool {
		planData.PriorityBuilds = f.Bool()
	}

	if planData.Name == "" {
		return nil, fmt.Errorf("failed to extract plan name from %T", plan)
	}

	return planData, nil
}

// SubscriptionRepoAdapter adapts api.SubscriptionRepo to services.SubscriptionRepository interface
type SubscriptionRepoAdapter struct {
	subRepo interface {
		GetSubscriptionByUserID(ctx context.Context, userID string) (interface{}, error)
	}
	logger *zap.Logger
}

// NewSubscriptionRepoAdapter creates a new subscription repo adapter
func NewSubscriptionRepoAdapter(subRepo interface {
	GetSubscriptionByUserID(ctx context.Context, userID string) (interface{}, error)
}, logger *zap.Logger) SubscriptionRepository {
	return &SubscriptionRepoAdapter{
		subRepo: subRepo,
		logger:  logger,
	}
}

// GetSubscriptionByUserID retrieves a subscription and converts to SubscriptionData
func (a *SubscriptionRepoAdapter) GetSubscriptionByUserID(ctx context.Context, userID string) (*SubscriptionData, error) {
	sub, err := a.subRepo.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return a.convertToSubscriptionData(sub)
}

// convertToSubscriptionData converts the subscription interface to SubscriptionData using reflection
func (a *SubscriptionRepoAdapter) convertToSubscriptionData(sub interface{}) (*SubscriptionData, error) {
	if sub == nil {
		return nil, fmt.Errorf("subscription is nil")
	}

	v := reflect.ValueOf(sub)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("subscription is not a struct: %T", sub)
	}

	subData := &SubscriptionData{}

	// Extract fields using reflection
	if f := v.FieldByName("Plan"); f.IsValid() && f.Kind() == reflect.String {
		subData.Plan = f.String()
	}
	if f := v.FieldByName("Status"); f.IsValid() && f.Kind() == reflect.String {
		subData.Status = f.String()
	}

	return subData, nil
}


package services

import (
	"context"

	"go.uber.org/zap"
)

// Service is the base interface for all services
type Service interface {
	// Start initializes and starts the service
	Start(ctx context.Context) error
	// Stop gracefully stops the service
	Stop(ctx context.Context) error
}

// BaseService provides common functionality for services
type BaseService struct {
	Logger *zap.Logger
}

// NewBaseService creates a new base service
func NewBaseService(logger *zap.Logger) *BaseService {
	return &BaseService{
		Logger: logger,
	}
}


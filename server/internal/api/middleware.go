package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// AuthMiddleware validates JWT tokens and adds user info to context
func AuthMiddleware(jwtService *services.JWTService, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Authorization header required"}`))
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Invalid authorization header format"}`))
				return
			}

			token := parts[1]

			// Validate backend JWT token
			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				logger.Warn("JWT token validation failed", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Invalid or expired token"}`))
				return
			}

			// Backend JWT token is valid - add user info to context
			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "user_email", claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireActiveBilling checks if a user has active billing (trial or active subscription)
// Returns error if billing is inactive
func RequireActiveBilling(user *User) error {
	if user == nil {
		return errors.New("user not found")
	}

	// Active billing: status is "active"
	if user.BillingStatus == "active" {
		return nil
	}

	// Trial billing: status is "trial" and trial hasn't expired
	if user.BillingStatus == "trial" {
		if user.TrialEndsAt == nil {
			// Trial without end date - treat as expired for safety
			return errors.New("trial has expired. Upgrade to continue")
		}
		if time.Now().Before(*user.TrialEndsAt) {
			return nil
		}
		// Trial expired
		return errors.New("your free trial has ended. Upgrade to continue")
	}

	// Billing is inactive (expired, cancelled, etc.)
	return errors.New("billing inactive. Upgrade to continue")
}

// BillingMiddleware enforces active billing for protected endpoints
// Must be used after AuthMiddleware (requires user_id in context)
func BillingMiddleware(userRepo *UserRepo, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by AuthMiddleware)
			userID, ok := r.Context().Value("user_id").(string)
			if !ok || userID == "" {
				logger.Error("BillingMiddleware: user_id not found in context")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "User not authenticated"})
				return
			}

			// Get user with billing info
			user, err := userRepo.GetUserByID(userID)
			if err != nil {
				logger.Error("BillingMiddleware: failed to get user", zap.Error(err), zap.String("user_id", userID))
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to verify billing status"})
				return
			}

			// Check billing status
			if err := RequireActiveBilling(user); err != nil {
				logger.Info("BillingMiddleware: billing check failed",
					zap.String("user_id", userID),
					zap.String("billing_status", user.BillingStatus),
					zap.Error(err),
				)
				w.WriteHeader(http.StatusPaymentRequired)
				json.NewEncoder(w).Encode(map[string]string{
					"error":         err.Error(),
					"billing_status": user.BillingStatus,
				})
				return
			}

			// Billing is active - continue
			next.ServeHTTP(w, r)
		})
	}
}

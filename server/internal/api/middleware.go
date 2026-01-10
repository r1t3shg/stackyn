package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

// AuthMiddleware validates JWT tokens and adds user info to context
// Supports both backend JWT tokens and Firebase tokens
func AuthMiddleware(jwtService *services.JWTService, userRepo *UserRepo, logger *zap.Logger) func(http.Handler) http.Handler {
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

			// Try to validate as backend JWT token first
			claims, err := jwtService.ValidateToken(token)
			if err == nil {
				// Backend JWT token is valid
				ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
				ctx = context.WithValue(ctx, "user_email", claims.Email)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Log the backend JWT validation error for debugging
			logger.Debug("Backend JWT validation failed, trying Firebase token", zap.Error(err))

			// If backend JWT validation failed, try Firebase token
			// Firebase tokens have 3 parts separated by dots
			tokenParts := strings.Split(token, ".")
			if len(tokenParts) != 3 {
				// Token doesn't have 3 parts - it's invalid for both backend and Firebase
				logger.Warn("Invalid token format - not a valid JWT (missing parts)", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Invalid or expired token"}`))
				return
			}

			// Decode Firebase token payload (second part)
			payload, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
			if err != nil {
				logger.Warn("Failed to decode Firebase token", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Invalid token format"}`))
				return
			}

			// Parse Firebase token payload
			var firebaseClaims map[string]interface{}
			if err := json.Unmarshal(payload, &firebaseClaims); err != nil {
				logger.Warn("Failed to parse Firebase token", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Invalid token format"}`))
				return
			}

			// Check if this looks like a backend JWT (has "user_id" field)
			// Backend JWTs have "user_id" field, Firebase tokens don't
			if userID, hasUserID := firebaseClaims["user_id"].(string); hasUserID && userID != "" {
				// This is a backend JWT that failed validation
				logger.Warn("Backend JWT token validation failed", 
					zap.Error(err), 
					zap.String("user_id", userID),
					zap.String("error_type", fmt.Sprintf("%T", err)))
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Invalid or expired token"}`))
				return
			}

			// Extract email from Firebase token
			email, ok := firebaseClaims["email"].(string)
			if !ok || email == "" {
				logger.Warn("Firebase token missing email")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Token missing required information"}`))
				return
			}

			// Look up user by email in database
			if userRepo == nil {
				logger.Error("User repository not available for Firebase token lookup")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"Authentication service unavailable"}`))
				return
			}

			user, err := userRepo.GetUserByEmail(email)
			if err != nil {
				// If user doesn't exist, create them automatically from Firebase token
				if errors.Is(err, pgx.ErrNoRows) {
					logger.Info("Creating new user from Firebase token", zap.String("email", email))
					
					// Extract full name from Firebase token if available
					fullName := ""
					if name, ok := firebaseClaims["name"].(string); ok && name != "" {
						fullName = name
					}
					
					// Create user without password (Firebase handles authentication)
					user, err = userRepo.CreateUser(email, fullName, "", "")
					if err != nil {
						logger.Error("Failed to create user from Firebase token", zap.String("email", email), zap.Error(err))
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte(`{"error":"Failed to create user account"}`))
						return
					}
					logger.Info("Created new user from Firebase token", zap.String("user_id", user.ID), zap.String("email", email))
				} else {
					logger.Warn("Failed to get user for Firebase token", zap.String("email", email), zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"Authentication service error"}`))
					return
				}
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), "user_id", user.ID)
			ctx = context.WithValue(ctx, "user_email", user.Email)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}


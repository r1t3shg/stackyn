package api

import (
	"context"
	"net/http"
	"strings"

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

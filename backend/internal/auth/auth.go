// Package auth provides JWT-based authentication middleware and utilities.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"

// GetUserID extracts user_id from request context
func GetUserID(r *http.Request) (string, bool) {
	if userID, ok := r.Context().Value(userIDKey).(string); ok {
		return userID, true
	}
	return "", false
}

// SetUserID sets user_id in request context
func SetUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), userIDKey, userID)
	return r.WithContext(ctx)
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

func init() {
	// Try to get JWT secret from environment variable
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a random secret if not set (for development only)
		log.Println("WARNING: JWT_SECRET not set, generating a random secret (this will invalidate tokens on restart)")
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatalf("Failed to generate JWT secret: %v", err)
		}
		secret = base64.URLEncoding.EncodeToString(b)
	}
	jwtSecret = []byte(secret)
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, email string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyToken verifies and parses a JWT token
func VerifyToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// AuthMiddleware is a middleware that verifies JWT tokens and sets user_id in context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check if it starts with "Bearer "
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Verify the token
		claims, err := VerifyToken(tokenString)
		if err != nil {
			log.Printf("[AUTH] Token verification failed: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Set user_id in context
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}


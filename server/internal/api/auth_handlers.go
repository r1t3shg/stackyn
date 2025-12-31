package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
	"stackyn/server/internal/services"
)

type AuthHandlers struct {
	logger     *zap.Logger
	otpService *services.OTPService
	jwtService *services.JWTService
	userRepo   UserRepository
	otpRepo    OTPRepository
}

// GetJWTService returns the JWT service (for use in handlers)
func (h *AuthHandlers) GetJWTService() *services.JWTService {
	return h.jwtService
}

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	FullName    string `json:"full_name,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
}

type UserRepository interface {
	GetUserByEmail(email string) (*User, error)
	CreateUser(email, fullName, companyName string) (*User, error)
	UpdateUser(userID, fullName, companyName string) (*User, error)
}

type OTPRepository interface {
	CreateOTP(email, otpHash string, expiresAt time.Time) error
	GetOTPByEmail(email string) (otpID, otpHash string, expiresAt time.Time, err error)
	MarkOTPAsUsed(otpID string) error
}

type SendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type SendOTPResponse struct {
	Message string `json:"message"`
	OTP     string `json:"otp,omitempty"` // Only in development
}

type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type VerifyOTPResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func NewAuthHandlers(logger *zap.Logger, otpService *services.OTPService, jwtService *services.JWTService, userRepo UserRepository, otpRepo OTPRepository) *AuthHandlers {
	return &AuthHandlers{
		logger:     logger,
		otpService: otpService,
		jwtService: jwtService,
		userRepo:   userRepo,
		otpRepo:    otpRepo,
	}
}

// SendOTP sends an OTP to the user's email
// POST /api/auth/send-otp
func (h *AuthHandlers) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate email
	if !ValidateEmail(req.Email) {
		h.writeError(w, http.StatusBadRequest, "Invalid email address")
		return
	}

	// Generate and send OTP
	otp, err := h.otpService.SendOTP(req.Email)
	if err != nil {
		h.logger.Error("Failed to send OTP", zap.Error(err), zap.String("email", req.Email))
		h.writeError(w, http.StatusInternalServerError, "Failed to send OTP")
		return
	}

	// In development, return OTP in response (remove in production)
	response := SendOTPResponse{
		Message: "OTP sent to email",
	}
	
	// Only include OTP in development mode
	if r.URL.Query().Get("dev") == "true" {
		response.OTP = otp
		h.logger.Info("OTP generated (dev mode)", zap.String("email", req.Email), zap.String("otp", otp))
	} else {
		h.logger.Info("OTP sent", zap.String("email", req.Email))
		// TODO: Send OTP via email service
	}

	h.writeJSON(w, http.StatusOK, response)
}

// VerifyOTP verifies an OTP and returns a JWT token
// POST /api/auth/verify-otp
func (h *AuthHandlers) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate email
	if !ValidateEmail(req.Email) {
		h.writeError(w, http.StatusBadRequest, "Invalid email address")
		return
	}

	// Validate OTP format
	if len(req.OTP) != 6 {
		h.writeError(w, http.StatusBadRequest, "OTP must be 6 digits")
		return
	}

	// Get OTP from database
	otpID, otpHash, expiresAt, err := h.otpRepo.GetOTPByEmail(req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeError(w, http.StatusUnauthorized, "Invalid or expired OTP")
			return
		}
		h.logger.Error("Failed to get OTP", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to verify OTP")
		return
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		h.writeError(w, http.StatusUnauthorized, "OTP has expired")
		return
	}

	// Verify OTP
	if !h.otpService.VerifyOTP(req.OTP, otpHash) {
		h.writeError(w, http.StatusUnauthorized, "Invalid OTP")
		return
	}

	// Mark OTP as used
	if err := h.otpRepo.MarkOTPAsUsed(otpID); err != nil {
		h.logger.Warn("Failed to mark OTP as used", zap.Error(err))
	}

	// Get or create user
	user, err := h.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create new user
			user, err = h.userRepo.CreateUser(req.Email, "", "")
			if err != nil {
				h.logger.Error("Failed to create user", zap.Error(err))
				h.writeError(w, http.StatusInternalServerError, "Failed to create user")
				return
			}
		} else {
			h.logger.Error("Failed to get user", zap.Error(err))
			h.writeError(w, http.StatusInternalServerError, "Failed to get user")
			return
		}
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user.ID, user.Email, 3600) // 1 hour expiration
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := VerifyOTPResponse{
		Token: token,
		User:  *user,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Login handles login with OTP (alias for verify-otp for compatibility)
// POST /api/auth/login
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate email
	if !ValidateEmail(req.Email) {
		h.writeError(w, http.StatusBadRequest, "Invalid email address")
		return
	}

	// Validate OTP format
	if len(req.OTP) != 6 {
		h.writeError(w, http.StatusBadRequest, "OTP must be 6 digits")
		return
	}

	// Get OTP from database
	otpID, otpHash, expiresAt, err := h.otpRepo.GetOTPByEmail(req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeError(w, http.StatusUnauthorized, "Invalid email or OTP")
			return
		}
		h.logger.Error("Failed to get OTP", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to verify OTP")
		return
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		h.writeError(w, http.StatusUnauthorized, "OTP has expired")
		return
	}

	// Verify OTP
	if !h.otpService.VerifyOTP(req.OTP, otpHash) {
		h.writeError(w, http.StatusUnauthorized, "Invalid email or OTP")
		return
	}

	// Mark OTP as used
	if err := h.otpRepo.MarkOTPAsUsed(otpID); err != nil {
		h.logger.Warn("Failed to mark OTP as used", zap.Error(err))
	}

	// Get user
	user, err := h.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeError(w, http.StatusUnauthorized, "User not found")
			return
		}
		h.logger.Error("Failed to get user", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user.ID, user.Email, 3600) // 1 hour expiration
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := LoginResponse{
		Token: token,
		User:  *user,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Helper to write JSON response
func (h *AuthHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// Helper to write error response
func (h *AuthHandlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// Helper function to validate email
func ValidateEmail(email string) bool {
	// Simple email validation - can be enhanced
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	// Check for @ symbol
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			if atIndex != -1 {
				return false // Multiple @ symbols
			}
			atIndex = i
		}
	}
	if atIndex == -1 || atIndex == 0 || atIndex == len(email)-1 {
		return false
	}
	// Check for dot after @
	hasDot := false
	for i := atIndex + 1; i < len(email); i++ {
		if email[i] == '.' {
			hasDot = true
			break
		}
	}
	return hasDot
}

type UpdateUserRequest struct {
	FullName    string `json:"full_name"`
	CompanyName string `json:"company_name"`
}

// UpdateUserProfile updates user profile details
// POST /api/auth/update-profile
// Requires: AuthMiddleware (sets user_id in context)
func (h *AuthHandlers) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by AuthMiddleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update user
	user, err := h.userRepo.UpdateUser(userID, req.FullName, req.CompanyName)
	if err != nil {
		h.logger.Error("Failed to update user", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	h.writeJSON(w, http.StatusOK, user)
}


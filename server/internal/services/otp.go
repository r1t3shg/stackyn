package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
)

type OTPService struct {
	logger       *zap.Logger
	db           OTPRepository
	emailService *EmailService
}

type OTPRepository interface {
	CreateOTP(email, otpHash string, expiresAt time.Time) error
	GetOTPByEmail(email string) (otpID, otpHash string, expiresAt time.Time, err error)
	MarkOTPAsUsed(otpID string) error
}

// NewOTPService creates a new OTP service
func NewOTPService(logger *zap.Logger, db OTPRepository, emailService *EmailService) *OTPService {
	return &OTPService{
		logger:       logger,
		db:           db,
		emailService: emailService,
	}
}

// GenerateOTP generates a 6-digit OTP
func (s *OTPService) GenerateOTP() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Generate 6-digit OTP (000000-999999)
	otp := fmt.Sprintf("%06d", int(bytes[0])<<16|int(bytes[1])<<8|int(bytes[2])%1000000)
	return otp, nil
}

// HashOTP hashes an OTP using bcrypt
func (s *OTPService) HashOTP(otp string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash OTP: %w", err)
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

// VerifyOTP verifies an OTP against a hash
func (s *OTPService) VerifyOTP(otp, hash string) bool {
	decodedHash, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword(decodedHash, []byte(otp)) == nil
}

// SendOTP generates, hashes, and stores an OTP for the given email
func (s *OTPService) SendOTP(email string) (string, error) {
	// Generate OTP
	otp, err := s.GenerateOTP()
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Hash OTP
	otpHash, err := s.HashOTP(otp)
	if err != nil {
		return "", fmt.Errorf("failed to hash OTP: %w", err)
	}

	// Set expiry to 10 minutes from now
	expiresAt := time.Now().Add(10 * time.Minute)

	// Store in database
	if err := s.db.CreateOTP(email, otpHash, expiresAt); err != nil {
		return "", fmt.Errorf("failed to store OTP: %w", err)
	}

	s.logger.Info("OTP generated and stored",
		zap.String("email", email),
		zap.Time("expires_at", expiresAt),
	)

	// Send OTP via email service
	if s.emailService != nil {
		if err := s.emailService.SendOTPEmail(email, otp); err != nil {
			s.logger.Error("Failed to send OTP email", zap.Error(err), zap.String("email", email))
			// Don't fail the entire operation if email fails - OTP is still stored
			// In development, we might want to return the OTP in the response
		}
	} else {
		s.logger.Warn("Email service not configured, OTP not sent", zap.String("email", email))
	}

	// Return plain OTP for sending via email (in production, send via email service)
	return otp, nil
}

// VerifyOTP verifies an OTP for the given email
func (s *OTPService) VerifyOTPForEmail(email, otp string) (bool, error) {
	// Get OTP from database
	_, otpHash, expiresAt, err := s.db.GetOTPByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("no valid OTP found for email")
		}
		return false, fmt.Errorf("failed to get OTP: %w", err)
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		return false, fmt.Errorf("OTP has expired")
	}

	// Verify OTP
	if !s.VerifyOTP(otp, otpHash) {
		return false, fmt.Errorf("invalid OTP")
	}

	return true, nil
}

// MarkOTPAsUsed marks an OTP as used
func (s *OTPService) MarkOTPAsUsed(email string) error {
	// Get OTP ID from database
	otpID, _, _, err := s.db.GetOTPByEmail(email)
	if err != nil {
		return err
	}

	// Mark as used using OTP ID
	return s.db.MarkOTPAsUsed(otpID)
}


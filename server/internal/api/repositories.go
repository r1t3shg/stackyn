package api

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"
	"stackyn/server/internal/db"
)

// OTPRepo implements OTPRepository interface using database
type OTPRepo struct {
	db     *db.DB
	logger *zap.Logger
}

// NewOTPRepo creates a new OTP repository
func NewOTPRepo(database *db.DB, logger *zap.Logger) *OTPRepo {
	return &OTPRepo{
		db:     database,
		logger: logger,
	}
}

// CreateOTP creates a new OTP record
func (r *OTPRepo) CreateOTP(email, otpHash string, expiresAt time.Time) error {
	ctx := context.Background()
	_, err := r.db.GetPool().Exec(ctx,
		"INSERT INTO otps (email, otp_hash, expires_at) VALUES ($1, $2, $3)",
		email, otpHash, expiresAt,
	)
	if err != nil {
		r.logger.Error("Failed to create OTP", zap.Error(err), zap.String("email", email))
		return err
	}
	return nil
}

// GetOTPByEmail retrieves the most recent unused OTP for an email
func (r *OTPRepo) GetOTPByEmail(email string) (otpID, otpHash string, expiresAt time.Time, err error) {
	ctx := context.Background()
	var id string
	err = r.db.GetPool().QueryRow(ctx,
		"SELECT id, otp_hash, expires_at FROM otps WHERE email = $1 AND used = false AND expires_at > NOW() ORDER BY created_at DESC LIMIT 1",
		email,
	).Scan(&id, &otpHash, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", time.Time{}, sql.ErrNoRows
		}
		r.logger.Error("Failed to get OTP", zap.Error(err), zap.String("email", email))
		return "", "", time.Time{}, err
	}
	return id, otpHash, expiresAt, nil
}

// MarkOTPAsUsed marks an OTP as used
func (r *OTPRepo) MarkOTPAsUsed(otpID string) error {
	ctx := context.Background()
	_, err := r.db.GetPool().Exec(ctx,
		"UPDATE otps SET used = true WHERE id = $1",
		otpID,
	)
	if err != nil {
		r.logger.Error("Failed to mark OTP as used", zap.Error(err), zap.String("otp_id", otpID))
		return err
	}
	return nil
}

// UserRepo implements UserRepository interface using database
type UserRepo struct {
	db     *db.DB
	logger *zap.Logger
}

// NewUserRepo creates a new user repository
func NewUserRepo(database *db.DB, logger *zap.Logger) *UserRepo {
	return &UserRepo{
		db:     database,
		logger: logger,
	}
}

// GetUserByEmail retrieves a user by email
func (r *UserRepo) GetUserByEmail(email string) (*User, error) {
	ctx := context.Background()
	var user User
	err := r.db.GetPool().QueryRow(ctx,
		"SELECT id, email, full_name, company_name FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to get user", zap.Error(err), zap.String("email", email))
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (r *UserRepo) CreateUser(email, fullName, companyName string) (*User, error) {
	ctx := context.Background()
	var user User
	err := r.db.GetPool().QueryRow(ctx,
		"INSERT INTO users (email, full_name, company_name) VALUES ($1, $2, $3) RETURNING id, email, full_name, company_name",
		email, fullName, companyName,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName)
	if err != nil {
		r.logger.Error("Failed to create user", zap.Error(err), zap.String("email", email))
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates user details
func (r *UserRepo) UpdateUser(userID, fullName, companyName string) (*User, error) {
	ctx := context.Background()
	var user User
	err := r.db.GetPool().QueryRow(ctx,
		"UPDATE users SET full_name = $1, company_name = $2 WHERE id = $3 RETURNING id, email, full_name, company_name",
		fullName, companyName, userID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName)
	if err != nil {
		r.logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	return &user, nil
}


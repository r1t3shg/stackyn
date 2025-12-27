// Package users provides data models and database operations for user authentication.
package users

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     *string   `json:"full_name,omitempty"`
	CompanyName  *string   `json:"company_name,omitempty"`
	EmailVerified bool     `json:"email_verified"`
	Plan         string    `json:"plan"` // Pricing plan (free, starter, builder, pro)
	PasswordHash string    `json:"-"` // Never expose password hash in JSON
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateUser creates a new user with a hashed password
func (s *Store) CreateUser(email, password string) (*User, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user User
	id := uuid.New().String()
	err = s.db.QueryRow(
		"INSERT INTO users (id, email, password_hash, plan) VALUES ($1, $2, $3, 'free') RETURNING id, email, plan, created_at, updated_at",
		id, email, string(hashedPassword),
	).Scan(&user.ID, &user.Email, &user.Plan, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUserWithDetails creates a new user with full details (for signup completion)
func (s *Store) CreateUserWithDetails(email, password, fullName, companyName string, emailVerified bool) (*User, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user User
	id := uuid.New().String()
	err = s.db.QueryRow(
		"INSERT INTO users (id, email, password_hash, full_name, company_name, email_verified, plan) VALUES ($1, $2, $3, $4, $5, $6, 'free') RETURNING id, email, full_name, company_name, email_verified, plan, created_at, updated_at",
		id, email, string(hashedPassword), fullName, companyName, emailVerified,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &user.EmailVerified, &user.Plan, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Store) GetUserByEmail(email string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		"SELECT id, email, password_hash, full_name, company_name, email_verified, COALESCE(plan, 'free') as plan, created_at, updated_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.CompanyName, &user.EmailVerified, &user.Plan, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *Store) GetUserByID(id string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		"SELECT id, email, password_hash, full_name, company_name, email_verified, COALESCE(plan, 'free') as plan, created_at, updated_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.CompanyName, &user.EmailVerified, &user.Plan, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// VerifyPassword checks if the provided password matches the user's hashed password
func (s *Store) VerifyPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// UpdatePlan updates a user's plan
func (s *Store) UpdatePlan(userID, plan string) error {
	_, err := s.db.Exec("UPDATE users SET plan = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", plan, userID)
	return err
}


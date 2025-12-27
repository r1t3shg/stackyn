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
	IsAdmin      bool      `json:"is_admin"` // Admin role flag
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
		"SELECT id, email, password_hash, full_name, company_name, email_verified, COALESCE(plan, 'free') as plan, COALESCE(is_admin, false) as is_admin, created_at, updated_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.CompanyName, &user.EmailVerified, &user.Plan, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *Store) GetUserByID(id string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		"SELECT id, email, password_hash, full_name, company_name, email_verified, COALESCE(plan, 'free') as plan, COALESCE(is_admin, false) as is_admin, created_at, updated_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.CompanyName, &user.EmailVerified, &user.Plan, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
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

// ListUsers retrieves all users with pagination and optional search
func (s *Store) ListUsers(limit, offset int, searchEmail string) ([]*User, error) {
	var query string
	var args []interface{}
	
	if searchEmail != "" {
		query = "SELECT id, email, full_name, company_name, email_verified, COALESCE(plan, 'free') as plan, COALESCE(is_admin, false) as is_admin, created_at, updated_at FROM users WHERE email ILIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		args = []interface{}{"%" + searchEmail + "%", limit, offset}
	} else {
		query = "SELECT id, email, full_name, company_name, email_verified, COALESCE(plan, 'free') as plan, COALESCE(is_admin, false) as is_admin, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		args = []interface{}{limit, offset}
	}
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []*User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &user.EmailVerified, &user.Plan, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

// CountUsers counts total number of users (for pagination)
func (s *Store) CountUsers(searchEmail string) (int, error) {
	var count int
	var err error
	
	if searchEmail != "" {
		err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE email ILIKE $1", "%"+searchEmail+"%").Scan(&count)
	} else {
		err = s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	}
	
	if err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateUserStatus updates a user's status (for suspend/activate)
// For now, we'll use a simple approach - we can add a status column later if needed
// This is a placeholder that can be extended
func (s *Store) UpdateUserStatus(userID string, suspended bool) error {
	// For now, we don't have a status column, so this is a placeholder
	// In a real implementation, you'd add a status column or use is_admin differently
	// For now, we'll just update updated_at to track changes
	_, err := s.db.Exec("UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = $1", userID)
	return err
}


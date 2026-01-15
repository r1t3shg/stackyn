package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// OTPRepo implements OTPRepository interface using database
type OTPRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewOTPRepo creates a new OTP repository
func NewOTPRepo(pool *pgxpool.Pool, logger *zap.Logger) *OTPRepo {
	return &OTPRepo{
		pool:   pool,
		logger: logger,
	}
}

// CreateOTP creates a new OTP record
func (r *OTPRepo) CreateOTP(email, otpHash string, expiresAt time.Time) error {
	ctx := context.Background()
	_, err := r.pool.Exec(ctx,
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
	err = r.pool.QueryRow(ctx,
		"SELECT id, otp_hash, expires_at FROM otps WHERE email = $1 AND used = false AND expires_at > NOW() ORDER BY created_at DESC LIMIT 1",
		email,
	).Scan(&id, &otpHash, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", time.Time{}, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get OTP", zap.Error(err), zap.String("email", email))
		return "", "", time.Time{}, err
	}
	return id, otpHash, expiresAt, nil
}

// MarkOTPAsUsed marks an OTP as used
func (r *OTPRepo) MarkOTPAsUsed(otpID string) error {
	ctx := context.Background()
	_, err := r.pool.Exec(ctx,
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
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewUserRepo creates a new user repository
func NewUserRepo(pool *pgxpool.Pool, logger *zap.Logger) *UserRepo {
	return &UserRepo{
		pool:   pool,
		logger: logger,
	}
}

// GetUserByEmail retrieves a user by email
func (r *UserRepo) GetUserByEmail(email string) (*User, error) {
	ctx := context.Background()
	var user User
	var passwordHash sql.NullString
	var billingStatus, plan, subscriptionID sql.NullString
	var trialStartedAt, trialEndsAt sql.NullTime
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, full_name, company_name, password_hash, 
		        billing_status, plan, trial_started_at, trial_ends_at, subscription_id 
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &passwordHash,
		&billingStatus, &plan, &trialStartedAt, &trialEndsAt, &subscriptionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get user", zap.Error(err), zap.String("email", email))
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if billingStatus.Valid {
		user.BillingStatus = billingStatus.String
	}
	if plan.Valid {
		user.Plan = plan.String
	}
	if subscriptionID.Valid {
		user.SubscriptionID = subscriptionID.String
	}
	if trialStartedAt.Valid {
		user.TrialStartedAt = &trialStartedAt.Time
	}
	if trialEndsAt.Valid {
		user.TrialEndsAt = &trialEndsAt.Time
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepo) GetUserByID(userID string) (*User, error) {
	ctx := context.Background()
	var user User
	var passwordHash sql.NullString
	var billingStatus, plan, subscriptionID sql.NullString
	var trialStartedAt, trialEndsAt sql.NullTime
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, full_name, company_name, password_hash, 
		        billing_status, plan, trial_started_at, trial_ends_at, subscription_id 
		 FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &passwordHash,
		&billingStatus, &plan, &trialStartedAt, &trialEndsAt, &subscriptionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get user", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if billingStatus.Valid {
		user.BillingStatus = billingStatus.String
	}
	if plan.Valid {
		user.Plan = plan.String
	}
	if subscriptionID.Valid {
		user.SubscriptionID = subscriptionID.String
	}
	if trialStartedAt.Valid {
		user.TrialStartedAt = &trialStartedAt.Time
	}
	if trialEndsAt.Valid {
		user.TrialEndsAt = &trialEndsAt.Time
	}
	return &user, nil
}

// DeleteUser deletes a user by ID (admin operation - cascades to apps, subscriptions, etc.)
func (r *UserRepo) DeleteUser(ctx context.Context, userID string) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		r.logger.Error("Failed to delete user", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	r.logger.Info("User deleted successfully", zap.String("user_id", userID))
	return nil
}

// GetUserDates retrieves created_at and updated_at for a user
func (r *UserRepo) GetUserDates(ctx context.Context, userID string) (createdAt, updatedAt time.Time, err error) {
	err = r.pool.QueryRow(ctx,
		"SELECT created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, time.Time{}, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get user dates", zap.Error(err), zap.String("user_id", userID))
		return time.Time{}, time.Time{}, err
	}
	return createdAt, updatedAt, nil
}

// CreateUser creates a new user (no default plan - trial is created separately)
func (r *UserRepo) CreateUser(email, fullName, companyName, passwordHash string) (*User, error) {
	ctx := context.Background()
	var user User
	var hash sql.NullString
	if passwordHash != "" {
		hash = sql.NullString{String: passwordHash, Valid: true}
	}
	
	// No default plan - users get a trial subscription instead
	var planID sql.NullString
	planID = sql.NullString{Valid: false}
	
	err := r.pool.QueryRow(ctx,
		"INSERT INTO users (email, full_name, company_name, password_hash, plan_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, email, full_name, company_name, password_hash",
		email, fullName, companyName, hash, planID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &hash)
	if err != nil {
		r.logger.Error("Failed to create user", zap.Error(err), zap.String("email", email))
		return nil, err
	}
	if hash.Valid {
		user.PasswordHash = hash.String
	}
	return &user, nil
}

// UpdateUser updates user details
func (r *UserRepo) UpdateUser(userID, fullName, companyName, passwordHash string) (*User, error) {
	ctx := context.Background()
	var user User
	var hash sql.NullString
	
	// Build query dynamically based on whether password is being updated
	if passwordHash != "" {
		hash = sql.NullString{String: passwordHash, Valid: true}
		err := r.pool.QueryRow(ctx,
			"UPDATE users SET full_name = $1, company_name = $2, password_hash = $3 WHERE id = $4 RETURNING id, email, full_name, company_name, password_hash",
			fullName, companyName, hash, userID,
		).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &hash)
		if err != nil {
			r.logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", userID))
			return nil, err
		}
	} else {
		err := r.pool.QueryRow(ctx,
			"UPDATE users SET full_name = $1, company_name = $2 WHERE id = $3 RETURNING id, email, full_name, company_name, password_hash",
			fullName, companyName, userID,
		).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &hash)
		if err != nil {
			r.logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", userID))
			return nil, err
		}
	}
	if hash.Valid {
		user.PasswordHash = hash.String
	}
	return &user, nil
}

// UpdateUserPassword updates a user's password
func (r *UserRepo) UpdateUserPassword(userID, passwordHash string) error {
	ctx := context.Background()
	hash := sql.NullString{String: passwordHash, Valid: true}
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET password_hash = $1 WHERE id = $2",
		hash, userID,
	)
	if err != nil {
		r.logger.Error("Failed to update user password", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	return nil
}

// UpdateUserBilling updates a user's billing fields
func (r *UserRepo) UpdateUserBilling(ctx context.Context, userID, billingStatus, plan, subscriptionID string, trialStartedAt, trialEndsAt *time.Time) error {
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{userID}
	argNum := 2

	if billingStatus != "" {
		setParts = append(setParts, fmt.Sprintf("billing_status = $%d", argNum))
		args = append(args, billingStatus)
		argNum++
	}
	if plan != "" {
		setParts = append(setParts, fmt.Sprintf("plan = $%d", argNum))
		args = append(args, plan)
		argNum++
	}
	if subscriptionID != "" {
		setParts = append(setParts, fmt.Sprintf("subscription_id = $%d", argNum))
		args = append(args, subscriptionID)
		argNum++
	} else {
		// Allow setting to NULL
		setParts = append(setParts, "subscription_id = NULL")
	}
	if trialStartedAt != nil {
		setParts = append(setParts, fmt.Sprintf("trial_started_at = $%d", argNum))
		args = append(args, *trialStartedAt)
		argNum++
	} else {
		setParts = append(setParts, "trial_started_at = NULL")
	}
	if trialEndsAt != nil {
		setParts = append(setParts, fmt.Sprintf("trial_ends_at = $%d", argNum))
		args = append(args, *trialEndsAt)
		argNum++
	} else {
		setParts = append(setParts, "trial_ends_at = NULL")
	}

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $1", strings.Join(setParts, ", "))
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to update user billing", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	return nil
}

// ListAllUsers lists all users with pagination and optional search
func (r *UserRepo) ListAllUsers(limit, offset int, search string) ([]User, int, error) {
	ctx := context.Background()
	var rows pgx.Rows
	var err error
	var total int

	// Build query with optional search
	query := `SELECT id, email, full_name, company_name, password_hash FROM users`
	countQuery := `SELECT COUNT(*) FROM users`
	
	args := []interface{}{}
	argNum := 1
	
	if search != "" {
		query += ` WHERE email ILIKE $` + fmt.Sprintf("%d", argNum)
		countQuery += ` WHERE email ILIKE $` + fmt.Sprintf("%d", argNum)
		args = append(args, "%"+search+"%")
		argNum++
	}
	
	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", argNum) + ` OFFSET $` + fmt.Sprintf("%d", argNum+1)
	args = append(args, limit, offset)
	
	// Get total count
	err = r.pool.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		r.logger.Error("Failed to get total users count", zap.Error(err))
		return nil, 0, err
	}
	
	// Get users
	rows, err = r.pool.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to list users", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var passwordHash sql.NullString
		err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &passwordHash)
		if err != nil {
			r.logger.Error("Failed to scan user", zap.Error(err))
			continue
		}
		if passwordHash.Valid {
			user.PasswordHash = passwordHash.String
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating users", zap.Error(err))
		return nil, 0, err
	}

	return users, total, nil
}

// AppRepo implements AppRepository interface using database
type AppRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewAppRepo creates a new app repository
func NewAppRepo(pool *pgxpool.Pool, logger *zap.Logger) *AppRepo {
	return &AppRepo{
		pool:   pool,
		logger: logger,
	}
}

// GetAppsByUserID retrieves all apps for a user
func (r *AppRepo) GetAppsByUserID(userID string) ([]App, error) {
	ctx := context.Background()
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, slug, status, url, repo_url, branch, created_at, updated_at 
		 FROM apps 
		 WHERE user_id = $1 
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		r.logger.Error("Failed to get apps", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		var url sql.NullString
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&app.ID,
			&app.Name,
			&app.Slug,
			&app.Status,
			&url,
			&app.RepoURL,
			&app.Branch,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan app", zap.Error(err))
			continue
		}
		if url.Valid {
			app.URL = url.String
		}
		app.CreatedAt = createdAt.Format(time.RFC3339)
		app.UpdatedAt = updatedAt.Format(time.RFC3339)
		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating apps", zap.Error(err))
		return nil, err
	}

	return apps, nil
}

// GetAppCountByUserID gets the count of apps for a user
func (r *AppRepo) GetAppCountByUserID(userID string) (int, error) {
	ctx := context.Background()
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM apps WHERE user_id = $1",
		userID,
	).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to get app count", zap.Error(err), zap.String("user_id", userID))
		return 0, err
	}
	return count, nil
}

// ListAllApps lists all apps with pagination
func (r *AppRepo) ListAllApps(limit, offset int) ([]App, int, error) {
	ctx := context.Background()
	var total int
	
	// Get total count
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM apps").Scan(&total)
	if err != nil {
		r.logger.Error("Failed to get total apps count", zap.Error(err))
		return nil, 0, err
	}
	
	// Get apps
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, slug, status, url, repo_url, branch, created_at, updated_at 
		 FROM apps 
		 ORDER BY created_at DESC 
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		r.logger.Error("Failed to list apps", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		var url sql.NullString
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&app.ID,
			&app.Name,
			&app.Slug,
			&app.Status,
			&url,
			&app.RepoURL,
			&app.Branch,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan app", zap.Error(err))
			continue
		}
		if url.Valid {
			app.URL = url.String
		}
		app.CreatedAt = createdAt.Format(time.RFC3339)
		app.UpdatedAt = updatedAt.Format(time.RFC3339)
		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating apps", zap.Error(err))
		return nil, 0, err
	}

	return apps, total, nil
}

// GetAppByIDWithoutUserCheck retrieves an app by ID (no user ownership check, for admin)
func (r *AppRepo) GetAppByIDWithoutUserCheck(appID string) (*App, error) {
	ctx := context.Background()
	var app App
	var url sql.NullString
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, status, url, repo_url, branch, created_at, updated_at 
		 FROM apps 
		 WHERE id = $1`,
		appID,
	).Scan(
		&app.ID,
		&app.Name,
		&app.Slug,
		&app.Status,
		&url,
		&app.RepoURL,
		&app.Branch,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID))
		return nil, err
	}
	if url.Valid {
		app.URL = url.String
	}
	app.CreatedAt = createdAt.Format(time.RFC3339)
	app.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &app, nil
}

// GetAppByID retrieves an app by ID (must belong to the user)
func (r *AppRepo) GetAppByID(appID, userID string) (*App, error) {
	ctx := context.Background()
	var app App
	var url sql.NullString
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, status, url, repo_url, branch, created_at, updated_at 
		 FROM apps 
		 WHERE id = $1 AND user_id = $2`,
		appID, userID,
	).Scan(
		&app.ID,
		&app.Name,
		&app.Slug,
		&app.Status,
		&url,
		&app.RepoURL,
		&app.Branch,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get app", zap.Error(err), zap.String("app_id", appID), zap.String("user_id", userID))
		return nil, err
	}
	if url.Valid {
		app.URL = url.String
	}
	app.CreatedAt = createdAt.Format(time.RFC3339)
	app.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &app, nil
}

// CreateApp creates a new app in the database
func (r *AppRepo) CreateApp(userID, name, repoURL, branch string) (*App, error) {
	ctx := context.Background()
	
	// Generate slug from name (simple version - convert to lowercase, replace spaces with hyphens)
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters, keep only alphanumeric and hyphens
	var slugBuilder strings.Builder
	for _, char := range slug {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			slugBuilder.WriteRune(char)
		}
	}
	slug = slugBuilder.String()
	if slug == "" {
		slug = "app"
	}
	
	var app App
	var url sql.NullString
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`INSERT INTO apps (user_id, name, slug, repo_url, branch, status) 
		 VALUES ($1, $2, $3, $4, $5, 'pending') 
		 RETURNING id, name, slug, status, url, repo_url, branch, created_at, updated_at`,
		userID, name, slug, repoURL, branch,
	).Scan(
		&app.ID,
		&app.Name,
		&app.Slug,
		&app.Status,
		&url,
		&app.RepoURL,
		&app.Branch,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to create app", zap.Error(err), zap.String("user_id", userID), zap.String("name", name))
		return nil, err
	}
	if url.Valid {
		app.URL = url.String
	}
	app.CreatedAt = createdAt.Format(time.RFC3339)
	app.UpdatedAt = updatedAt.Format(time.RFC3339)
	
	return &app, nil
}

// GetAppUserID gets the user_id for an app (for admin operations)
func (r *AppRepo) GetAppUserID(ctx context.Context, appID string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx,
		"SELECT user_id FROM apps WHERE id = $1",
		appID,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		r.logger.Error("Failed to get app user_id", zap.Error(err), zap.String("app_id", appID))
		return "", err
	}
	return userID, nil
}

// DeleteApp deletes an app by ID (must belong to the user)
func (r *AppRepo) DeleteApp(appID, userID string) error {
	ctx := context.Background()
	
	// First verify the app exists and belongs to the user
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM apps WHERE id = $1 AND user_id = $2)",
		appID, userID,
	).Scan(&exists)
	if err != nil {
		r.logger.Error("Failed to check app ownership", zap.Error(err), zap.String("app_id", appID), zap.String("user_id", userID))
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	
	// Begin transaction to ensure atomic deletion of app and logs
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("Failed to begin transaction for app deletion", zap.Error(err), zap.String("app_id", appID))
		return err
	}
	// Defer rollback - will be a no-op if Commit succeeds
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			r.logger.Warn("Transaction rollback error (may be expected if commit succeeded)", zap.Error(err))
		}
	}()
	
	// Step 1: Delete all app_logs associated with this app
	// Note: app_logs uses TEXT for app_id, so it doesn't cascade automatically
	// Note: app_logs table is optional (only exists if Postgres log persistence is enabled)
	// Check if table exists before attempting to delete
	var tableExists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'app_logs'
		)`,
	).Scan(&tableExists)
	if err != nil {
		r.logger.Error("Failed to check if app_logs table exists", zap.Error(err), zap.String("app_id", appID))
		return err
	}
	
	if tableExists {
		logsResult, err := tx.Exec(ctx,
			"DELETE FROM app_logs WHERE app_id = $1",
			appID,
		)
		if err != nil {
			r.logger.Error("Failed to delete app logs", zap.Error(err), zap.String("app_id", appID))
			return err
		}
		r.logger.Info("Deleted app logs", 
			zap.String("app_id", appID), 
			zap.Int64("logs_deleted", logsResult.RowsAffected()),
		)
	} else {
		r.logger.Debug("app_logs table does not exist, skipping log deletion", zap.String("app_id", appID))
	}
	
	// Step 2: Delete the app (cascade will handle related records: build_jobs, deployments, env_vars, runtime_instances)
	result, err := tx.Exec(ctx,
		"DELETE FROM apps WHERE id = $1 AND user_id = $2",
		appID, userID,
	)
	if err != nil {
		// Transaction is aborted, return error (defer will handle rollback)
		r.logger.Error("Failed to delete app", zap.Error(err), zap.String("app_id", appID), zap.String("user_id", userID))
		return err
	}
	
	if result.RowsAffected() == 0 {
		r.logger.Warn("No app deleted", zap.String("app_id", appID), zap.String("user_id", userID))
		return pgx.ErrNoRows
	}
	
	// Commit transaction (this will prevent the defer rollback from executing)
	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("Failed to commit transaction for app deletion", zap.Error(err), zap.String("app_id", appID))
		return err
	}
	
	r.logger.Info("App and all associated resources deleted successfully", 
		zap.String("app_id", appID), 
		zap.String("user_id", userID),
	)
	return nil
}

// UpdateApp updates app status and URL
func (r *AppRepo) UpdateApp(appID, status, url string) error {
	ctx := context.Background()
	
	var urlValue sql.NullString
	if url != "" {
		urlValue = sql.NullString{String: url, Valid: true}
	}
	
	_, err := r.pool.Exec(ctx,
		`UPDATE apps SET status = $1, url = $2, updated_at = NOW() WHERE id = $3`,
		status, urlValue, appID,
	)
	if err != nil {
		r.logger.Error("Failed to update app", zap.Error(err), zap.String("app_id", appID), zap.String("status", status))
		return err
	}
	
	r.logger.Info("App updated successfully", zap.String("app_id", appID), zap.String("status", status), zap.String("url", url))
	return nil
}

// DeploymentRepo implements deployment repository using database
type DeploymentRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewDeploymentRepo creates a new deployment repository
func NewDeploymentRepo(pool *pgxpool.Pool, logger *zap.Logger) *DeploymentRepo {
	return &DeploymentRepo{
		pool:   pool,
		logger: logger,
	}
}

// CreateDeployment creates a new deployment record
// Returns the deployment UUID as a string
// build_job_id is optional (can be NULL) since it has a foreign key constraint
func (r *DeploymentRepo) CreateDeployment(appID, buildJobID, status, imageName, containerID, subdomain string) (string, error) {
	ctx := context.Background()
	var id string
	// Build job ID is optional - verify it exists in build_jobs table before using it
	// This prevents foreign key constraint violations
	var buildJobIDPtr interface{}
	if buildJobID != "" {
		// Verify build_job exists before using it
		var exists bool
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM build_jobs WHERE id = $1)`,
			buildJobID,
		).Scan(&exists)
		if err == nil && exists {
			buildJobIDPtr = buildJobID
			r.logger.Debug("Build job found in database, using build_job_id",
				zap.String("build_job_id", buildJobID),
				zap.String("app_id", appID),
			)
		} else {
			// Build job doesn't exist, use NULL instead
			// This prevents foreign key constraint violations
			// NOTE: This means build logs won't be retrievable for this deployment
			// TODO: Create build_job record in database before creating deployment
			if err != nil {
				r.logger.Warn("Failed to check if build job exists, using NULL for build_job_id",
					zap.Error(err),
					zap.String("build_job_id", buildJobID),
					zap.String("app_id", appID),
				)
			} else {
				r.logger.Warn("Build job not found in database, using NULL for build_job_id (build logs won't be retrievable)",
					zap.String("build_job_id", buildJobID),
					zap.String("app_id", appID),
					zap.String("suggestion", "Ensure build_job is created in build_jobs table before creating deployment"),
				)
			}
			buildJobIDPtr = nil
		}
	} else {
		buildJobIDPtr = nil
	}
	
	err := r.pool.QueryRow(ctx,
		`INSERT INTO deployments (app_id, build_job_id, status, image_name, container_id, subdomain)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		appID, buildJobIDPtr, status, imageName, containerID, subdomain,
	).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to create deployment",
			zap.Error(err),
			zap.String("app_id", appID),
			zap.String("build_job_id", buildJobID),
			zap.String("build_job_id_ptr", fmt.Sprintf("%v", buildJobIDPtr)),
		)
		return "", err
	}
	
	r.logger.Info("Deployment created successfully",
		zap.String("deployment_id", id),
		zap.String("app_id", appID),
		zap.String("build_job_id", buildJobID),
		zap.Bool("has_build_job_id", buildJobID != "" && buildJobIDPtr != nil),
	)
	return id, nil
}

// UpdateDeployment updates deployment status and details
func (r *DeploymentRepo) UpdateDeployment(deploymentID, status, imageName, containerID, subdomain, errorMsg string) error {
	ctx := context.Background()
	// Sanitize error message to remove NULL bytes (PostgreSQL TEXT cannot contain 0x00)
	if errorMsg != "" {
		errorMsg = strings.ReplaceAll(errorMsg, "\x00", "")
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE deployments 
		 SET status = COALESCE(NULLIF($2, ''), status),
		     image_name = COALESCE(NULLIF($3, ''), image_name),
		     container_id = COALESCE(NULLIF($4, ''), container_id),
		     subdomain = COALESCE(NULLIF($5, ''), subdomain),
		     error_message = COALESCE(NULLIF($6, ''), error_message),
		     updated_at = NOW()
		 WHERE id = $1`,
		deploymentID, status, imageName, containerID, subdomain, errorMsg,
	)
	if err != nil {
		r.logger.Error("Failed to update deployment", zap.Error(err), zap.String("deployment_id", deploymentID))
		return err
	}
	return nil
}

// GetDeploymentsByAppID retrieves all deployments for an app
func (r *DeploymentRepo) GetDeploymentsByAppID(appID string) ([]map[string]interface{}, error) {
	ctx := context.Background()
	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, build_job_id, status, image_name, container_id, subdomain, 
		        build_log, runtime_log, error_message, created_at, updated_at
		 FROM deployments
		 WHERE app_id = $1
		 ORDER BY created_at DESC`,
		appID,
	)
	if err != nil {
		r.logger.Error("Failed to get deployments", zap.Error(err), zap.String("app_id", appID))
		return nil, err
	}
	defer rows.Close()

	var deployments []map[string]interface{}
	for rows.Next() {
		var id, appID string // UUIDs are strings
		var status string
		var buildJobID, imageName, containerID, subdomain sql.NullString
		var buildLog, runtimeLog, errorMsg sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &appID, &buildJobID, &status, &imageName, &containerID, &subdomain,
			&buildLog, &runtimeLog, &errorMsg, &createdAt, &updatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan deployment", zap.Error(err))
			continue
		}

		deployment := map[string]interface{}{
			"id":         id,
			"app_id":     appID,
			"status":     status,
			"created_at": createdAt.Format(time.RFC3339),
			"updated_at": updatedAt.Format(time.RFC3339),
		}

		if buildJobID.Valid {
			deployment["build_job_id"] = buildJobID.String
		}
		if imageName.Valid {
			deployment["image_name"] = map[string]interface{}{"String": imageName.String, "Valid": true}
		} else {
			deployment["image_name"] = map[string]interface{}{"String": "", "Valid": false}
		}
		if containerID.Valid {
			deployment["container_id"] = map[string]interface{}{"String": containerID.String, "Valid": true}
		} else {
			deployment["container_id"] = map[string]interface{}{"String": "", "Valid": false}
		}
		if subdomain.Valid {
			deployment["subdomain"] = map[string]interface{}{"String": subdomain.String, "Valid": true}
		} else {
			deployment["subdomain"] = map[string]interface{}{"String": "", "Valid": false}
		}
		if buildLog.Valid {
			deployment["build_log"] = map[string]interface{}{"String": buildLog.String, "Valid": true}
		} else {
			deployment["build_log"] = map[string]interface{}{"String": "", "Valid": false}
		}
		if runtimeLog.Valid {
			deployment["runtime_log"] = map[string]interface{}{"String": runtimeLog.String, "Valid": true}
		} else {
			deployment["runtime_log"] = map[string]interface{}{"String": "", "Valid": false}
		}
		if errorMsg.Valid {
			deployment["error_message"] = map[string]interface{}{"String": errorMsg.String, "Valid": true}
		} else {
			deployment["error_message"] = map[string]interface{}{"String": "", "Valid": false}
		}

		deployments = append(deployments, deployment)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating deployments", zap.Error(err))
		return nil, err
	}

	return deployments, nil
}

// UpdateDeploymentsByContainerIDs updates deployment status for multiple containers
func (r *DeploymentRepo) UpdateDeploymentsByContainerIDs(ctx context.Context, containerIDs []string, status string) error {
	if len(containerIDs) == 0 {
		return nil
	}
	
	_, err := r.pool.Exec(ctx,
		`UPDATE deployments 
		 SET status = $1, updated_at = NOW()
		 WHERE container_id = ANY($2::text[])`,
		status, containerIDs,
	)
	if err != nil {
		r.logger.Error("Failed to update deployments by container IDs", 
			zap.Error(err), 
			zap.Strings("container_ids", containerIDs),
			zap.String("status", status),
		)
		return err
	}
	
	r.logger.Info("Updated deployments to stopped status",
		zap.Int("count", len(containerIDs)),
		zap.String("status", status),
	)
	return nil
}

// GetDeploymentByID retrieves a deployment by ID
func (r *DeploymentRepo) GetDeploymentByID(deploymentID string) (map[string]interface{}, error) {
	ctx := context.Background()
	var id, appID string // UUIDs are strings
	var status string
	var buildJobID, imageName, containerID, subdomain sql.NullString
	var buildLog, runtimeLog, errorMsg sql.NullString
	var createdAt, updatedAt time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, app_id, build_job_id, status, image_name, container_id, subdomain,
		        build_log, runtime_log, error_message, created_at, updated_at
		 FROM deployments
		 WHERE id = $1`,
		deploymentID,
	).Scan(
		&id, &appID, &buildJobID, &status, &imageName, &containerID, &subdomain,
		&buildLog, &runtimeLog, &errorMsg, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get deployment", zap.Error(err), zap.String("deployment_id", deploymentID))
		return nil, err
	}

	deployment := map[string]interface{}{
		"id":         id,
		"app_id":     appID,
		"status":     status,
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	}

	// Always include build_job_id, even if NULL (for debugging and API consistency)
	if buildJobID.Valid {
		deployment["build_job_id"] = buildJobID.String
	} else {
		deployment["build_job_id"] = nil // Explicitly set to nil so we know it exists but is NULL
	}
	if imageName.Valid {
		deployment["image_name"] = map[string]interface{}{"String": imageName.String, "Valid": true}
	} else {
		deployment["image_name"] = map[string]interface{}{"String": "", "Valid": false}
	}
	if containerID.Valid {
		deployment["container_id"] = map[string]interface{}{"String": containerID.String, "Valid": true}
	} else {
		deployment["container_id"] = map[string]interface{}{"String": "", "Valid": false}
	}
	if subdomain.Valid {
		deployment["subdomain"] = map[string]interface{}{"String": subdomain.String, "Valid": true}
	} else {
		deployment["subdomain"] = map[string]interface{}{"String": "", "Valid": false}
	}
	if buildLog.Valid {
		deployment["build_log"] = map[string]interface{}{"String": buildLog.String, "Valid": true}
	} else {
		deployment["build_log"] = map[string]interface{}{"String": "", "Valid": false}
	}
	if runtimeLog.Valid {
		deployment["runtime_log"] = map[string]interface{}{"String": runtimeLog.String, "Valid": true}
	} else {
		deployment["runtime_log"] = map[string]interface{}{"String": "", "Valid": false}
	}
	if errorMsg.Valid {
		deployment["error_message"] = map[string]interface{}{"String": errorMsg.String, "Valid": true}
	} else {
		deployment["error_message"] = map[string]interface{}{"String": "", "Valid": false}
	}

	return deployment, nil
}

// PlanRepo implements plan repository using database
type PlanRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPlanRepo creates a new plan repository
func NewPlanRepo(pool *pgxpool.Pool, logger *zap.Logger) *PlanRepo {
	return &PlanRepo{
		pool:   pool,
		logger: logger,
	}
}

// Plan represents a plan from the database
type Plan struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"`
	Price            int       `json:"price"`
	MaxRAMMB         int       `json:"max_ram_mb"`
	MaxDiskMB        int       `json:"max_disk_mb"`
	MaxApps          int       `json:"max_apps"`
	AlwaysOn         bool      `json:"always_on"`
	AutoDeploy       bool      `json:"auto_deploy"`
	HealthChecks     bool      `json:"health_checks"`
	Logs             bool      `json:"logs"`
	ZeroDowntime     bool      `json:"zero_downtime"`
	Workers          bool      `json:"workers"`
	PriorityBuilds   bool      `json:"priority_builds"`
	ManualDeployOnly bool      `json:"manual_deploy_only"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// GetPlanByID retrieves a plan by ID
func (r *PlanRepo) GetPlanByID(ctx context.Context, planID string) (*Plan, error) {
	var plan Plan
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, display_name, price, max_ram_mb, max_disk_mb, max_apps,
		        always_on, auto_deploy, health_checks, logs, zero_downtime,
		        workers, priority_builds, manual_deploy_only, created_at, updated_at
		 FROM plans
		 WHERE id = $1`,
		planID,
	).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Price,
		&plan.MaxRAMMB, &plan.MaxDiskMB, &plan.MaxApps,
		&plan.AlwaysOn, &plan.AutoDeploy, &plan.HealthChecks, &plan.Logs,
		&plan.ZeroDowntime, &plan.Workers, &plan.PriorityBuilds,
		&plan.ManualDeployOnly, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get plan by ID", zap.Error(err), zap.String("plan_id", planID))
		return nil, err
	}
	return &plan, nil
}

// GetPlanByName retrieves a plan by name
func (r *PlanRepo) GetPlanByName(ctx context.Context, planName string) (*Plan, error) {
	var plan Plan
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, display_name, price, max_ram_mb, max_disk_mb, max_apps,
		        always_on, auto_deploy, health_checks, logs, zero_downtime,
		        workers, priority_builds, manual_deploy_only, created_at, updated_at
		 FROM plans
		 WHERE name = $1`,
		planName,
	).Scan(
		&plan.ID, &plan.Name, &plan.DisplayName, &plan.Price,
		&plan.MaxRAMMB, &plan.MaxDiskMB, &plan.MaxApps,
		&plan.AlwaysOn, &plan.AutoDeploy, &plan.HealthChecks, &plan.Logs,
		&plan.ZeroDowntime, &plan.Workers, &plan.PriorityBuilds,
		&plan.ManualDeployOnly, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get plan by name", zap.Error(err), zap.String("plan_name", planName))
		return nil, err
	}
	return &plan, nil
}

// GetDefaultPlan retrieves the default (starter) plan
func (r *PlanRepo) GetDefaultPlan(ctx context.Context) (*Plan, error) {
	return r.GetPlanByName(ctx, "starter")
}

// SubscriptionRepo implements subscription repository using database
type SubscriptionRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewSubscriptionRepo creates a new subscription repository
func NewSubscriptionRepo(pool *pgxpool.Pool, logger *zap.Logger) *SubscriptionRepo {
	return &SubscriptionRepo{
		pool:   pool,
		logger: logger,
	}
}

// Subscription represents a subscription from the database
type Subscription struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	LemonSubscriptionID *string    `json:"lemon_subscription_id,omitempty"` // External subscription ID (e.g., Lemon Squeezy) - nullable
	Plan               string     `json:"plan"`                             // Plan name (starter | pro)
	Status             string     `json:"status"`                           // trial | active | expired | cancelled
	TrialStartedAt     *time.Time `json:"trial_started_at,omitempty"`       // When trial started
	TrialEndsAt        *time.Time `json:"trial_ends_at,omitempty"`          // When trial ends
	RAMLimitMB         int        `json:"ram_limit_mb"`                     // RAM limit in MB
	DiskLimitGB        int        `json:"disk_limit_gb"`                    // Disk limit in GB
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// GetSubscriptionByUserID retrieves a subscription for a user
// Prefers active or trial subscriptions over expired/cancelled ones
func (r *SubscriptionRepo) GetSubscriptionByUserID(ctx context.Context, userID string) (*Subscription, error) {
	var sub Subscription
	var lemonSubID sql.NullString
	var trialStartedAt, trialEndsAt sql.NullTime
	
	// First try to get an active or trial subscription
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, lemon_subscription_id, plan, status, trial_started_at, trial_ends_at, 
		        ram_limit_mb, disk_limit_gb, created_at, updated_at
		 FROM subscriptions
		 WHERE user_id = $1 AND status IN ('trial', 'active')
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	).Scan(
		&sub.ID, &sub.UserID, &lemonSubID, &sub.Plan, &sub.Status,
		&trialStartedAt, &trialEndsAt, &sub.RAMLimitMB, &sub.DiskLimitGB,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	
	// If no active/trial subscription found, get the most recent one (might be expired)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		err = r.pool.QueryRow(ctx,
			`SELECT id, user_id, lemon_subscription_id, plan, status, trial_started_at, trial_ends_at, 
			        ram_limit_mb, disk_limit_gb, created_at, updated_at
			 FROM subscriptions
			 WHERE user_id = $1
			 ORDER BY created_at DESC
			 LIMIT 1`,
			userID,
		).Scan(
			&sub.ID, &sub.UserID, &lemonSubID, &sub.Plan, &sub.Status,
			&trialStartedAt, &trialEndsAt, &sub.RAMLimitMB, &sub.DiskLimitGB,
			&sub.CreatedAt, &sub.UpdatedAt,
		)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		r.logger.Error("Failed to get subscription", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	if lemonSubID.Valid {
		sub.LemonSubscriptionID = &lemonSubID.String
	}
	if trialStartedAt.Valid {
		sub.TrialStartedAt = &trialStartedAt.Time
	}
	if trialEndsAt.Valid {
		sub.TrialEndsAt = &trialEndsAt.Time
	}
	return &sub, nil
}

// CreateSubscription creates a new subscription
// If creating a trial, subscriptionID should be empty string and trial_started_at/trial_ends_at should be set
func (r *SubscriptionRepo) CreateSubscription(ctx context.Context, userID, lemonSubscriptionID, plan, status string, trialStartedAt, trialEndsAt *time.Time, ramLimitMB, diskLimitGB int) (*Subscription, error) {
	var sub Subscription
	var lemonSubID sql.NullString
	var trialStart, trialEnd sql.NullTime
	
	// Handle nullable lemon_subscription_id
	var lemonSubIDPtr interface{}
	if lemonSubscriptionID != "" {
		lemonSubIDPtr = lemonSubscriptionID
	} else {
		lemonSubIDPtr = nil
	}
	
	// Handle nullable trial dates
	var trialStartPtr, trialEndPtr interface{}
	if trialStartedAt != nil {
		trialStartPtr = *trialStartedAt
	} else {
		trialStartPtr = nil
	}
	if trialEndsAt != nil {
		trialEndPtr = *trialEndsAt
	} else {
		trialEndPtr = nil
	}
	
	err := r.pool.QueryRow(ctx,
		`INSERT INTO subscriptions (user_id, lemon_subscription_id, plan, status, trial_started_at, trial_ends_at, ram_limit_mb, disk_limit_gb)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, user_id, lemon_subscription_id, plan, status, trial_started_at, trial_ends_at, ram_limit_mb, disk_limit_gb, created_at, updated_at`,
		userID, lemonSubIDPtr, plan, status, trialStartPtr, trialEndPtr, ramLimitMB, diskLimitGB,
	).Scan(
		&sub.ID, &sub.UserID, &lemonSubID, &sub.Plan, &sub.Status,
		&trialStart, &trialEnd, &sub.RAMLimitMB, &sub.DiskLimitGB,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		// Check if error is due to unique constraint (user already has active/trial subscription)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Unique constraint violation - user already has an active/trial subscription
			// Try to get the existing subscription
			r.logger.Warn("Unique constraint violation creating subscription, fetching existing",
				zap.String("user_id", userID),
				zap.String("pg_error_code", pgErr.Code),
			)
			existingSub, getErr := r.GetSubscriptionByUserID(ctx, userID)
			if getErr == nil {
				return existingSub, nil
			}
		}
		r.logger.Error("Failed to create subscription", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	if lemonSubID.Valid {
		sub.LemonSubscriptionID = &lemonSubID.String
	}
	if trialStart.Valid {
		sub.TrialStartedAt = &trialStart.Time
	}
	if trialEnd.Valid {
		sub.TrialEndsAt = &trialEnd.Time
	}
	return &sub, nil
}

// UpdateSubscription updates a subscription by internal ID
func (r *SubscriptionRepo) UpdateSubscription(ctx context.Context, subscriptionID, plan, status string, ramLimitMB, diskLimitGB *int, lemonSubID *string) error {
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{subscriptionID}
	argNum := 2
	
	if plan != "" {
		setParts = append(setParts, fmt.Sprintf("plan = $%d", argNum))
		args = append(args, plan)
		argNum++
	}
	if status != "" {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argNum))
		args = append(args, status)
		argNum++
	}
	if ramLimitMB != nil {
		setParts = append(setParts, fmt.Sprintf("ram_limit_mb = $%d", argNum))
		args = append(args, *ramLimitMB)
		argNum++
	}
	if diskLimitGB != nil {
		setParts = append(setParts, fmt.Sprintf("disk_limit_gb = $%d", argNum))
		args = append(args, *diskLimitGB)
		argNum++
	}
	if lemonSubID != nil {
		if *lemonSubID == "" {
			setParts = append(setParts, fmt.Sprintf("lemon_subscription_id = NULL"))
		} else {
			setParts = append(setParts, fmt.Sprintf("lemon_subscription_id = $%d", argNum))
			args = append(args, *lemonSubID)
			argNum++
		}
	}
	
	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id = $1", strings.Join(setParts, ", "))
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to update subscription", zap.Error(err), zap.String("subscription_id", subscriptionID))
		return err
	}
	return nil
}

// UpdateSubscriptionByUserID updates a user's subscription
func (r *SubscriptionRepo) UpdateSubscriptionByUserID(ctx context.Context, userID, plan, status string, ramLimitMB, diskLimitGB *int, lemonSubID *string) error {
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{userID}
	argNum := 2
	
	if plan != "" {
		setParts = append(setParts, fmt.Sprintf("plan = $%d", argNum))
		args = append(args, plan)
		argNum++
	}
	if status != "" {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argNum))
		args = append(args, status)
		argNum++
	}
	if ramLimitMB != nil {
		setParts = append(setParts, fmt.Sprintf("ram_limit_mb = $%d", argNum))
		args = append(args, *ramLimitMB)
		argNum++
	}
	if diskLimitGB != nil {
		setParts = append(setParts, fmt.Sprintf("disk_limit_gb = $%d", argNum))
		args = append(args, *diskLimitGB)
		argNum++
	}
	if lemonSubID != nil {
		if *lemonSubID == "" {
			setParts = append(setParts, fmt.Sprintf("lemon_subscription_id = NULL"))
		} else {
			setParts = append(setParts, fmt.Sprintf("lemon_subscription_id = $%d", argNum))
			args = append(args, *lemonSubID)
			argNum++
		}
	}
	
	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE user_id = $1", strings.Join(setParts, ", "))
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to update subscription by user ID", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	return nil
}

// GetTrialSubscriptions retrieves all trial subscriptions that need processing
// Used by cron job for trial lifecycle management
func (r *SubscriptionRepo) GetTrialSubscriptions(ctx context.Context) ([]*Subscription, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, lemon_subscription_id, plan, status, trial_started_at, trial_ends_at, 
		        ram_limit_mb, disk_limit_gb, created_at, updated_at
		 FROM subscriptions
		 WHERE status = 'trial'
		 ORDER BY trial_ends_at ASC`,
	)
	if err != nil {
		r.logger.Error("Failed to get trial subscriptions", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*Subscription
	for rows.Next() {
		var sub Subscription
		var lemonSubID sql.NullString
		var trialStartedAt, trialEndsAt sql.NullTime
		
		err := rows.Scan(
			&sub.ID, &sub.UserID, &lemonSubID, &sub.Plan, &sub.Status,
			&trialStartedAt, &trialEndsAt, &sub.RAMLimitMB, &sub.DiskLimitGB,
			&sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan subscription", zap.Error(err))
			continue
		}
		
		if lemonSubID.Valid {
			sub.LemonSubscriptionID = &lemonSubID.String
		}
		if trialStartedAt.Valid {
			sub.TrialStartedAt = &trialStartedAt.Time
		}
		if trialEndsAt.Valid {
			sub.TrialEndsAt = &trialEndsAt.Time
		}
		
		subscriptions = append(subscriptions, &sub)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating subscriptions", zap.Error(err))
		return nil, err
	}

	return subscriptions, nil
}

// UserPlanRepo implements user plan repository for getting plan_id from users table
type UserPlanRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewUserPlanRepo creates a new user plan repository
func NewUserPlanRepo(pool *pgxpool.Pool, logger *zap.Logger) *UserPlanRepo {
	return &UserPlanRepo{
		pool:   pool,
		logger: logger,
	}
}

// GetUserPlanID retrieves the plan_id for a user
func (r *UserPlanRepo) GetUserPlanID(ctx context.Context, userID string) (string, error) {
	var planID sql.NullString
	err := r.pool.QueryRow(ctx,
		"SELECT plan_id FROM users WHERE id = $1",
		userID,
	).Scan(&planID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		r.logger.Error("Failed to get user plan_id", zap.Error(err), zap.String("user_id", userID))
		return "", err
	}
	if !planID.Valid {
		return "", nil // No plan assigned
	}
	return planID.String, nil
}

// UpdateUserPlanID updates a user's plan_id
func (r *UserPlanRepo) UpdateUserPlanID(ctx context.Context, userID, planID string) error {
	var planIDPtr interface{}
	if planID != "" {
		planIDPtr = planID
	} else {
		planIDPtr = nil
	}
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET plan_id = $1, updated_at = NOW() WHERE id = $2",
		planIDPtr, userID,
	)
	if err != nil {
		r.logger.Error("Failed to update user plan_id", zap.Error(err), zap.String("user_id", userID), zap.String("plan_id", planID))
		return err
	}
	return nil
}

// EnvVarRepo implements environment variables repository using database
type EnvVarRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewEnvVarRepo creates a new environment variables repository
func NewEnvVarRepo(pool *pgxpool.Pool, logger *zap.Logger) *EnvVarRepo {
	return &EnvVarRepo{
		pool:   pool,
		logger: logger,
	}
}

// GetEnvVarsByAppID retrieves all environment variables for an app
func (r *EnvVarRepo) GetEnvVarsByAppID(ctx context.Context, appID string) ([]*EnvVar, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, key, value, created_at, updated_at 
		 FROM env_vars 
		 WHERE app_id = $1 
		 ORDER BY created_at DESC`,
		appID,
	)
	if err != nil {
		r.logger.Error("Failed to get env vars", zap.Error(err), zap.String("app_id", appID))
		return nil, err
	}
	defer rows.Close()

	var envVars []*EnvVar
	for rows.Next() {
		var envVar EnvVar
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&envVar.ID,
			&envVar.AppID,
			&envVar.Key,
			&envVar.Value,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan env var", zap.Error(err))
			continue
		}
		envVar.CreatedAt = createdAt.Format(time.RFC3339)
		envVar.UpdatedAt = updatedAt.Format(time.RFC3339)
		envVars = append(envVars, &envVar)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating env vars", zap.Error(err))
		return nil, err
	}

	return envVars, nil
}

// CreateEnvVar creates a new environment variable
func (r *EnvVarRepo) CreateEnvVar(ctx context.Context, appID, key, value string) (*EnvVar, error) {
	var envVar EnvVar
	var createdAt, updatedAt time.Time
	
	err := r.pool.QueryRow(ctx,
		`INSERT INTO env_vars (app_id, key, value) 
		 VALUES ($1, $2, $3) 
		 ON CONFLICT (app_id, key) 
		 DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		 RETURNING id, app_id, key, value, created_at, updated_at`,
		appID, key, value,
	).Scan(
		&envVar.ID,
		&envVar.AppID,
		&envVar.Key,
		&envVar.Value,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to create env var", zap.Error(err), zap.String("app_id", appID), zap.String("key", key))
		return nil, err
	}
	
	envVar.CreatedAt = createdAt.Format(time.RFC3339)
	envVar.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &envVar, nil
}

// DeleteEnvVar deletes an environment variable by app ID and key
func (r *EnvVarRepo) DeleteEnvVar(ctx context.Context, appID, key string) error {
	result, err := r.pool.Exec(ctx,
		"DELETE FROM env_vars WHERE app_id = $1 AND key = $2",
		appID, key,
	)
	if err != nil {
		r.logger.Error("Failed to delete env var", zap.Error(err), zap.String("app_id", appID), zap.String("key", key))
		return err
	}
	
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	
	return nil
}

// BuildJobRepo handles build_jobs table operations
type BuildJobRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewBuildJobRepo creates a new BuildJob repository
func NewBuildJobRepo(pool *pgxpool.Pool, logger *zap.Logger) *BuildJobRepo {
	return &BuildJobRepo{
		pool:   pool,
		logger: logger,
	}
}

// CreateBuildJob creates a new build_job record in the database
// This ensures the build_job_id exists when CreateDeployment is called
func (r *BuildJobRepo) CreateBuildJob(ctx context.Context, buildJobID, appID, status string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO build_jobs (id, app_id, status)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO NOTHING`,
		buildJobID, appID, status,
	)
	if err != nil {
		r.logger.Error("Failed to create build_job",
			zap.Error(err),
			zap.String("build_job_id", buildJobID),
			zap.String("app_id", appID),
			zap.String("status", status),
		)
		return err
	}
	
	r.logger.Info("Build job created in database",
		zap.String("build_job_id", buildJobID),
		zap.String("app_id", appID),
		zap.String("status", status),
	)
	return nil
}

// UpdateBuildJob updates a build_job record
func (r *BuildJobRepo) UpdateBuildJob(ctx context.Context, buildJobID, status, buildLog, errorMsg string) error {
	// Sanitize error message to remove NULL bytes
	if errorMsg != "" {
		errorMsg = strings.ReplaceAll(errorMsg, "\x00", "")
	}
	
	_, err := r.pool.Exec(ctx,
		`UPDATE build_jobs 
		 SET status = COALESCE(NULLIF($2, ''), status),
		     build_log = COALESCE(NULLIF($3, ''), build_log),
		     error_message = COALESCE(NULLIF($4, ''), error_message),
		     updated_at = NOW()
		 WHERE id = $1`,
		buildJobID, status, buildLog, errorMsg,
	)
	if err != nil {
		r.logger.Error("Failed to update build_job",
			zap.Error(err),
			zap.String("build_job_id", buildJobID),
		)
		return err
	}
	return nil
}


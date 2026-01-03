package api

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
	err := r.pool.QueryRow(ctx,
		"SELECT id, email, full_name, company_name, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CompanyName, &passwordHash)
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
	return &user, nil
}

// CreateUser creates a new user
func (r *UserRepo) CreateUser(email, fullName, companyName, passwordHash string) (*User, error) {
	ctx := context.Background()
	var user User
	var hash sql.NullString
	if passwordHash != "" {
		hash = sql.NullString{String: passwordHash, Valid: true}
	}
	err := r.pool.QueryRow(ctx,
		"INSERT INTO users (email, full_name, company_name, password_hash) VALUES ($1, $2, $3, $4) RETURNING id, email, full_name, company_name, password_hash",
		email, fullName, companyName, hash,
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
	
	// Delete the app (cascade will handle related records if foreign keys are set up)
	result, err := r.pool.Exec(ctx,
		"DELETE FROM apps WHERE id = $1 AND user_id = $2",
		appID, userID,
	)
	if err != nil {
		r.logger.Error("Failed to delete app", zap.Error(err), zap.String("app_id", appID), zap.String("user_id", userID))
		return err
	}
	
	if result.RowsAffected() == 0 {
		r.logger.Warn("No app deleted", zap.String("app_id", appID), zap.String("user_id", userID))
		return pgx.ErrNoRows
	}
	
	r.logger.Info("App deleted successfully", zap.String("app_id", appID), zap.String("user_id", userID))
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
	// Build job ID is optional - if it doesn't exist in build_jobs table, set to NULL
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
		} else {
			// Build job doesn't exist, use NULL instead
			r.logger.Debug("Build job not found in database, using NULL for build_job_id",
				zap.String("build_job_id", buildJobID),
				zap.String("app_id", appID),
			)
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
		r.logger.Error("Failed to create deployment", zap.Error(err), zap.String("app_id", appID))
		return "", err
	}
	return id, nil
}

// UpdateDeployment updates deployment status and details
func (r *DeploymentRepo) UpdateDeployment(deploymentID, status, imageName, containerID, subdomain, errorMsg string) error {
	ctx := context.Background()
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

	return deployment, nil
}


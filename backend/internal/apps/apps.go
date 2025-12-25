// Package apps provides data models and database operations for applications.
// An app represents a deployable application with a Git repository, branch,
// and deployment status. Apps are the primary entity that users interact with.
//
// Key Concepts:
//   - App: A deployable application with name, repository URL, and branch
//   - Slug: URL-friendly version of the app name (auto-generated)
//   - Status: Current deployment status (Pending, Building, Running, Failed)
//   - URL: The public URL where the app is accessible
//
// Database Schema:
//   - apps table stores app metadata
//   - Each app can have multiple deployments (one-to-many relationship)
//   - Apps are associated with users via user_id (for multi-tenancy)
package apps

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type App struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"` // Not included in JSON response
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	URL       string    `json:"url"`
	RepoURL   string    `json:"repo_url"`
	Branch    string    `json:"branch"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(userID, name, repoURL, branch string) (*App, error) {
	log.Printf("Creating app with branch: '%s' for user: %s", branch, userID)
	var app App
	err := s.db.QueryRow(
		"INSERT INTO apps (user_id, name, repo_url, branch) VALUES ($1, $2, $3, $4) RETURNING id, user_id, name, repo_url, branch, COALESCE(url, '') as url, COALESCE(status, '') as status, created_at, updated_at",
		userID, name, repoURL, branch,
	).Scan(&app.ID, &app.UserID, &app.Name, &app.RepoURL, &app.Branch, &app.URL, &app.Status, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	log.Printf("App created with ID: %s, branch saved as: '%s'", app.ID, app.Branch)
	return &app, nil
}

func (s *Store) GetByID(id int) (*App, error) {
	var app App
	err := s.db.QueryRow(
		"SELECT id, name, COALESCE(slug, '') as slug, COALESCE(status, '') as status, COALESCE(url, '') as url, repo_url, COALESCE(branch, '') as branch, created_at, updated_at FROM apps WHERE id = $1",
		id,
	).Scan(&app.ID, &app.Name, &app.Slug, &app.Status, &app.URL, &app.RepoURL, &app.Branch, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Store) List() ([]*App, error) {
	rows, err := s.db.Query("SELECT id, name, COALESCE(slug, '') as slug, repo_url, COALESCE(branch, '') as branch, COALESCE(url, '') as url, COALESCE(status, '') as status, created_at, updated_at FROM apps ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		var app App
		if err := rows.Scan(&app.ID, &app.Name, &app.Slug, &app.RepoURL, &app.Branch, &app.URL, &app.Status, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, &app)
	}
	return apps, rows.Err()
}

func (s *Store) Delete(id int) error {
	_, err := s.db.Exec("DELETE FROM apps WHERE id = $1", id)
	return err
}

// UpdateStatus updates the status of an app
func (s *Store) UpdateStatus(id int, status string) error {
	_, err := s.db.Exec(
		"UPDATE apps SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		status, id,
	)
	return err
}

// UpdateURL updates the URL of an app
func (s *Store) UpdateURL(id int, url string) error {
	_, err := s.db.Exec(
		"UPDATE apps SET url = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		url, id,
	)
	return err
}

// UpdateStatusAndURL updates both status and URL of an app
func (s *Store) UpdateStatusAndURL(id int, status, url string) error {
	_, err := s.db.Exec(
		"UPDATE apps SET status = $1, url = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3",
		status, url, id,
	)
	return err
}

// ListAppsByUserID queries all apps owned by the given user_id, ordered by created_at DESC.
// Returns an empty slice if no apps are found.
// SQL Query:
//
//	SELECT id, user_id, name, slug, status, url, repo_url, branch, created_at, updated_at
//	FROM apps
//	WHERE user_id = $1
//	ORDER BY created_at DESC
func (s *Store) ListAppsByUserID(ctx context.Context, userID string) ([]App, error) {
	query := `
       SELECT id, user_id, name, COALESCE(slug, '') as slug, COALESCE(status, '') as status, COALESCE(url, '') as url, repo_url, COALESCE(branch, '') as branch, created_at, updated_at
       FROM apps
       WHERE user_id = $1
       ORDER BY created_at DESC
   `

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		if err := rows.Scan(
			&app.ID,
			&app.UserID,
			&app.Name,
			&app.Slug,
			&app.Status,
			&app.URL,
			&app.RepoURL,
			&app.Branch,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return apps, nil
}

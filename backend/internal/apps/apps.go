package apps

import (
	"context"
	"database/sql"
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

func (s *Store) Create(name, repoURL string) (*App, error) {
	var app App
	err := s.db.QueryRow(
		"INSERT INTO apps (name, repo_url) VALUES ($1, $2) RETURNING id, name, repo_url, created_at, updated_at",
		name, repoURL,
	).Scan(&app.ID, &app.Name, &app.RepoURL, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Store) GetByID(id int) (*App, error) {
	var app App
	err := s.db.QueryRow(
		"SELECT id, name, repo_url, created_at, updated_at FROM apps WHERE id = $1",
		id,
	).Scan(&app.ID, &app.Name, &app.RepoURL, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Store) List() ([]*App, error) {
	rows, err := s.db.Query("SELECT id, name, repo_url, created_at, updated_at FROM apps ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		var app App
		if err := rows.Scan(&app.ID, &app.Name, &app.RepoURL, &app.CreatedAt, &app.UpdatedAt); err != nil {
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
       SELECT id, user_id, name, slug, status, url, repo_url, branch, created_at, updated_at
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

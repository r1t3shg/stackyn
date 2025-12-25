// Package envvars provides data models and database operations for environment variables.
// Environment variables are key-value pairs associated with apps that get injected
// into running containers.
package envvars

import (
	"database/sql"
	"time"
)

// EnvVar represents a single environment variable for an app.
type EnvVar struct {
	ID        int       `json:"id"`
	AppID     int       `json:"app_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store provides database operations for the EnvVar model.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store instance with the provided database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create creates or updates an environment variable for an app.
// If a variable with the same key already exists for the app, it updates it.
func (s *Store) Create(appID int, key, value string) (*EnvVar, error) {
	var envVar EnvVar
	err := s.db.QueryRow(
		`INSERT INTO environment_variables (app_id, key, value, updated_at)
		 VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		 ON CONFLICT (app_id, key) 
		 DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
		 RETURNING id, app_id, key, value, created_at, updated_at`,
		appID, key, value,
	).Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &envVar.CreatedAt, &envVar.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &envVar, nil
}

// GetByAppID retrieves all environment variables for a specific app.
func (s *Store) GetByAppID(appID int) ([]*EnvVar, error) {
	rows, err := s.db.Query(
		"SELECT id, app_id, key, value, created_at, updated_at FROM environment_variables WHERE app_id = $1 ORDER BY key ASC",
		appID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var envVars []*EnvVar
	for rows.Next() {
		var envVar EnvVar
		if err := rows.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
			return nil, err
		}
		envVars = append(envVars, &envVar)
	}
	return envVars, rows.Err()
}

// Delete removes an environment variable by app ID and key.
func (s *Store) Delete(appID int, key string) error {
	_, err := s.db.Exec(
		"DELETE FROM environment_variables WHERE app_id = $1 AND key = $2",
		appID, key,
	)
	return err
}

// DeleteByAppID removes all environment variables for an app.
func (s *Store) DeleteByAppID(appID int) error {
	_, err := s.db.Exec(
		"DELETE FROM environment_variables WHERE app_id = $1",
		appID,
	)
	return err
}

// GetAsMap returns all environment variables for an app as a map[string]string.
// This is convenient for passing to Docker container configuration.
func (s *Store) GetAsMap(appID int) (map[string]string, error) {
	envVars, err := s.GetByAppID(appID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, envVar := range envVars {
		result[envVar.Key] = envVar.Value
	}
	return result, nil
}


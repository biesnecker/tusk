package config

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func getConfigDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, ".local", "share")
	case "linux":
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			baseDir = xdgData
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			baseDir = filepath.Join(home, ".local", "share")
		}
	case "windows":
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	configDir := filepath.Join(baseDir, "tusk")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return configDir, nil
}

func NewStore() (*Store, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	dbPath := filepath.Join(configDir, "tusk.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initDB(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS post_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		status_id TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Migrate old last_post table if it exists
	var tableName string
	err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='last_post'").Scan(&tableName)
	if err == nil {
		// Migrate data from old table
		var statusID string
		err := s.db.QueryRow("SELECT status_id FROM last_post WHERE id = 1").Scan(&statusID)
		if err == nil && statusID != "" {
			s.db.Exec("INSERT OR IGNORE INTO post_history (status_id) VALUES (?)", statusID)
		}
		s.db.Exec("DROP TABLE last_post")
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Set(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}

func (s *Store) Get(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *Store) Delete(key string) error {
	_, err := s.db.Exec("DELETE FROM config WHERE key = ?", key)
	return err
}

func (s *Store) AddPostToHistory(statusID string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO post_history (status_id, created_at) VALUES (?, CURRENT_TIMESTAMP)",
		statusID,
	)
	return err
}

func (s *Store) GetLastPostID() (string, error) {
	var statusID string
	err := s.db.QueryRow(
		"SELECT status_id FROM post_history ORDER BY id DESC LIMIT 1",
	).Scan(&statusID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return statusID, err
}

func (s *Store) RemovePostFromHistory(statusID string) error {
	_, err := s.db.Exec("DELETE FROM post_history WHERE status_id = ?", statusID)
	return err
}

func (s *Store) ClearPostHistory() error {
	_, err := s.db.Exec("DELETE FROM post_history")
	return err
}

func (s *Store) ClearAll() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM config"); err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM post_history"); err != nil {
		return err
	}

	return tx.Commit()
}

package config

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDatabase(cfg *Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.GetDSN())
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sqlx.DB) error {
	entries, err := os.ReadDir("migration")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		content, readErr := os.ReadFile(filepath.Join("migration", name))
		if readErr != nil {
			return readErr
		}
		if _, execErr := db.Exec(string(content)); execErr != nil {
			return execErr
		}
	}

	return nil
}

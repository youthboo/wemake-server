package config

import (
	"fmt"
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
	// Prevent concurrent app instances from running DDL at the same time.
	// Concurrent startup migrations can deadlock with each other and with live queries.
	const migrationLockKey int64 = 2026042501
	if _, err := db.Exec(`SELECT pg_advisory_lock($1)`, migrationLockKey); err != nil {
		return err
	}
	defer func() {
		_, _ = db.Exec(`SELECT pg_advisory_unlock($1)`, migrationLockKey)
	}()

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
			return fmt.Errorf("migration %s failed: %w", name, execErr)
		}
	}

	return nil
}

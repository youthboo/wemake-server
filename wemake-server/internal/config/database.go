package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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

	if err := ensureSchemaMigrations(db); err != nil {
		return err
	}
	applied, err := loadAppliedMigrations(db)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		baselined, err := baselineExistingSchema(db, files)
		if err != nil {
			return err
		}
		applied = baselined
	}

	for _, name := range files {
		if applied[name] {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join("migration", name))
		if readErr != nil {
			return readErr
		}
		if _, execErr := db.Exec(string(content)); execErr != nil {
			return fmt.Errorf("migration %s failed: %w", name, execErr)
		}
		if _, err := db.Exec(`
			INSERT INTO schema_migrations (filename)
			VALUES ($1)
			ON CONFLICT (filename) DO NOTHING
		`, name); err != nil {
			return fmt.Errorf("record migration %s failed: %w", name, err)
		}
	}

	return nil
}

func ensureSchemaMigrations(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func loadAppliedMigrations(db *sqlx.DB) (map[string]bool, error) {
	rows := []string{}
	if err := db.Select(&rows, `SELECT filename FROM schema_migrations`); err != nil {
		return nil, err
	}
	applied := make(map[string]bool, len(rows))
	for _, name := range rows {
		applied[name] = true
	}
	return applied, nil
}

func baselineExistingSchema(db *sqlx.DB, files []string) (map[string]bool, error) {
	applied := map[string]bool{}
	exists, err := tableExists(db, "users")
	if err != nil {
		return nil, err
	}
	if !exists {
		return applied, nil
	}

	// Older versions of the app ran every SQL file on every startup and did not
	// persist migration history. If an application schema already exists but the
	// history table is empty, record the pre-history migrations as applied so
	// startup does not replay destructive/non-idempotent DDL. Keep migration 040+
	// runnable because this runner was added with the factory config assignment
	// work and those migrations may still need to be applied.
	for _, name := range files {
		if migrationNumber(name) >= 40 {
			continue
		}
		if _, err := db.Exec(`
			INSERT INTO schema_migrations (filename)
			VALUES ($1)
			ON CONFLICT (filename) DO NOTHING
		`, name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, nil
}

func tableExists(db *sqlx.DB, table string) (bool, error) {
	var exists bool
	err := db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = $1
		)
	`, table)
	return exists, err
}

func migrationNumber(name string) int {
	prefix := name
	if idx := strings.Index(prefix, "_"); idx >= 0 {
		prefix = prefix[:idx]
	}
	n, err := strconv.Atoi(prefix)
	if err != nil {
		return 0
	}
	return n
}

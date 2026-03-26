package migrations

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Apply executes embedded SQL migrations from the provided filesystem in
// filename order and records applied versions in schema_migrations.
func Apply(ctx context.Context, db *sql.DB, migrationFS fs.FS) error {
	if db == nil || migrationFS == nil {
		return nil
	}

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		);
	`); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		applied, err := alreadyApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			return err
		}
		sqlText := strings.TrimSpace(string(body))
		if sqlText == "" {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, sqlText); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schema_migrations (version, applied_at)
			VALUES (?, ?)
		`, name, time.Now().UTC().UnixNano()); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func alreadyApplied(ctx context.Context, db *sql.DB, version string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM schema_migrations
		WHERE version = ?
	`, version).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

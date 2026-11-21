package dbmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func ApplyMigrations(ctx context.Context, db *sql.DB, log *slog.Logger, migrationsDir string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	files, err := migrationFiles(migrationsDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no migrations found in %s", migrationsDir)
	}

	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now'))
);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	if log != nil {
		log.Info("database migrations started", "dir", migrationsDir, "count", len(files))
	}
	for _, file := range files {
		if err := applyMigration(ctx, db, log, file); err != nil {
			return err
		}
	}
	if log != nil {
		log.Info("database migrations ready")
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, log *slog.Logger, file string) error {
	version, err := migrationVersion(file)
	if err != nil {
		return err
	}

	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, version).Scan(&exists); err != nil {
		return fmt.Errorf("check migration %d: %w", version, err)
	}
	if exists {
		return nil
	}

	content, err := os.ReadFile(file) //nolint:gosec // migration files are loaded from the configured migrations directory.
	if err != nil {
		return fmt.Errorf("read migration %s: %w", file, err)
	}
	upSQL, err := gooseUpSection(string(content))
	if err != nil {
		return fmt.Errorf("parse migration %s: %w", file, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", version, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, upSQL); err != nil {
		return fmt.Errorf("apply migration %s: %w", file, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", version, err)
	}
	committed = true

	if log != nil {
		log.Info("database migration applied", "version", version, "file", filepath.Base(file))
	}
	return nil
}

func migrationFiles(dir string) ([]string, error) {
	dir = resolveMigrationDir(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		if _, err := migrationVersion(entry.Name()); err != nil {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func resolveMigrationDir(dir string) string {
	if dir == "" {
		dir = "migrations"
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	if filepath.IsAbs(dir) {
		return dir
	}

	searchRoots := make([]string, 0, 2)
	if wd, err := os.Getwd(); err == nil {
		searchRoots = append(searchRoots, wd)
	}
	if exe, err := os.Executable(); err == nil {
		searchRoots = append(searchRoots, filepath.Dir(exe))
	}

	for _, root := range searchRoots {
		current := root
		for {
			candidate := filepath.Join(current, dir)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}

	return dir
}

func migrationVersion(path string) (int64, error) {
	base := filepath.Base(path)
	index := strings.IndexByte(base, '_')
	if index <= 0 {
		return 0, fmt.Errorf("invalid migration filename %q", base)
	}
	version, err := strconv.ParseInt(base[:index], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version %q: %w", base, err)
	}
	return version, nil
}

func gooseUpSection(content string) (string, error) {
	const upMarker = "-- +goose Up"
	const downMarker = "-- +goose Down"

	upStart := strings.Index(content, upMarker)
	if upStart < 0 {
		return "", fmt.Errorf("migration missing %q marker", upMarker)
	}
	upStart += len(upMarker)

	downStart := strings.Index(content[upStart:], downMarker)
	if downStart < 0 {
		return "", fmt.Errorf("migration missing %q marker", downMarker)
	}

	upSQL := strings.TrimSpace(content[upStart : upStart+downStart])
	if upSQL == "" {
		return "", fmt.Errorf("migration up section is empty")
	}
	return upSQL, nil
}

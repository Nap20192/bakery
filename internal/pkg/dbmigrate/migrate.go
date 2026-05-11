package dbmigrate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyMigrations(ctx context.Context, db *pgxpool.Pool, log *slog.Logger, migrationsDir string) error {
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

	if _, err := db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version BIGINT PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT (now()::text)
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

func applyMigration(ctx context.Context, db *pgxpool.Pool, log *slog.Logger, file string) error {
	version, err := migrationVersion(file)
	if err != nil {
		return err
	}

	var exists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
		return fmt.Errorf("check migration %d: %w", version, err)
	}
	if exists {
		return nil
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", file, err)
	}
	upSQL, err := gooseUpSection(string(content))
	if err != nil {
		return fmt.Errorf("parse migration %s: %w", file, err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", version, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err := tx.Exec(ctx, upSQL); err != nil {
		return fmt.Errorf("apply migration %s: %w", file, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %d: %w", version, err)
	}
	committed = true

	if log != nil {
		log.Info("database migration applied", "version", version, "file", filepath.Base(file))
	}
	return nil
}

func migrationFiles(dir string) ([]string, error) {
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

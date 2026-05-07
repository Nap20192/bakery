package dbmigrate

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const initialMigrationPath = "migrations/00001_init.sql"

func ApplyInitialSchema(db *sql.DB, log *slog.Logger) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	content, err := os.ReadFile(initialMigrationPath)
	if err != nil {
		return fmt.Errorf("read initial migration: %w", err)
	}

	upSQL, err := gooseUpSection(string(content))
	if err != nil {
		return err
	}

	if _, err := db.Exec(upSQL); err != nil {
		return fmt.Errorf("apply initial schema: %w", err)
	}

	if log != nil {
		log.Info("database schema ready", "migration", initialMigrationPath)
	}
	return nil
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

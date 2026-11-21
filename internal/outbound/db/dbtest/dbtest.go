package dbtest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/pkg/dbmigrate"
)

func Open(t testing.TB) *sql.DB {
	t.Helper()
	ctx := context.Background()
	conn, err := outbounddb.OpenSQLite(ctx, filepath.Join(t.TempDir(), "bakery.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := dbmigrate.ApplyMigrations(ctx, conn, nil, "migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return conn
}

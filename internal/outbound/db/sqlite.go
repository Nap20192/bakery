package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"modernc.org/sqlite"
)

const sqliteParams = "_pragma=foreign_keys(1)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=busy_timeout(5000)" +
	"&_txlock=immediate" +
	"&_time_format=sqlite" +
	"&_timezone=UTC"

func init() {
	sqlite.MustRegisterDeterministicScalarFunction("lower", 1, unicodeLower)
}

func unicodeLower(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	switch v := args[0].(type) {
	case nil:
		return nil, nil
	case string:
		return strings.ToLower(v), nil
	case []byte:
		return strings.ToLower(string(v)), nil
	default:
		return v, nil
	}
}

func OpenSQLite(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?"+sqliteParams)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	slog.InfoContext(ctx, "sqlite opened", "path", path)
	return db, nil
}

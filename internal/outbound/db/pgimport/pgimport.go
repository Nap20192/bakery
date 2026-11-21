package pgimport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/pkg/dbmigrate"

	"github.com/jackc/pgx/v5"
)

var tables = []string{
	"departments",
	"order_categories",
	"auth_users",
	"iiko_products",
	"iiko_sync_runs",
	"iiko_assembly_charts",
	"iiko_assembly_chart_items",
	"iiko_prepared_charts",
	"iiko_prepared_chart_items",
	"dish_groups",
	"dish_catalog",
	"orders",
	"order_counters",
	"order_items",
	"order_history",
	"order_history_items",
	"production_sheets",
	"production_sheet_items",
	"production_sheet_orders",
	"production_sheet_loads",
	"order_drafts",
	"order_outbox",
}

var clearSeeded = []string{"DELETE FROM departments", "DELETE FROM order_categories"}

var sourceExpressions = map[string]string{
	"orders.comments":                "comments::text",
	"orders.fulfillment_date":        "fulfillment_date::text",
	"order_drafts.comments":          "comments::text",
	"order_drafts.items":             "items::text",
	"order_drafts.fulfillment_date":  "fulfillment_date::text",
	"order_categories.monitor_codes": "to_json(monitor_codes)::text",
	"order_outbox.payload":           "payload::text",
}

func Import(ctx context.Context, from, to, migrationsDir string) error {
	if _, err := os.Stat(to); err == nil {
		return fmt.Errorf("target %s already exists", to)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat target: %w", err)
	}
	staging := to + ".import"
	removeStaging(staging)
	if err := copyInto(ctx, from, staging, migrationsDir); err != nil {
		removeStaging(staging)
		return err
	}
	if err := os.Rename(staging, to); err != nil {
		removeStaging(staging)
		return fmt.Errorf("move imported database into place: %w", err)
	}
	slog.InfoContext(ctx, "postgres import complete", "target", to)
	return nil
}

func removeStaging(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
}

func copyInto(ctx context.Context, from, to, migrationsDir string) error {
	source, err := pgx.Connect(ctx, from)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer func() { _ = source.Close(ctx) }()
	snapshot, err := source.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("begin postgres snapshot: %w", err)
	}
	defer func() { _ = snapshot.Rollback(ctx) }()

	target, err := outbounddb.OpenSQLite(ctx, to)
	if err != nil {
		return err
	}
	defer func() { _ = target.Close() }()
	if err := dbmigrate.ApplyMigrations(ctx, target, slog.Default(), migrationsDir); err != nil {
		return err
	}

	tx, err := target.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sqlite tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, statement := range clearSeeded {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("%s: %w", statement, err)
		}
	}
	for _, table := range tables {
		copied, err := copyTable(ctx, snapshot, tx, table)
		if err != nil {
			return err
		}
		if err := verifyCount(ctx, snapshot, tx, table, copied); err != nil {
			return err
		}
		if err := carrySequence(ctx, snapshot, tx, table); err != nil {
			return err
		}
		slog.Info("table copied", "table", table, "rows", copied)
	}
	if err := checkForeignKeys(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite tx: %w", err)
	}
	if _, err := target.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return fmt.Errorf("checkpoint sqlite: %w", err)
	}
	return nil
}

func copyTable(ctx context.Context, source pgx.Tx, tx *sql.Tx, table string) (int64, error) {
	columns, err := targetColumns(ctx, tx, table)
	if err != nil {
		return 0, err
	}
	selects := make([]string, len(columns))
	for i, column := range columns {
		if expr, ok := sourceExpressions[table+"."+column]; ok {
			selects[i] = expr
		} else {
			selects[i] = column
		}
	}
	rows, err := source.Query(ctx, fmt.Sprintf("SELECT %s FROM %s", strings.Join(selects, ", "), table))
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", table, err)
	}
	defer rows.Close()

	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", ")
	insert, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ", "), placeholders))
	if err != nil {
		return 0, fmt.Errorf("prepare insert %s: %w", table, err)
	}
	defer func() { _ = insert.Close() }()

	var copied int64
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return 0, fmt.Errorf("decode %s row: %w", table, err)
		}
		if _, err := insert.ExecContext(ctx, values...); err != nil {
			return 0, fmt.Errorf("insert %s row %v: %w", table, values[0], err)
		}
		copied++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read %s: %w", table, err)
	}
	return copied, nil
}

func targetColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT name FROM pragma_table_info(?) ORDER BY cid", table)
	if err != nil {
		return nil, fmt.Errorf("columns of %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("columns of %s: %w", table, err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("columns of %s: %w", table, err)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table %s not found in sqlite schema", table)
	}
	return columns, nil
}

func verifyCount(ctx context.Context, source pgx.Tx, tx *sql.Tx, table string, copied int64) error {
	var sourceCount, targetCount int64
	if err := source.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&sourceCount); err != nil {
		return fmt.Errorf("count source %s: %w", table, err)
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&targetCount); err != nil {
		return fmt.Errorf("count target %s: %w", table, err)
	}
	if sourceCount != targetCount || copied != targetCount {
		return fmt.Errorf("row count mismatch in %s: postgres=%d copied=%d sqlite=%d", table, sourceCount, copied, targetCount)
	}
	return nil
}

func carrySequence(ctx context.Context, source pgx.Tx, tx *sql.Tx, table string) error {
	var autoincrement bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? AND sql LIKE '%AUTOINCREMENT%')`,
		table,
	).Scan(&autoincrement); err != nil {
		return fmt.Errorf("inspect %s: %w", table, err)
	}
	if !autoincrement {
		return nil
	}
	var position int64
	err := source.QueryRow(ctx,
		`SELECT COALESCE(s.last_value, 0) FROM pg_sequences s
		 WHERE format('%I.%I', s.schemaname, s.sequencename) = pg_get_serial_sequence($1, 'id')`,
		table,
	).Scan(&position)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("sequence of %s.id not found in postgres", table)
	}
	if err != nil {
		return fmt.Errorf("read sequence of %s: %w", table, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO sqlite_sequence(name, seq) SELECT ?, 0 WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = ?)`,
		table, table,
	); err != nil {
		return fmt.Errorf("init sequence of %s: %w", table, err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sqlite_sequence SET seq = MAX(seq, ?) WHERE name = ?`, position, table); err != nil {
		return fmt.Errorf("carry sequence of %s: %w", table, err)
	}
	return nil
}

func checkForeignKeys(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("foreign key check: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		var table, parent string
		var rowID sql.NullInt64
		var fkID int64
		if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			return fmt.Errorf("foreign key check: %w", err)
		}
		return fmt.Errorf("foreign key violation: %s row %v references missing %s", table, rowID.Int64, parent)
	}
	return rows.Err()
}

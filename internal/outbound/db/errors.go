package db

import (
	"errors"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func IsUniqueViolation(err error, table string, columns ...string) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	switch sqliteErr.Code() {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
	default:
		return false
	}
	qualified := make([]string, len(columns))
	for i, column := range columns {
		qualified[i] = table + "." + column
	}
	want := "UNIQUE constraint failed: " + strings.Join(qualified, ", ")
	msg := sqliteErr.Error()
	idx := strings.Index(msg, want)
	if idx < 0 {
		return false
	}
	rest := msg[idx+len(want):]
	return rest == "" || strings.HasPrefix(rest, " (")
}

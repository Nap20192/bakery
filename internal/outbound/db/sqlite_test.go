package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestLowerFoldsCyrillic(t *testing.T) {
	conn := openTestDB(t)
	var got string
	if err := conn.QueryRow(`SELECT lower('ХЛЕБ Бородинский')`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "хлеб бородинский" {
		t.Fatalf("lower = %q", got)
	}
	var match bool
	if err := conn.QueryRow(`SELECT lower('Булочка с МАКОМ') LIKE '%' || lower('маком') || '%'`).Scan(&match); err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Fatal("cyrillic case-insensitive LIKE did not match")
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	conn := openTestDB(t)
	if _, err := conn.Exec(`
CREATE TABLE parent (id INTEGER PRIMARY KEY);
CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER NOT NULL REFERENCES parent(id));`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO child (parent_id) VALUES (42)`); err == nil {
		t.Fatal("expected foreign key violation")
	}
}

func TestTimesStoredAsComparableUTCText(t *testing.T) {
	conn := openTestDB(t)
	if _, err := conn.Exec(`CREATE TABLE ts (at DATETIME NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	almaty := time.FixedZone("UTC+5", 5*60*60)
	at := time.Date(2026, 11, 21, 3, 4, 5, 0, almaty)
	if _, err := conn.Exec(`INSERT INTO ts (at) VALUES (?)`, at); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := conn.QueryRow(`SELECT CAST(at AS TEXT) FROM ts`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if raw != "2026-11-20 22:04:05+00:00" {
		t.Fatalf("stored time = %q", raw)
	}
	var got time.Time
	if err := conn.QueryRow(`SELECT at FROM ts`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(at) {
		t.Fatalf("round trip = %v, want %v", got, at)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	conn := openTestDB(t)
	if _, err := conn.Exec(`
CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT NOT NULL, login TEXT);
CREATE UNIQUE INDEX idx_users_username ON users(username) WHERE username <> '';
CREATE TABLE pairs (a INTEGER NOT NULL, b INTEGER NOT NULL UNIQUE, PRIMARY KEY (a, b));
INSERT INTO users (username) VALUES ('alice');
INSERT INTO pairs (a, b) VALUES (1, 2);`); err != nil {
		t.Fatal(err)
	}
	_, err := conn.Exec(`INSERT INTO users (username) VALUES ('alice')`)
	if !IsUniqueViolation(err, "users", "username") {
		t.Fatalf("expected users.username violation, got %v", err)
	}
	if IsUniqueViolation(err, "users", "user") {
		t.Fatal("matched a column-name prefix")
	}
	_, err = conn.Exec(`INSERT INTO pairs (a, b) VALUES (3, 2)`)
	if !IsUniqueViolation(err, "pairs", "b") {
		t.Fatalf("expected pairs.b violation, got %v", err)
	}
	_, err = conn.Exec(`INSERT INTO pairs (a, b) VALUES (1, 2)`)
	if !IsUniqueViolation(err, "pairs", "a", "b") {
		t.Fatalf("expected pairs primary key violation, got %v", err)
	}
	if IsUniqueViolation(err, "pairs", "b") {
		t.Fatal("primary key violation matched single-column constraint")
	}
}

package dbmigrate

import (
	"os"
	"path/filepath"
	"testing"
)

// tempDir returns t.TempDir() with symlinks resolved, so paths compare equal
// to those derived from os.Getwd() (macOS symlinks /var to /private/var).
func tempDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	return root
}

func TestResolveMigrationDirFindsParentDirectory(t *testing.T) {
	root := tempDir(t)
	migrationsDir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o750); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}

	nestedDir := filepath.Join(root, "cmd", "worker")
	if err := os.MkdirAll(nestedDir, 0o750); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(nestedDir); err != nil {
		t.Fatalf("chdir nested dir: %v", err)
	}

	got := resolveMigrationDir("migrations")
	if got != migrationsDir {
		t.Fatalf("resolveMigrationDir() = %q, want %q", got, migrationsDir)
	}
}

func TestMigrationFilesUsesResolvedDirectory(t *testing.T) {
	root := tempDir(t)
	migrationsDir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o750); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}

	file := filepath.Join(migrationsDir, "00001_init.sql")
	content := []byte("-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 1;\n")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	nestedDir := filepath.Join(root, "cmd", "worker")
	if err := os.MkdirAll(nestedDir, 0o750); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(nestedDir); err != nil {
		t.Fatalf("chdir nested dir: %v", err)
	}

	files, err := migrationFiles("migrations")
	if err != nil {
		t.Fatalf("migrationFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("migrationFiles() returned %d files, want 1", len(files))
	}
	if files[0] != file {
		t.Fatalf("migrationFiles()[0] = %q, want %q", files[0], file)
	}
}

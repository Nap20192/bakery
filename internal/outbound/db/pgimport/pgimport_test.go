package pgimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestImportRefusesExistingTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "bakery.db")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Import(context.Background(), "postgres://127.0.0.1:1/none", target, "../../../../migrations"); err == nil {
		t.Fatal("expected error for existing target")
	}
	data, err := os.ReadFile(filepath.Clean(target))
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing target changed: %q, %v", data, err)
	}
}

func TestImportFailureLeavesNoDatabaseFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bakery.db")
	if err := Import(context.Background(), "postgres://127.0.0.1:1/none?connect_timeout=1", target, "../../../../migrations"); err == nil {
		t.Fatal("expected error for unreachable source")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("files left after failed import: %v", entries)
	}
}

package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain ensures a pristine fixture directory for every test run: the
// engine persists tables in <file>.db / <file>.db.metadata, and leftover
// files from earlier runs make schema-creating tests fail spuriously.
func TestMain(m *testing.M) {
	matches, _ := filepath.Glob("test_*.db*")
	for _, p := range matches {
		os.Remove(p)
	}
	os.Remove("wedb.log")
	os.Remove("temp_check.txt")
	code := m.Run()
	// best-effort cleanup of artifacts created during this run
	for _, p := range matches {
		os.Remove(p)
	}
	os.Exit(code)
}

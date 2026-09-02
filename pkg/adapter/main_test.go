package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain removes leftover database fixtures so every run starts clean.
func TestMain(m *testing.M) {
	patterns := []string{"*.db", "*.db.metadata", "*.db-journal", "wedb.log"}
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		for _, f := range matches {
			os.Remove(f)
		}
	}
	code := m.Run()
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		for _, f := range matches {
			os.Remove(f)
		}
	}
	os.Exit(code)
}

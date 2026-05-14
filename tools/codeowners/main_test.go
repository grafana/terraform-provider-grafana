package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGoldenFile runs the full generator against the real repo and verifies
// the output matches the checked-in .github/CODEOWNERS file.
func TestGoldenFile(t *testing.T) {
	// Find repository root (walk up from test file)
	root := findRepoRoot(t)

	// Run the check
	err := run(root, true)
	if err != nil {
		t.Fatalf("CODEOWNERS is out of date: %v\nRegenerate with: make codeowners", err)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	// Start from the current directory and walk up looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}

package grafana

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ensures NDJSON debug instrumentation can write when an explicit log path is set (no live Grafana).
func TestUnitOrgPrefsDebugNDJSONWritesFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.ndjson")
	t.Setenv("TF_ACC", "1")
	t.Setenv("GRAFANA_ORG_PREFS_DEBUG_LOG", logPath)

	debugOrgPrefsNDJSON(context.Background(), "T", "resource_organization_preferences_internal_test.go", "unit probe", map[string]any{"ok": true})

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"hypothesisId":"T"`) {
		t.Fatalf("missing hypothesis in written log: %s", b)
	}
}

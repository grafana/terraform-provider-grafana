package main

import "testing"

func TestExtractTeamName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"group:default/platform-monitoring", "platform-monitoring"},
		{"group:default/alerting-squad", "alerting-squad"},
		{"", ""},
		{"platform-monitoring", "platform-monitoring"},
	}

	for _, tt := range tests {
		got := extractTeamName(tt.input)
		if got != tt.want {
			t.Errorf("extractTeamName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractPkgInfo(t *testing.T) {
	tests := []struct {
		dir     string
		wantDir string
		wantPkg string
	}{
		{"/abs/path/internal/resources/grafana", "internal/resources/grafana", "grafana"},
		{"internal/resources/cloud", "internal/resources/cloud", "cloud"},
		{"internal/resources/appplatform", "internal/resources/appplatform", "appplatform"},
		{"/some/other/path", "/some/other/path", "path"},
	}

	for _, tt := range tests {
		gotDir, gotPkg := extractPkgInfo(tt.dir)
		if gotDir != tt.wantDir || gotPkg != tt.wantPkg {
			t.Errorf("extractPkgInfo(%q) = (%q, %q), want (%q, %q)",
				tt.dir, gotDir, gotPkg, tt.wantDir, tt.wantPkg)
		}
	}
}

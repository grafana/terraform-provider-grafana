package main

import (
	"testing"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		relPath string
		pattern string
		want    bool
	}{
		// ** matches any file in directory and subdirectories
		{"internal/resources/slo/resource_slo.go", "/internal/resources/slo/**", true},
		{"internal/resources/slo/resource_slo_test.go", "/internal/resources/slo/**", true},
		// ** matches subdirectories
		{"internal/resources/appplatform/generic/generic_resource.go", "/internal/resources/appplatform/**", true},
		{"internal/resources/appplatform/generic/helpers.go", "/internal/resources/appplatform/**", true},
		// ** does not match outside the directory
		{"internal/resources/cloud/resource_stack.go", "/internal/resources/appplatform/**", false},
		// Specific prefix match (single *)
		{"internal/resources/grafana/resource_dashboard.go", "/internal/resources/grafana/resource_dashboard*", true},
		{"internal/resources/grafana/resource_dashboard_test.go", "/internal/resources/grafana/resource_dashboard*", true},
		{"internal/resources/grafana/resource_folder.go", "/internal/resources/grafana/resource_dashboard*", false},
		// Different directory
		{"internal/resources/cloud/resource_dashboard.go", "/internal/resources/grafana/resource_dashboard*", false},
		// Example directory patterns
		{"examples/resources/grafana_dashboard/resource.tf", "/examples/resources/grafana_dashboard/*", true},
		{"examples/data-sources/grafana_folder/data-source.tf", "/examples/data-sources/grafana_folder/*", true},
		// Doc file patterns (exact match, no wildcard)
		{"docs/resources/dashboard.md", "/docs/resources/dashboard.md", true},
		{"docs/resources/folder.md", "/docs/resources/dashboard.md", false},
	}

	for _, tt := range tests {
		got := matchPattern(tt.relPath, tt.pattern)
		if got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.relPath, tt.pattern, got, tt.want)
		}
	}
}

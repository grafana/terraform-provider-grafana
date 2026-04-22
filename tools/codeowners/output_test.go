package main

import (
	"strings"
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

func TestExtractSectionFromPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"/internal/resources/grafana/**", "internal/resources/grafana"},
		{"/internal/resources/grafana/resource_dashboard*", "internal/resources/grafana"},
		{"/internal/resources/appplatform/**", "internal/resources/appplatform"},
		{"/examples/resources/grafana_dashboard/*", "examples/resources"},
		{"/examples/data-sources/grafana_folder/*", "examples/data-sources"},
		{"/docs/resources/dashboard.md", "docs/resources"},
		{"/docs/data-sources/folder.md", "docs/data-sources"},
		{"/some/other/path", "/some/other/path"},
	}

	for _, tt := range tests {
		got := extractSectionFromPattern(tt.pattern)
		if got != tt.want {
			t.Errorf("extractSectionFromPattern(%q) = %q, want %q", tt.pattern, got, tt.want)
		}
	}
}

func TestFormatOutput(t *testing.T) {
	rules := []rule{
		{Pattern: "/internal/resources/slo/**", Team: "@grafana/slo-squad"},
		{Pattern: "/internal/resources/k6/**", Team: "@grafana/k6-cloud-provisioning"},
		{Pattern: "/examples/resources/grafana_slo/*", Team: "@grafana/slo-squad"},
		{Pattern: "/docs/resources/slo.md", Team: "@grafana/slo-squad"},
	}

	staticContent := "# Static rules\n/pkg/provider/** @grafana/platform-monitoring\n"

	output := formatOutput("platform-monitoring", staticContent, rules)

	if !strings.Contains(output, "* @grafana/platform-monitoring") {
		t.Error("output missing default rule")
	}
	if !strings.Contains(output, "# Static rules") {
		t.Error("output missing static content")
	}
	if !strings.Contains(output, "/pkg/provider/** @grafana/platform-monitoring") {
		t.Error("output missing static rule")
	}
	if !strings.Contains(output, "# internal/resources/slo") {
		t.Error("output missing slo section comment")
	}
	if !strings.Contains(output, "/internal/resources/slo/**") {
		t.Error("output missing slo rule")
	}
	if !strings.Contains(output, "# examples/resources") {
		t.Error("output missing examples section comment")
	}
	if !strings.Contains(output, "/examples/resources/grafana_slo/*") {
		t.Error("output missing example rule")
	}
	if !strings.Contains(output, "# docs/resources") {
		t.Error("output missing docs section comment")
	}
	if !strings.Contains(output, "/docs/resources/slo.md") {
		t.Error("output missing doc rule")
	}
	if !strings.Contains(output, "Auto-generated") {
		t.Error("output missing header comment")
	}
	if !strings.Contains(output, "CODEOWNERS.in") {
		t.Error("output missing CODEOWNERS.in reference")
	}

	// Verify ordering: static content appears between default rule and generated rules
	defaultIdx := strings.Index(output, "* @grafana/platform-monitoring")
	staticIdx := strings.Index(output, "# Static rules")
	generatedIdx := strings.Index(output, "# internal/resources/slo")
	if defaultIdx >= staticIdx || staticIdx >= generatedIdx {
		t.Errorf("wrong ordering: default(%d) static(%d) generated(%d)", defaultIdx, staticIdx, generatedIdx)
	}
}

func TestFormatOutput_NoStatic(t *testing.T) {
	rules := []rule{
		{Pattern: "/internal/resources/slo/**", Team: "@grafana/slo-squad"},
	}

	output := formatOutput("platform-monitoring", "", rules)

	if !strings.Contains(output, "* @grafana/platform-monitoring") {
		t.Error("output missing default rule")
	}
	if !strings.Contains(output, "/internal/resources/slo/**") {
		t.Error("output missing slo rule")
	}
}

func TestParseStaticPatterns(t *testing.T) {
	content := `# Comment
/pkg/provider/** @grafana/platform-monitoring
/internal/common/** @grafana/platform-monitoring

# Another section
/scripts/** @grafana/platform-monitoring
`
	patterns := parseStaticPatterns(content)
	if len(patterns) != 3 {
		t.Fatalf("expected 3 patterns, got %d: %v", len(patterns), patterns)
	}
	if patterns[0] != "/pkg/provider/**" {
		t.Errorf("patterns[0] = %q, want /pkg/provider/**", patterns[0])
	}
	if patterns[1] != "/internal/common/**" {
		t.Errorf("patterns[1] = %q, want /internal/common/**", patterns[1])
	}
	if patterns[2] != "/scripts/**" {
		t.Errorf("patterns[2] = %q, want /scripts/**", patterns[2])
	}
}

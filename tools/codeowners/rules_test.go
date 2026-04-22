package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePackageRules_MultiOwner(t *testing.T) {
	pg := &packageGroup{
		pkgDir:  "internal/resources/grafana",
		pkgName: "grafana",
		comps: []component{
			{TFName: "grafana_dashboard", Owner: "dashboards-squad", SourceFiles: []string{"internal/resources/grafana/resource_dashboard.go"}},
			{TFName: "grafana_folder", Owner: "search-squad", SourceFiles: []string{"internal/resources/grafana/resource_folder.go"}},
			{TFName: "grafana_role", Owner: "access-squad", SourceFiles: []string{"internal/resources/grafana/resource_role.go"}},
			{TFName: "grafana_role_assignment", Owner: "access-squad", SourceFiles: []string{"internal/resources/grafana/resource_role_assignment.go"}},
			{TFName: "grafana_organization", Owner: "access-squad", SourceFiles: []string{"internal/resources/grafana/resource_organization.go"}},
		},
	}

	rules := generatePackageRules(pg)

	// First rule should be the wildcard for the majority owner (access-squad)
	if rules[0].Pattern != "/internal/resources/grafana/**" {
		t.Errorf("first rule pattern = %q, want wildcard", rules[0].Pattern)
	}
	if rules[0].Team != "@grafana/access-squad" {
		t.Errorf("first rule team = %q, want @grafana/access-squad", rules[0].Team)
	}

	// Should have specific rules for minority owners
	found := map[string]bool{}
	for _, r := range rules[1:] {
		found[r.Pattern] = true
	}
	if !found["/internal/resources/grafana/resource_dashboard*"] {
		t.Error("missing specific rule for resource_dashboard")
	}
	if !found["/internal/resources/grafana/resource_folder*"] {
		t.Error("missing specific rule for resource_folder")
	}
	// access-squad resources should NOT have specific rules (they're the majority)
	if found["/internal/resources/grafana/resource_role*"] {
		t.Error("majority owner should not have specific rules")
	}
}

func TestGenerateExampleAndDocRules(t *testing.T) {
	// Create a temp directory with example and doc paths
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "examples", "resources", "grafana_dashboard"), 0o755)
	os.MkdirAll(filepath.Join(root, "examples", "data-sources", "grafana_folder"), 0o755)
	os.MkdirAll(filepath.Join(root, "docs", "resources"), 0o755)
	os.MkdirAll(filepath.Join(root, "docs", "data-sources"), 0o755)
	os.WriteFile(filepath.Join(root, "docs", "resources", "dashboard.md"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "docs", "data-sources", "folder.md"), []byte("x"), 0o644)

	comps := []component{
		{TFName: "grafana_dashboard", Type: "terraform-resource", Owner: "dashboards-squad"},
		{TFName: "grafana_folder", Type: "terraform-data-source", Owner: "search-squad"},
		{TFName: "grafana_missing", Type: "terraform-resource", Owner: "nobody"}, // no example/doc on disk
		{TFName: "grafana_empty_owner", Type: "terraform-resource", Owner: ""},   // empty owner skipped
	}

	rules := generateExampleAndDocRules(root, comps)

	patterns := map[string]string{}
	for _, r := range rules {
		patterns[r.Pattern] = r.Team
	}

	// Example rules
	if team, ok := patterns["/examples/resources/grafana_dashboard/*"]; !ok || team != "@grafana/dashboards-squad" {
		t.Errorf("expected example rule for grafana_dashboard, got %v", patterns)
	}
	if team, ok := patterns["/examples/data-sources/grafana_folder/*"]; !ok || team != "@grafana/search-squad" {
		t.Errorf("expected example rule for grafana_folder, got %v", patterns)
	}

	// Doc rules
	if team, ok := patterns["/docs/resources/dashboard.md"]; !ok || team != "@grafana/dashboards-squad" {
		t.Errorf("expected doc rule for dashboard.md, got %v", patterns)
	}
	if team, ok := patterns["/docs/data-sources/folder.md"]; !ok || team != "@grafana/search-squad" {
		t.Errorf("expected doc rule for folder.md, got %v", patterns)
	}

	// Missing paths should NOT have rules
	if _, ok := patterns["/examples/resources/grafana_missing/*"]; ok {
		t.Error("should not emit rule for non-existent example dir")
	}
	if _, ok := patterns["/docs/resources/missing.md"]; ok {
		t.Error("should not emit rule for non-existent doc file")
	}
}

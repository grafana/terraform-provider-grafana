package cloud

import (
	"strings"
	"testing"
)

func TestUnitDecideStateVersion(t *testing.T) {
	assigned := func(target string) *managedPluginStatus {
		return &managedPluginStatus{Enrolled: true, Strategy: managedStrategyAssigned, TargetVersion: target}
	}
	aligned := &managedPluginStatus{Enrolled: true, Strategy: managedStrategyMajorAligned}
	const assignedWarning = "grafana-athena-datasource on mystack: version is managed by Grafana Cloud"
	const alignedWarning = "grafana-athena-datasource on mystack: minor and patch versions are managed by Grafana Cloud"

	for name, tt := range map[string]struct {
		prior, installed, catalog string
		managed                   *managedPluginStatus
		wantState                 string
		wantWarning               string
	}{
		"no managed field uses today's rule for a pin": {
			prior: "3.3.8", installed: "3.3.9", wantState: "3.3.9",
		},
		"no managed field keeps latest at the catalog latest": {
			prior: "latest", installed: "3.3.9", catalog: "3.3.9", wantState: "latest",
		},
		"not enrolled uses today's rule": {
			prior: "3.3.8", installed: "3.3.9", managed: &managedPluginStatus{}, wantState: "3.3.9",
		},
		"assigned pin is kept once the plugin is at the target": {
			prior: "3.3.8", installed: "3.3.9", managed: assigned("3.3.9"),
			wantState: "3.3.8", wantWarning: assignedWarning,
		},
		"assigned pin before the sync acts has no diff": {
			prior: "3.3.8", installed: "3.3.8", managed: assigned("3.3.9"), wantState: "3.3.8",
		},
		"assigned pin changed outside Terraform shows a diff": {
			prior: "3.3.8", installed: "3.3.5", managed: assigned("3.3.9"), wantState: "3.3.5",
		},
		"assigned latest is kept at a target that lags the catalog": {
			prior: "latest", installed: "2.9.2", catalog: "2.9.3", managed: assigned("2.9.2"), wantState: "latest",
		},
		"assigned latest upgrades when the sync is not acting": {
			prior: "latest", installed: "2.9.1", catalog: "2.9.3", managed: assigned("2.9.2"), wantState: "2.9.1",
		},
		"assigned with an unknown target uses today's rule": {
			prior: "3.3.8", installed: "3.3.9", managed: assigned(""), wantState: "3.3.9",
		},
		"an unknown strategy uses today's rule": {
			prior: "3.3.8", installed: "3.3.9", managed: &managedPluginStatus{Enrolled: true, TargetVersion: "3.3.9"}, wantState: "3.3.9",
		},
		"state written by today's provider has no diff and no warning": {
			prior: "3.3.9", installed: "3.3.9", managed: assigned("3.3.9"), wantState: "3.3.9",
		},
		"major-aligned pin inside the installed major is kept": {
			prior: "2.0.0", installed: "2.1.2", managed: aligned,
			wantState: "2.0.0", wantWarning: alignedWarning,
		},
		"major-aligned pin in another major shows a diff": {
			prior: "3.0.0", installed: "2.1.2", managed: aligned, wantState: "2.1.2",
		},
		"major-aligned latest is kept inside the catalog's major": {
			prior: "latest", installed: "2.1.2", catalog: "2.1.3", managed: aligned, wantState: "latest",
		},
		"major-aligned latest upgrades across majors": {
			prior: "latest", installed: "2.1.2", catalog: "3.0.0", managed: aligned, wantState: "2.1.2",
		},
		"pre-release versions compare by major": {
			prior: "0.2.3-preview", installed: "0.2.4-preview", managed: aligned,
			wantState: "0.2.3-preview", wantWarning: alignedWarning,
		},
		"empty prior state after an import uses today's rule": {
			prior: "", installed: "3.3.9", managed: assigned("3.3.9"), wantState: "3.3.9",
		},
		"a prior value that is not a version uses today's rule": {
			prior: "not-a-version", installed: "3.3.9", managed: assigned("3.3.9"), wantState: "3.3.9",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := decideStateVersion("mystack", "grafana-athena-datasource", tt.prior, tt.installed, tt.catalog, tt.managed)
			if got.StateVersion != tt.wantState {
				t.Errorf("state version: got %q, want %q", got.StateVersion, tt.wantState)
			}
			switch {
			case tt.wantWarning == "" && got.Warning != nil:
				t.Errorf("unexpected warning: %q", got.Warning.Summary)
			case tt.wantWarning != "" && got.Warning == nil:
				t.Errorf("want warning %q, got none", tt.wantWarning)
			case tt.wantWarning != "" && got.Warning.Summary != tt.wantWarning:
				t.Errorf("warning summary: got %q, want %q", got.Warning.Summary, tt.wantWarning)
			}
			if got.Warning != nil && !strings.HasPrefix(got.Warning.Summary, "grafana-athena-datasource on mystack:") {
				t.Errorf("warning summary %q does not start with the plugin and stack", got.Warning.Summary)
			}
		})
	}
}

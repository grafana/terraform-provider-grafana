package cloud

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

const testPluginPath = "/instances/mystack/plugins/grafana-athena-datasource"

func instancePluginBody(version, managed string) string {
	body := `{"instanceSlug":"mystack","pluginSlug":"grafana-athena-datasource","version":"` + version + `"`
	if managed != "" {
		body += `,"managed":` + managed
	}
	return body + `}`
}

func catalogRoute(version string) *stubRoute {
	return &stubRoute{
		match: func(r *http.Request) bool {
			return r.Method == http.MethodGet &&
				!strings.Contains(r.URL.Path, "/instances/") &&
				strings.HasSuffix(r.URL.Path, "/plugins/grafana-athena-datasource")
		},
		script: []stubResponse{{status: http.StatusOK, body: `{"version":"` + version + `"}`}},
	}
}

func priorState(t *testing.T, attributes map[string]string) *schema.ResourceData {
	t.Helper()
	attributes["id"] = "mystack:grafana-athena-datasource"
	return resourcePluginInstallation().Schema.Data(&terraform.InstanceState{
		ID:         "mystack:grafana-athena-datasource",
		Attributes: attributes,
	})
}

func warnings(diags diag.Diagnostics) []string {
	var summaries []string
	for _, d := range diags {
		if d.Severity == diag.Warning {
			summaries = append(summaries, d.Summary)
		}
	}
	return summaries
}

func readWith(t *testing.T, prior map[string]string, installed, managed, catalog string) (*schema.ResourceData, diag.Diagnostics) {
	t.Helper()
	routes := []*stubRoute{{
		match:  methodContains(http.MethodGet, testPluginPath),
		script: []stubResponse{{status: http.StatusOK, body: instancePluginBody(installed, managed)}},
	}}
	if catalog != "" {
		routes = append(routes, catalogRoute(catalog))
	}
	stub := newStubbedGcomClient(t, routes...)
	d := priorState(t, prior)
	return d, resourcePluginInstallationRead(context.Background(), d, stub.client)
}

func TestUnitPluginInstallationRead(t *testing.T) {
	pinned := func(version string) map[string]string {
		return map[string]string{"stack_slug": "mystack", "slug": "grafana-athena-datasource", "version": version}
	}

	for name, tt := range map[string]struct {
		prior        map[string]string
		managed      string
		catalog      string
		wantVersion  string
		wantWarnings int
	}{
		"keeps the pin once the sync has acted": {
			prior:        pinned("3.3.8"),
			managed:      `{"enrolled":true,"strategy":"assigned","targetVersion":"3.3.9"}`,
			wantVersion:  "3.3.8",
			wantWarnings: 1,
		},
		"keeps a major-aligned pin inside the installed major": {
			prior:        pinned("3.3.0"),
			managed:      `{"enrolled":true,"strategy":"major-aligned","targetVersion":"3.3.9"}`,
			wantVersion:  "3.3.0",
			wantWarnings: 1,
		},
		"uses today's rule when the target is null": {
			prior:       pinned("3.3.8"),
			managed:     `{"enrolled":true,"strategy":"assigned","targetVersion":null}`,
			wantVersion: "3.3.9",
		},
		"uses today's rule when the strategy is null": {
			prior:       pinned("3.3.8"),
			managed:     `{"enrolled":true,"strategy":null,"targetVersion":"3.3.9"}`,
			wantVersion: "3.3.9",
		},
		"behaves as today without the managed field": {
			prior:       pinned("3.3.8"),
			wantVersion: "3.3.9",
		},
		"keeps latest at the catalog latest without the managed field": {
			prior:       pinned("latest"),
			catalog:     "3.3.9",
			wantVersion: "latest",
		},
		"sets installed_version on state written by today's provider": {
			prior:       pinned("3.3.9"),
			managed:     `{"enrolled":true,"strategy":"assigned","targetVersion":"3.3.9"}`,
			wantVersion: "3.3.9",
		},
		"uses today's rule after an import": {
			prior:       map[string]string{},
			managed:     `{"enrolled":true,"strategy":"assigned","targetVersion":"3.3.9"}`,
			wantVersion: "3.3.9",
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, diags := readWith(t, tt.prior, "3.3.9", tt.managed, tt.catalog)
			assertWantErr(t, diags, "")

			if got := d.Get("version").(string); got != tt.wantVersion {
				t.Errorf("version: got %q, want %q", got, tt.wantVersion)
			}
			if got := d.Get("installed_version").(string); got != "3.3.9" {
				t.Errorf("installed_version: got %q, want %q", got, "3.3.9")
			}
			got := warnings(diags)
			if len(got) != tt.wantWarnings {
				t.Fatalf("warnings: got %v, want %d", got, tt.wantWarnings)
			}
			for _, summary := range got {
				if !strings.HasPrefix(summary, "grafana-athena-datasource on mystack:") {
					t.Errorf("warning summary %q does not start with the plugin and stack", summary)
				}
			}
		})
	}
}

func bodyContains(method, path, want string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		if r.Method != method || !strings.Contains(r.URL.Path, path) {
			return false
		}
		raw, _ := io.ReadAll(r.Body)
		return strings.Contains(string(raw), want)
	}
}

func pluginState(version string) *terraform.InstanceState {
	return &terraform.InstanceState{
		ID: "mystack:grafana-athena-datasource",
		Attributes: map[string]string{
			"id":                "mystack:grafana-athena-datasource",
			"stack_slug":        "mystack",
			"slug":              "grafana-athena-datasource",
			"version":           version,
			"installed_version": version,
		},
	}
}

func pluginConfig(version string) *terraform.ResourceConfig {
	return terraform.NewResourceConfigRaw(map[string]any{
		"stack_slug": "mystack",
		"slug":       "grafana-athena-datasource",
		"version":    version,
	})
}

func updateData(t *testing.T, oldVersion, newVersion string) *schema.ResourceData {
	t.Helper()
	diff := &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{
		"version": {Old: oldVersion, New: newVersion},
	}}
	d, err := schema.InternalMap(resourcePluginInstallation().Schema.Schema).Data(pluginState(oldVersion), diff)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestUnitPluginInstallationCreateSetsInstalledVersion(t *testing.T) {
	stub := newStubbedGcomClient(t, &stubRoute{
		match:  methodContains(http.MethodPost, "/instances/mystack/plugins"),
		script: []stubResponse{{status: http.StatusOK, body: instancePluginBody("3.3.8", "")}},
	})
	d := schema.TestResourceDataRaw(t, resourcePluginInstallation().Schema.Schema, map[string]any{
		"stack_slug": "mystack",
		"slug":       "grafana-athena-datasource",
		"version":    "3.3.8",
	})

	assertWantErr(t, resourcePluginInstallationCreate(context.Background(), d, stub.client), "")

	if got := d.Get("installed_version").(string); got != "3.3.8" {
		t.Errorf("installed_version: got %q, want %q", got, "3.3.8")
	}
}

func TestUnitPluginInstallationUpdate(t *testing.T) {
	for name, newVersion := range map[string]string{
		"sends a pin":                          "3.3.9",
		"sends latest when version is removed": "latest",
	} {
		t.Run(name, func(t *testing.T) {
			post := &stubRoute{
				match:  bodyContains(http.MethodPost, testPluginPath, `"version":"`+newVersion+`"`),
				script: []stubResponse{{status: http.StatusOK, body: instancePluginBody("3.3.9", "")}},
			}
			del := &stubRoute{match: methodContains(http.MethodDelete, testPluginPath), script: codes(http.StatusOK)}
			stub := newStubbedGcomClient(t, post, del)
			d := updateData(t, "3.3.8", newVersion)

			assertWantErr(t, resourcePluginInstallationUpdate(context.Background(), d, stub.client), "")

			if post.count != 1 {
				t.Errorf("POST with version %q: got %d calls, want 1", newVersion, post.count)
			}
			if del.count != 0 {
				t.Errorf("DELETE: got %d calls, want 0", del.count)
			}
			if got := d.Get("installed_version").(string); got != "3.3.9" {
				t.Errorf("installed_version: got %q, want %q", got, "3.3.9")
			}
		})
	}
}

func TestUnitPluginInstallationUpdateFailureKeepsState(t *testing.T) {
	stub := newStubbedGcomClient(t, &stubRoute{
		match:  methodContains(http.MethodPost, testPluginPath),
		script: []stubResponse{{status: http.StatusBadRequest, body: `{"message":"bad version"}`}},
	})
	d := updateData(t, "3.3.8", "3.3.9")

	diags := resourcePluginInstallationUpdate(context.Background(), d, stub.client)
	if !diags.HasError() {
		t.Fatal("want an error")
	}
	if got := d.State().Attributes["version"]; got != "3.3.8" {
		t.Errorf("state version after a failed update: got %q, want %q", got, "3.3.8")
	}
}

func TestUnitPluginInstallationDiffMarksInstalledVersionUnknown(t *testing.T) {
	diff, err := resourcePluginInstallation().Schema.Diff(context.Background(), pluginState("3.3.8"), pluginConfig("3.3.9"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff.RequiresNew() {
		t.Error("a version change must update in place, not replace")
	}
	installed, ok := diff.Attributes["installed_version"]
	if !ok || !installed.NewComputed {
		t.Errorf("installed_version must be unknown in the plan, got %+v", installed)
	}
}

func TestUnitPluginInstallationPlanIsEmptyAfterRefreshingOldState(t *testing.T) {
	d, diags := readWith(t,
		map[string]string{"stack_slug": "mystack", "slug": "grafana-athena-datasource", "version": "3.3.9"},
		"3.3.9",
		`{"enrolled":true,"strategy":"assigned","targetVersion":"3.3.9"}`,
		"",
	)
	assertWantErr(t, diags, "")

	diff, err := resourcePluginInstallation().Schema.Diff(context.Background(), d.State(), pluginConfig("3.3.9"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff != nil && len(diff.Attributes) > 0 {
		t.Errorf("want an empty plan after refresh, got %+v", diff.Attributes)
	}
}

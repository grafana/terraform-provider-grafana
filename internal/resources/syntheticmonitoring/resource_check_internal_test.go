package syntheticmonitoring

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// TestUnitCheck_secretManagerEnabled checks that the deprecated
// secret_manager_enabled is still sent to the API until the attribute is removed.
func TestUnitCheck_secretManagerEnabled(t *testing.T) {
	for _, tc := range []struct {
		name     string
		raw      map[string]any
		expected bool
	}{
		{
			name: "enabled",
			raw: map[string]any{
				"secret_manager_enabled": true,
				"bearer_token":           "${secrets.my-api-token}",
			},
			expected: true,
		},
		{
			name:     "default is false",
			raw:      map[string]any{},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]any{
				"job":    "test",
				"target": "https://example.com",
				"settings": []any{
					map[string]any{
						"http": []any{tc.raw},
					},
				},
			}

			d := schema.TestResourceDataRaw(t, resourceCheck().Schema.Schema, raw)

			check, err := makeCheck(d)
			if err != nil {
				t.Fatalf("makeCheck returned unexpected error: %v", err)
			}
			if check.Settings.Http == nil {
				t.Fatal("expected http settings to be set")
			}
			if got := check.Settings.Http.SecretManagerEnabled; got != tc.expected {
				t.Errorf("SecretManagerEnabled = %t, want %t", got, tc.expected)
			}
		})
	}
}

// secretManagerCheckRaw builds a minimal HTTP check whose bearer token holds a
// secret reference.
func secretManagerCheckRaw(httpExtra map[string]any) map[string]any {
	h := map[string]any{
		"bearer_token": "${secrets.my-api-token}",
		"ip_version":   "V4",
		"method":       "GET",
	}
	maps.Copy(h, httpExtra)
	return map[string]any{
		"job":      "test",
		"target":   "https://example.com",
		"probes":   []any{1},
		"settings": []any{map[string]any{"http": []any{h}}},
	}
}

// secretManagerCheckState builds the state of an applied check. It uses the
// SDK field writer so the set hashes match the provider's.
func secretManagerCheckState(t *testing.T, r *schema.Resource, enabled bool) *terraform.InstanceState {
	t.Helper()

	w := &schema.MapFieldWriter{Schema: r.Schema}
	fields := secretManagerCheckRaw(map[string]any{"secret_manager_enabled": enabled})
	fields["enabled"] = true
	fields["alert_sensitivity"] = "none"
	fields["basic_metrics_only"] = true
	fields["frequency"] = 60000
	fields["timeout"] = 3000
	for k, v := range fields {
		if err := w.WriteField([]string{k}, v); err != nil {
			t.Fatalf("WriteField(%s): %v", k, err)
		}
	}

	attrs := w.Map()
	attrs["id"] = "1234"
	attrs["channels.#"] = "0"
	return &terraform.InstanceState{ID: "1234", Attributes: attrs}
}

// TestUnitCheck_secretManagerEnabledNoPerpetualDiff checks that
// secret_manager_enabled does not plan a change on every run. When state and
// configuration agree, nothing is planned. When they differ, the plan takes the
// configured value.
//
// A DiffSuppressFunc fails this test, because the attribute sits in a set block.
func TestUnitCheck_secretManagerEnabledNoPerpetualDiff(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state bool
		// config is nil when the attribute is omitted.
		config *bool
		// wantPlanned is nil when no change is planned.
		wantPlanned *bool
	}{
		{name: "state and config agree on true", state: true, config: common.Ref(true)},
		{name: "state and config agree on false", state: false, config: common.Ref(false)},
		{name: "config omits, state false", state: false},
		{name: "config omits, state true from the API", state: true, wantPlanned: common.Ref(false)},
		{name: "config false, state true from the API", state: true, config: common.Ref(false), wantPlanned: common.Ref(false)},
		{name: "config true, state false from the API", state: false, config: common.Ref(true), wantPlanned: common.Ref(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceCheck().Schema

			httpExtra := map[string]any{}
			if tc.config != nil {
				httpExtra["secret_manager_enabled"] = *tc.config
			}
			cfg := terraform.NewResourceConfigRaw(secretManagerCheckRaw(httpExtra))

			diff, err := r.Diff(context.Background(), secretManagerCheckState(t, r, tc.state), cfg, nil)
			if err != nil {
				t.Fatalf("Diff returned unexpected error: %v", err)
			}

			if tc.wantPlanned == nil {
				assertNoPlannedChange(t, diff)
				return
			}
			assertPlannedSecretManagerEnabled(t, diff, *tc.wantPlanned)
		})
	}
}

// newSecretManagerCheckServer serves the check from secretManagerCheckState,
// with secret_manager_enabled set to apiEnabled.
func newSecretManagerCheckServer(t *testing.T, apiEnabled bool) *smapi.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/check/1234" {
			http.NotFound(w, req)
			return
		}
		_ = json.NewEncoder(w).Encode(sm.Check{
			Id:               1234,
			Job:              "test",
			Target:           "https://example.com",
			Frequency:        60000,
			Timeout:          3000,
			Enabled:          true,
			BasicMetricsOnly: true,
			AlertSensitivity: "none",
			Probes:           []int64{1},
			Settings: sm.CheckSettings{Http: &sm.HttpSettings{
				IpVersion:            sm.IpVersion_V4,
				Method:               sm.HttpMethod_GET,
				BearerToken:          "${secrets.my-api-token}",
				SecretManagerEnabled: apiEnabled,
			}},
		})
	}))
	t.Cleanup(srv.Close)

	return smapi.NewClient(srv.URL, "token", srv.Client())
}

// TestUnitCheck_secretManagerEnabledRefresh checks that Read keeps the value
// from state when the API returns a different one, so the next plan is empty.
func TestUnitCheck_secretManagerEnabledRefresh(t *testing.T) {
	for _, tc := range []struct {
		name       string
		state      bool
		apiEnabled bool
	}{
		{name: "state false, API true", state: false, apiEnabled: true},
		{name: "state true, API false", state: true, apiEnabled: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceCheck().Schema
			d := r.Data(secretManagerCheckState(t, r, tc.state))
			if diags := resourceCheckRead(context.Background(), d, newSecretManagerCheckServer(t, tc.apiEnabled)); diags.HasError() {
				t.Fatalf("Read returned unexpected error: %v", diags)
			}
			if got := secretManagerEnabledFromState(d); got != tc.state {
				t.Fatalf("Read wrote secret_manager_enabled = %t, want %t from state", got, tc.state)
			}

			cfg := terraform.NewResourceConfigRaw(secretManagerCheckRaw(map[string]any{"secret_manager_enabled": tc.state}))
			diff, err := r.Diff(context.Background(), d.State(), cfg, nil)
			if err != nil {
				t.Fatalf("Diff returned unexpected error: %v", err)
			}
			assertNoPlannedChange(t, diff)
		})
	}
}

// TestUnitCheck_secretManagerEnabledImport checks the import path. Read starts
// from empty state and must write false even when the API returns true. A
// configured true then plans one in-place update, and none after that.
func TestUnitCheck_secretManagerEnabledImport(t *testing.T) {
	client := newSecretManagerCheckServer(t, true)

	r := resourceCheck().Schema
	d := r.Data(&terraform.InstanceState{ID: "1234"})
	if diags := resourceCheckRead(context.Background(), d, client); diags.HasError() {
		t.Fatalf("Read returned unexpected error: %v", diags)
	}
	if got := secretManagerEnabledFromState(d); got {
		t.Fatalf("Read wrote secret_manager_enabled = %t from the API, want false", got)
	}

	cfg := terraform.NewResourceConfigRaw(secretManagerCheckRaw(map[string]any{"secret_manager_enabled": true}))
	diff, err := r.Diff(context.Background(), d.State(), cfg, nil)
	if err != nil {
		t.Fatalf("Diff returned unexpected error: %v", err)
	}
	if diff.RequiresNew() {
		t.Error("expected an in-place update, got a replacement")
	}
	assertPlannedSecretManagerEnabled(t, diff, true)

	diff, err = r.Diff(context.Background(), secretManagerCheckState(t, r, true), cfg, nil)
	if err != nil {
		t.Fatalf("Diff returned unexpected error: %v", err)
	}
	assertNoPlannedChange(t, diff)
}

// TestUnitCheck_secretManagerEnabledFromState checks that the value in state is
// returned, and false when it is missing.
func TestUnitCheck_secretManagerEnabledFromState(t *testing.T) {
	for _, tc := range []struct {
		name     string
		raw      map[string]any
		expected bool
	}{
		{name: "enabled", raw: secretManagerCheckRaw(map[string]any{"secret_manager_enabled": true}), expected: true},
		{name: "disabled", raw: secretManagerCheckRaw(map[string]any{"secret_manager_enabled": false}), expected: false},
		{name: "omitted", raw: secretManagerCheckRaw(nil), expected: false},
		{name: "no settings at all", raw: map[string]any{"job": "test", "target": "https://example.com"}, expected: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resourceCheck().Schema.Schema, tc.raw)
			if got := secretManagerEnabledFromState(d); got != tc.expected {
				t.Errorf("secretManagerEnabledFromState = %t, want %t", got, tc.expected)
			}
		})
	}
}

func assertNoPlannedChange(t *testing.T, diff *terraform.InstanceDiff) {
	t.Helper()
	if diff == nil {
		return
	}
	for k, attr := range diff.Attributes {
		if attr.Old != attr.New || attr.NewComputed {
			t.Errorf("expected no planned change, got %s: %q -> %q (computed=%t)",
				k, attr.Old, attr.New, attr.NewComputed)
		}
	}
}

// assertPlannedSecretManagerEnabled skips the removed set element, because a
// change inside a set block replaces the whole element.
func assertPlannedSecretManagerEnabled(t *testing.T, diff *terraform.InstanceDiff, want bool) {
	t.Helper()
	if diff == nil {
		t.Fatal("expected a plan while state and configuration disagree, got none")
	}
	for k, attr := range diff.Attributes {
		if !strings.HasSuffix(k, ".secret_manager_enabled") || attr.NewRemoved {
			continue
		}
		if attr.NewComputed {
			t.Fatal("expected the plan to resolve secret_manager_enabled to a known value")
		}
		if got := attr.New == "true"; got != want {
			t.Errorf("planned secret_manager_enabled = %t, want %t", got, want)
		}
		return
	}
	t.Fatal("expected the plan to set secret_manager_enabled, found no attribute")
}

package syntheticmonitoring

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestUnitCheck_secretManagerEnabled checks that makeCheck still copies the
// deprecated secret_manager_enabled into the API request. The API ignores it,
// but the provider keeps sending it until a later release removes the
// attribute.
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

// TestUnitCheck_secretManagerEnabledRefresh covers a refresh where the API
// reports a different secret_manager_enabled than state. Read must keep the
// state value, so the next plan against the same configuration is empty. An
// imported check has no prior state, so Read stores false.
func TestUnitCheck_secretManagerEnabledRefresh(t *testing.T) {
	for _, tc := range []struct {
		name       string
		imported   bool
		state      bool
		apiEnabled bool
	}{
		{name: "state false, API true", state: false, apiEnabled: true},
		{name: "state true, API false", state: true, apiEnabled: false},
		{name: "imported, API true", imported: true, apiEnabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceCheck().Schema
			st := &terraform.InstanceState{ID: "1234"}
			if !tc.imported {
				st = secretManagerCheckState(t, r, tc.state)
			}

			d := r.Data(st)
			if diags := resourceCheckRead(context.Background(), d, newSecretManagerCheckServer(t, tc.apiEnabled)); diags.HasError() {
				t.Fatalf("Read returned unexpected error: %v", diags)
			}
			if got := secretManagerEnabledFromState(d); got != tc.state {
				t.Fatalf("Read wrote secret_manager_enabled = %t, want %t", got, tc.state)
			}
			// Read leaves channels unset on import, so the plan always shows it
			// as computed.
			if tc.imported {
				return
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

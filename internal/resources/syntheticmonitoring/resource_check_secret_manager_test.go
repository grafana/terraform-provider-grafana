package syntheticmonitoring

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// secretManagerCheckRaw builds a minimal HTTP check whose bearer token holds a
// secret reference.
func secretManagerCheckRaw(httpExtra map[string]any) map[string]any {
	http := map[string]any{
		"bearer_token": "${secrets.my-api-token}",
		"ip_version":   "V4",
		"method":       "GET",
	}
	for k, v := range httpExtra {
		http[k] = v
	}
	return map[string]any{
		"job":      "test",
		"target":   "https://example.com",
		"probes":   []any{1},
		"settings": []any{map[string]any{"http": []any{http}}},
	}
}

// secretManagerCheckState builds the state an applied check carries. It writes
// through the SDK's own field writer so the set hashes match the ones the
// provider produces at runtime.
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

// TestUnitCheck_secretManagerEnabledNoPerpetualDiff checks that the deprecated
// secret_manager_enabled never leaves a plan that cannot settle. Where state and
// configuration agree there is no plan at all; where they disagree the plan
// resolves to the configured value, so the read after the apply settles.
//
// A DiffSuppressFunc does not pass this test: the attribute sits inside a set
// block, and suppression runs too late to stop the block being re-keyed.
func TestUnitCheck_secretManagerEnabledNoPerpetualDiff(t *testing.T) {
	for _, tc := range []struct {
		name string
		// state is the value secret_manager_enabled carries in state.
		state bool
		// config is the value the configuration sets, or nil when the
		// configuration omits the attribute entirely.
		config *bool
		// wantConverged is true when state and configuration already agree and
		// the check must plan no change at all.
		wantConverged bool
		// wantPlanned is the value the plan must resolve the attribute to when
		// state and configuration disagree.
		wantPlanned bool
	}{
		{name: "state and config agree on true", state: true, config: boolPtr(true), wantConverged: true},
		{name: "state and config agree on false", state: false, config: boolPtr(false), wantConverged: true},
		{name: "config omits, state false", state: false, config: nil, wantConverged: true},
		{name: "config omits, state true from the API", state: true, config: nil, wantPlanned: false},
		{name: "config false, state true from the API", state: true, config: boolPtr(false), wantPlanned: false},
		{name: "config true, state false from the API", state: false, config: boolPtr(true), wantPlanned: true},
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

			if tc.wantConverged {
				if diff == nil {
					return
				}
				for k, attr := range diff.Attributes {
					if attr.Old != attr.New || attr.NewComputed {
						t.Errorf("expected no planned change, got %s: %q -> %q (computed=%t)",
							k, attr.Old, attr.New, attr.NewComputed)
					}
				}
				return
			}

			if diff == nil {
				t.Fatal("expected a plan while state and configuration disagree, got none")
			}

			planned, ok := plannedSecretManagerEnabled(diff)
			if !ok {
				t.Fatal("expected the plan to resolve secret_manager_enabled to a known value")
			}
			if planned != tc.wantPlanned {
				t.Errorf("planned secret_manager_enabled = %t, want %t", planned, tc.wantPlanned)
			}
		})
	}
}

// TestUnitCheck_secretManagerEnabledFromState covers the read path that keeps
// state in step with the configuration.
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

// plannedSecretManagerEnabled reports the value the plan resolves
// secret_manager_enabled to, ignoring the set element the plan removes. A value
// still unknown at plan time reports false for ok, because Read cannot keep a
// value Terraform does not have yet.
func plannedSecretManagerEnabled(diff *terraform.InstanceDiff) (bool, bool) {
	for k, attr := range diff.Attributes {
		if !strings.HasSuffix(k, ".secret_manager_enabled") || attr.NewRemoved {
			continue
		}
		if attr.NewComputed {
			return false, false
		}
		return attr.New == "true", true
	}
	return false, false
}

func boolPtr(b bool) *bool { return &b }

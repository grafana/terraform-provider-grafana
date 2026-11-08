package syntheticmonitoring

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestUnitCheck_secretManagerEnabled verifies that the http block's
// secret_manager_enabled attribute survives the write path (schema ->
// sm.HttpSettings), including the interesting non-default (true) value.
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

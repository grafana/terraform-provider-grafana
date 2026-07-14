package oncall

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestGetRouteEscalationChainID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		set   bool
		want  string
	}{
		{name: "unset", set: false, want: ""},
		{name: "null", value: nil, set: true, want: ""},
		{name: "empty", value: "", set: true, want: ""},
		{name: "id", value: "EC123", set: true, want: "EC123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
				"escalation_chain_id": {Type: schema.TypeString, Optional: true},
			}, map[string]any{})
			if tc.set {
				if err := d.Set("escalation_chain_id", tc.value); err != nil {
					t.Fatalf("Set: %v", err)
				}
			}
			if got := getRouteEscalationChainID(d); got != tc.want {
				t.Fatalf("getRouteEscalationChainID() = %q, want %q", got, tc.want)
			}
		})
	}
}

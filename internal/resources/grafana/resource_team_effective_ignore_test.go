package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEffectiveIgnoreExternallySyncedMembers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   types.Bool
		want bool
	}{
		{"null matches legacy absent state", types.BoolNull(), true},
		{"unknown uses schema default", types.BoolUnknown(), true},
		{"explicit false", types.BoolValue(false), false},
		{"explicit true", types.BoolValue(true), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := effectiveIgnoreExternallySyncedMembers(tc.in); got != tc.want {
				t.Fatalf("effectiveIgnoreExternallySyncedMembers(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

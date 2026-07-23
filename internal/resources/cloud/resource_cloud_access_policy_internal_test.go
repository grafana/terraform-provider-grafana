package cloud

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestUnitRealmsEqualIgnoringOrder(t *testing.T) {
	labelPolicy := func(selectors ...string) *schema.Set {
		items := make([]any, len(selectors))
		for i, s := range selectors {
			items[i] = map[string]any{"selector": s}
		}
		return schema.NewSet(func(v any) int {
			return schema.HashString(v.(map[string]any)["selector"])
		}, items)
	}
	realm := func(typ, id string, selectors ...string) map[string]any {
		return map[string]any{"type": typ, "identifier": id, "label_policy": labelPolicy(selectors...)}
	}

	for name, tc := range map[string]struct {
		a, b []any
		want bool
	}{
		"same realms in a different order are equal": {
			a:    []any{realm("stack", "1"), realm("org", "a")},
			b:    []any{realm("org", "a"), realm("stack", "1")},
			want: true,
		},
		"reordered label_policy selectors are equal": {
			a:    []any{realm("stack", "1", "{a=\"1\"}", "{b=\"2\"}")},
			b:    []any{realm("stack", "1", "{b=\"2\"}", "{a=\"1\"}")},
			want: true,
		},
		"different length is not equal": {
			a:    []any{realm("org", "a")},
			b:    []any{realm("org", "a"), realm("stack", "1")},
			want: false,
		},
		"different identifier is not equal": {
			a:    []any{realm("stack", "1")},
			b:    []any{realm("stack", "2")},
			want: false,
		},
		"different label_policy selector is not equal": {
			a:    []any{realm("stack", "1", "{a=\"1\"}")},
			b:    []any{realm("stack", "1", "{a=\"2\"}")},
			want: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := realmsEqualIgnoringOrder(tc.a, tc.b); got != tc.want {
				t.Errorf("realmsEqualIgnoringOrder() = %v, want %v", got, tc.want)
			}
		})
	}
}

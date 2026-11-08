package cloud

import "testing"

func TestUnitIsPDCSigningPolicy(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		scopes []string
		want   bool
	}{
		{
			name:   "new PDC scope only",
			scopes: []string{"set:pdc-signing"},
			want:   true,
		},
		{
			name:   "old PDC scope only",
			scopes: []string{"pdc-signing:write"},
			want:   true,
		},
		{
			name:   "both PDC scopes",
			scopes: []string{"pdc-signing:write", "set:pdc-signing"},
			want:   true,
		},
		{
			name:   "PDC scope alongside unrelated scopes",
			scopes: []string{"metrics:read", "set:pdc-signing", "logs:write"},
			want:   true,
		},
		{
			name:   "no PDC scopes",
			scopes: []string{"metrics:read", "logs:write"},
			want:   false,
		},
		{
			name:   "empty scopes",
			scopes: nil,
			want:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isPDCSigningPolicy(tc.scopes); got != tc.want {
				t.Errorf("isPDCSigningPolicy(%v) = %v, want %v", tc.scopes, got, tc.want)
			}
		})
	}
}

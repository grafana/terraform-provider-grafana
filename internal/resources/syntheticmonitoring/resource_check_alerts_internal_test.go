package syntheticmonitoring

import (
	"errors"
	"testing"
)

func TestIsCheckNotFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			// The exact error the SM API client returns when the parent check has
			// already been deleted, as observed from crossplane-provider-grafana.
			name: "api message for a deleted check",
			err:  errors.New(`check alerts update request: status="404 Not Found", msg="invalid updated check", err="check not found"`),
			want: true,
		},
		{
			name: "bare 404",
			err:  errors.New("404 Not Found"),
			want: true,
		},
		{
			name: "bare check not found",
			err:  errors.New("check not found"),
			want: true,
		},
		{
			name: "unrelated client error",
			err:  errors.New(`check alerts update request: status="400 Bad Request", msg="invalid updated check", err="invalid check probes"`),
			want: false,
		},
		{
			name: "server error",
			err:  errors.New("500 Internal Server Error"),
			want: false,
		},
		{
			name: "auth error",
			err:  errors.New("401 Unauthorized"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isCheckNotFound(tc.err); got != tc.want {
				t.Fatalf("isCheckNotFound(%q) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

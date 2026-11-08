package asserts

import "testing"

// matchRule is a small helper to build the map shape that Terraform produces
// for a single "match" block element.
func matchRule(op string, values ...string) map[string]interface{} {
	vals := make([]interface{}, 0, len(values))
	for _, v := range values {
		vals = append(vals, v)
	}
	return map[string]interface{}{
		"property": "environment",
		"op":       op,
		"values":   vals,
	}
}

func TestUnitValidateMatchRules(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		matches   []interface{}
		wantError bool
	}{
		{
			name:    "nil matches",
			matches: nil,
		},
		{
			name:    "empty matches",
			matches: []interface{}{},
		},
		{
			name:    "comparison op with values",
			matches: []interface{}{matchRule("=", "production")},
		},
		{
			name:    "comparison op with multiple values",
			matches: []interface{}{matchRule("CONTAINS", "prod", "staging")},
		},
		{
			name:      "comparison op without values",
			matches:   []interface{}{matchRule("=")},
			wantError: true,
		},
		{
			name:    "IS NOT NULL without values",
			matches: []interface{}{matchRule("IS NOT NULL")},
		},
		{
			name:    "IS NULL without values",
			matches: []interface{}{matchRule("IS NULL")},
		},
		{
			name:      "IS NOT NULL with values",
			matches:   []interface{}{matchRule("IS NOT NULL", "production")},
			wantError: true,
		},
		{
			name:      "IS NULL with values",
			matches:   []interface{}{matchRule("IS NULL", "production")},
			wantError: true,
		},
		{
			name: "mixed valid rules",
			matches: []interface{}{
				matchRule("=", "production"),
				matchRule("IS NOT NULL"),
			},
		},
		{
			name: "second rule invalid",
			matches: []interface{}{
				matchRule("=", "production"),
				matchRule(">"),
			},
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateMatchRules(tc.matches)
			if tc.wantError && err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

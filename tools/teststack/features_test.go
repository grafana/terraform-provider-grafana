package main

import (
	"testing"
)

func TestParseFeatures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "basic only by default",
			input: "",
			want:  []string{"basic"},
		},
		{
			name:  "single feature implies basic",
			input: "k6",
			want:  []string{"basic", "k6"},
		},
		{
			name:  "comma list with whitespace",
			input: " basic , k6 , sm ",
			want:  []string{"basic", "k6", "sm"},
		},
		{
			name:    "rejects unknown feature",
			input:   "k6,bogus",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFeatures(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, want := range tc.want {
				if !got[want] {
					t.Errorf("expected %q to be enabled; got %v", want, got)
				}
			}
		})
	}
}

func TestStackSlugRegex(t *testing.T) {
	for _, tc := range []struct {
		slug string
		want bool
	}{
		{"tftest27606266952appplatform", true},
		{"tftest27606266952syntheticmonit", true}, // exactly 29 chars
		{"abc", true},
		// invalid: starts with a digit
		{"27606266952foo", false},
		// invalid: contains a hyphen
		{"tf-foo", false},
		// invalid: contains uppercase
		{"tfFoo", false},
		// invalid: single letter (regex requires >=2 chars)
		{"a", false},
		// invalid: empty
		{"", false},
	} {
		t.Run(tc.slug, func(t *testing.T) {
			got := stackSlugRegex.MatchString(tc.slug)
			if got != tc.want {
				t.Errorf("regex.MatchString(%q) = %v; want %v", tc.slug, got, tc.want)
			}
		})
	}
}

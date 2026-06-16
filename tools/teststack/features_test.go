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

func TestSMAPIURL(t *testing.T) {
	for _, tc := range []struct {
		region string
		want   string
	}{
		{"eu", "https://synthetic-monitoring-api-eu-west.grafana.net"},
		{"us", "https://synthetic-monitoring-api.grafana.net"},
		{"prod-us-central-0", "https://synthetic-monitoring-api-us-central-0.grafana.net"},
		{"prod-gb-south-0", "https://synthetic-monitoring-api-gb-south.grafana.net"},
	} {
		t.Run(tc.region, func(t *testing.T) {
			if got := smAPIURL(tc.region); got != tc.want {
				t.Errorf("smAPIURL(%q) = %q; want %q", tc.region, got, tc.want)
			}
		})
	}
}

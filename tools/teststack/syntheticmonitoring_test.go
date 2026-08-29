package main

import "testing"

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

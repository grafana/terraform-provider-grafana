package main

import (
	"context"
	"fmt"
	"strings"

	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
)

// smAPIURL returns the Synthetic Monitoring API URL for a given region slug,
// mirroring the table in
// internal/resources/cloud/resource_synthetic_monitoring_installation.go.
func smAPIURL(regionSlug string) string {
	exceptions := map[string]string{
		"au":              "https://synthetic-monitoring-api-au-southeast.grafana.net",
		"eu":              "https://synthetic-monitoring-api-eu-west.grafana.net",
		"prod-gb-south-0": "https://synthetic-monitoring-api-gb-south.grafana.net",
		"us":              "https://synthetic-monitoring-api.grafana.net",
		"us-azure":        "https://synthetic-monitoring-api-us-central-7.grafana.net",
	}
	if v, ok := exceptions[regionSlug]; ok {
		return v
	}
	return fmt.Sprintf("https://synthetic-monitoring-api-%s.grafana.net", strings.TrimPrefix(regionSlug, "prod-"))
}

// installSM installs Synthetic Monitoring on a stack and returns the SM access
// token plus the SM API URL.
//
// SM install needs the org-level CAP token (the "metrics publisher" token in
// the docs), so it lives in tools/teststack rather than in the SM test
// package.
func installSM(ctx context.Context, capToken string, info *stackInfo) (token, apiURL string, err error) {
	apiURL = smAPIURL(info.RegionSlug)
	client := smapi.NewClient(apiURL, "", nil)
	client.SetCustomClientID("teststack")
	client.SetCustomClientVersion("0.1")

	resp, err := client.Install(ctx, info.ID, info.HmInstancePromID, info.HlInstanceID, capToken)
	if err != nil {
		return "", "", fmt.Errorf("sm install: %w", err)
	}
	return resp.AccessToken, apiURL, nil
}

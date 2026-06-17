package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// installK6 calls the k6 install endpoint and returns the K6 access token and
// API URL. Mirrors the logic in
// internal/resources/cloud/resource_k6_installation.go.
//
// The k6 install needs an org-level CAP token (it talks directly to k6's
// global API at api.k6.io, not through the per-stack Grafana proxy), so it
// lives in tools/teststack rather than in the k6 test package.
func installK6(ctx context.Context, capToken string, info *stackInfo, cloudAPIBase string) (token, apiURL string, err error) {
	apiURL = "https://api.k6.io"
	url := fmt.Sprintf("%s/v3/account/grafana-app/start", apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("X-Stack-Id", fmt.Sprintf("%d", info.ID))
	req.Header.Set("X-Grafana-Key", capToken)
	req.Header.Set("X-Grafana-Service-Token", info.AdminSAToken)
	req.Header.Set("X-Grafana-User", "admin")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("k6 install request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("k6 install returned status %d", resp.StatusCode)
	}

	var out struct {
		V3GrafanaToken string `json:"v3_grafana_token"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", fmt.Errorf("decode k6 install response: %w", err)
	}
	if out.V3GrafanaToken == "" {
		return "", "", fmt.Errorf("k6 install response missing v3_grafana_token")
	}
	_ = cloudAPIBase
	return out.V3GrafanaToken, apiURL, nil
}

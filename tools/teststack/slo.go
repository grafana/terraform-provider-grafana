package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

// installSLO installs the grafana-slo-app plugin on a freshly-created
// stack and waits for its backend service to come up. Without this,
// SLO resource tests hit a steady stream of 503s on
// /api/plugins/grafana-slo-app/resources/v1/slo — the plugin proxy is
// up, but no plugin is registered at that slug yet.
//
// gcom returns 409 Conflict when the plugin is already installed, which
// we treat as success. After the install API call the plugin backend
// still needs a moment to start serving traffic; we poll the SLO
// resources endpoint until it returns anything other than 5xx before
// returning so the SLO tests don't have to absorb the warm-up time
// themselves.
func installSLO(ctx context.Context, client *gcom.APIClient, info *stackInfo) error {
	req := gcom.PostInstancePluginsRequest{
		Plugin: "grafana-slo-app",
	}
	installCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	_, _, err := client.InstancesAPI.PostInstancePlugins(installCtx, info.Slug).
		PostInstancePluginsRequest(req).
		XRequestId(requestID()).
		Execute()
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "conflict") {
		return fmt.Errorf("install grafana-slo-app plugin: %w", gcomErr(err))
	}

	// Wait for the plugin's backend to start answering. The list endpoint
	// (GET /api/plugins/grafana-slo-app/resources/v1/slo) is the same one
	// the SLO tests hit, so when this stops returning 5xx the tests can
	// proceed.
	readyCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	probeURL := strings.TrimSuffix(info.URL, "/") + "/api/plugins/grafana-slo-app/resources/v1/slo"
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return pollUntil(readyCtx, 5*time.Second, func(c context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(c, http.MethodGet, probeURL, nil)
		if err != nil {
			return false, err
		}
		req.Header.Set("Authorization", "Bearer "+info.AdminSAToken)
		resp, err := httpClient.Do(req)
		if err != nil {
			return false, err
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 500 {
			return false, fmt.Errorf("slo plugin status %d", resp.StatusCode)
		}
		return true, nil
	})
}

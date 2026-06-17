package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/grafana-com-public-clients/go/gcom"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
)

// known feature identifiers accepted by --features.
const (
	featureBasic        = "basic"
	featureK6           = "k6"
	featureSM           = "sm"
	featureOncall       = "oncall"
	featureFleet        = "fleet"
	featureAssertions   = "assertions"
	featureMLOSS        = "mloss"
	featureSLO          = "slo"
	featureIntegrations = "integrations"
)

// parseFeatures splits a comma-separated list and returns a set.
func parseFeatures(spec string) (map[string]bool, error) {
	out := map[string]bool{featureBasic: true}
	for _, raw := range strings.Split(spec, ",") {
		f := strings.TrimSpace(raw)
		if f == "" {
			continue
		}
		switch f {
		case featureBasic, featureK6, featureSM, featureOncall, featureFleet,
			featureAssertions, featureMLOSS, featureSLO, featureIntegrations:
			out[f] = true
		default:
			return nil, fmt.Errorf("unknown feature %q (allowed: basic,k6,sm,oncall,fleet,assertions,mloss,slo,integrations)", f)
		}
	}
	return out, nil
}

// installK6 calls the k6 install endpoint and returns the K6 access token and
// API URL. Mirrors the logic in resource_k6_installation.go.
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

// smAPIURL returns the Synthetic Monitoring API URL for a given region slug,
// mirroring the table in resource_synthetic_monitoring_installation.go.
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

// installFleet returns basic-auth credentials and a URL for Grafana Fleet
// Management. Fleet management is auto-provisioned with a stack; the auth
// format per the provider docs is `{fleet_management_user_id}:{cap_token}`
// (see docs/index.md). The user ID comes from the gcom "agent management"
// fields (legacy name for the same backend); the password is the org-level
// Cloud Access Policy token that authenticates the caller.
func installFleet(ctx context.Context, client *gcom.APIClient, capToken string, info *stackInfo) (auth, apiURL string, err error) {
	stack, _, err := client.InstancesAPI.GetInstance(ctx, info.Slug).Execute()
	if err != nil {
		return "", "", fmt.Errorf("get instance for fleet management URL: %w", gcomErr(err))
	}

	apiURL = strings.TrimSpace(stack.AgentManagementInstanceUrl)
	if apiURL == "" {
		return "", "", fmt.Errorf("stack %q has no fleet management URL configured", info.Slug)
	}

	fleetUserID := stack.AgentManagementInstanceId
	if fleetUserID == 0 {
		return "", "", fmt.Errorf("stack %q has no fleet management user id configured", info.Slug)
	}

	auth = fmt.Sprintf("%d:%s", int64(fleetUserID), capToken)
	return auth, apiURL, nil
}

// installAsserts performs the Asserts onboarding flow on a freshly-created
// stack. Without this, Asserts resource tests hit 403 Forbidden because the
// stack is in the `not_initialized` state. The flow mirrors what
// resource_stack.go does:
//
//  1. PUT /v2/stack — provision tokens
//  2. POST /v2/stack/datasets/auto-setup — auto-detect available datasets
//  3. POST /v2/stack/enable — enable the stack
//
// All three calls go through the Grafana plugin proxy on the stack itself,
// authenticated with the per-stack Admin SA token.
func installAsserts(ctx context.Context, capToken string, info *stackInfo) error {
	cfg := assertsapi.NewConfiguration()
	u, err := url.Parse(info.URL)
	if err != nil {
		return fmt.Errorf("parse stack URL %q: %w", info.URL, err)
	}
	cfg.Host = u.Host
	cfg.Scheme = u.Scheme
	cfg.Servers = assertsapi.ServerConfigurations{
		{
			URL: fmt.Sprintf("%s://%s/api/plugins/grafana-asserts-app/resources/asserts/api-server", u.Scheme, u.Host),
		},
	}
	cfg.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	if cfg.DefaultHeader == nil {
		cfg.DefaultHeader = make(map[string]string)
	}
	cfg.DefaultHeader["Authorization"] = "Bearer " + info.AdminSAToken
	cfg.DefaultHeader["Content-Type"] = "application/json"
	client := assertsapi.NewAPIClient(cfg)

	stackIDStr := fmt.Sprintf("%d", info.ID)

	// Step 1: provision tokens. Reuse the org-level CAP token for gcom /
	// mimir / detector, and the stack-scoped Admin SA token for the Grafana
	// token. This matches what resource_stack.go does when a user calls
	// `grafana_asserts_stack` with the same inputs.
	stackDto := assertsapi.NewStackDto()
	stackDto.SetGcomToken(capToken)
	stackDto.SetMimirToken(capToken)
	stackDto.SetAssertionDetectorToken(capToken)
	stackDto.SetGrafanaToken(info.AdminSAToken)
	if _, err := client.StackControllerAPI.PutV2Stack(ctx).
		StackDto(*stackDto).
		XScopeOrgID(stackIDStr).
		Execute(); err != nil {
		return fmt.Errorf("asserts PUT /v2/stack: %w", err)
	}

	// Step 2: auto-detect datasets. We don't pass dataset config explicitly;
	// the auto-setup endpoint inspects datasources on the stack and enables
	// whichever ones look healthy. On a fresh stack with no extra wiring
	// this is fine — Asserts just falls back to the no-dataset state.
	if _, _, err := client.StackControllerAPI.DetectAndAutoConfigureDatasets(ctx).
		XScopeOrgID(stackIDStr).
		Execute(); err != nil {
		return fmt.Errorf("asserts POST /v2/stack/datasets/auto-setup: %w", err)
	}

	// Step 3: enable. This may return 409 Conflict if sanity checks reject
	// the configuration (e.g. no datasources at all). We surface the error
	// so the shard fails loudly rather than running tests against a stack
	// that's still not_initialized.
	if _, _, err := client.StackControllerAPI.EnableV2Stack(ctx).
		XScopeOrgID(stackIDStr).
		Execute(); err != nil {
		return fmt.Errorf("asserts POST /v2/stack/enable: %w", err)
	}

	return nil
}

// getOnCallURL returns the per-stack OnCall API URL from gcom. OnCall is
// a regional service: the URL host varies by region (e.g.
// https://oncall-prod-eu-west-0.grafana.net), so we can't hard-code it.
// The provider requires `oncall_url` to be set before it builds the OnCall
// client, so the caller exports this value as GRAFANA_ONCALL_URL.
func getOnCallURL(ctx context.Context, client *gcom.APIClient, info *stackInfo) (string, error) {
	conn, _, err := client.InstancesAPI.GetConnections(ctx, info.Slug).Execute()
	if err != nil {
		return "", fmt.Errorf("get instance connections for OnCall URL: %w", gcomErr(err))
	}
	if conn.OncallApiUrl.IsSet() {
		if v := conn.OncallApiUrl.Get(); v != nil && *v != "" {
			return *v, nil
		}
	}
	return "", fmt.Errorf("stack %q has no OnCall API URL configured", info.Slug)
}

// installCloudIntegrations is a no-op stub: cloud integrations are enabled
// per-stack via the easystart plugin. Installation happens transparently when
// the tests call the integrations API, so no upfront provisioning is needed.
func installCloudIntegrations(ctx context.Context, info *stackInfo) error {
	_ = ctx
	_ = info
	return nil
}

// waitStackHealthy performs a lightweight GET / on the stack URL to confirm
// the Grafana microservices are up. New stacks frequently return 503/504 for
// tens of seconds after the gcom status flips to "active", so we poll until
// the stack actually responds.
func waitStackHealthy(ctx context.Context, info *stackInfo, timeout time.Duration) error {
	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := strings.TrimSuffix(info.URL, "/") + "/api/health"
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return pollUntil(healthCtx, 5*time.Second, func(c context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(c, http.MethodGet, url, nil)
		if err != nil {
			return false, err
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return false, err
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true, nil
		}
		return false, fmt.Errorf("stack health status %d", resp.StatusCode)
	})
}

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
)

// installAsserts performs the Asserts onboarding flow on a freshly-created
// stack. Without this, Asserts resource tests hit 403 Forbidden because the
// stack is in the `not_initialized` state. The flow mirrors what
// internal/resources/asserts/resource_stack.go does:
//
//  1. PUT  /v2/stack             — provision tokens
//  2. PUT  /product-activation   — activate the 'prometheus' product
//  3. PUT  /v2/stack/dataset     — configure a 'prometheus' dataset
//  4. POST /v2/stack/enable      — enable the stack
//
// We configure the prometheus dataset explicitly rather than relying on
// auto-setup because a freshly-created Grafana Cloud stack does not yet
// have any provisioned datasources visible to auto-detection, so the
// auto-setup endpoint finds nothing and the subsequent enable step
// fails the sanity-check with 422. A 'prometheus' dataset is always
// present on a Grafana Cloud stack (it's the stack's own Mimir tenant).
//
// PUT /v2/stack uses the org-level CAP token (it provisions backend
// tokens for gcom/mimir/detector), so this lives in tools/teststack rather
// than the asserts test package. The dataset/enable calls themselves
// could move into the asserts test package if we wanted, but keeping the
// full onboarding here keeps it as a single transaction.
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
		return fmt.Errorf("asserts PUT /v2/stack: %w", assertsErr(err))
	}

	// Step 2: activate the 'prometheus' product. Asserts gates dataset
	// configuration on the corresponding product being enabled — without
	// this step the dataset endpoint returns 422 "No Product Enabled for
	// Dataset: prometheus".
	prodDto := assertsapi.NewProductActivationDto("prometheus", true)
	if _, err := client.ProductActivationControllerAPI.UpsertProductActivation(ctx).
		ProductActivationDto(*prodDto).
		XScopeOrgID(stackIDStr).
		Execute(); err != nil {
		return fmt.Errorf("asserts UPSERT /product-activation (prometheus): %w", assertsErr(err))
	}

	// Step 3: configure a 'prometheus' dataset. Asserts ships with
	// 'kubernetes', 'otel', 'prometheus', and 'aws' as valid dataset
	// types; 'prometheus' is the only one guaranteed to exist on a
	// freshly-created Grafana Cloud stack (the stack's own Mimir).
	//
	// The dataset endpoint requires at least one filter group ("groups
	// Required" from the API), but rejects groups with both envName and
	// envLabel set ("Only one of envName or envLabel may be set"). We
	// use envName="prod" — a hardcoded env name that satisfies the API
	// without depending on any specific label being present on metrics.
	// (Tests in internal/resources/asserts/ don't depend on a particular
	// env name; they exercise the API/config, not the asserts pipeline.)
	datasetDto := assertsapi.NewStackDatasetDto("prometheus")
	filterGroup := *assertsapi.NewStackFilterGroupDto()
	filterGroup.SetEnvName("prod")
	datasetDto.FilterGroups = []assertsapi.StackFilterGroupDto{filterGroup}
	if _, _, err := client.StackControllerAPI.UpdateDataset(ctx).
		StackDatasetDto(*datasetDto).
		XScopeOrgID(stackIDStr).
		Execute(); err != nil {
		return fmt.Errorf("asserts PUT /v2/stack/dataset (prometheus): %w", assertsErr(err))
	}

	// Step 4: enable. With the prometheus product activated and dataset
	// configured the sanity check passes and the stack flips to
	// enabled=true.
	if _, _, err := client.StackControllerAPI.EnableV2Stack(ctx).
		XScopeOrgID(stackIDStr).
		Execute(); err != nil {
		return fmt.Errorf("asserts POST /v2/stack/enable: %w", assertsErr(err))
	}

	return nil
}

// assertsErr enriches an asserts API error with the response body, which is
// where the actual 422/409 detail lives. Mirrors gcomErr for the assertsapi
// generated client.
func assertsErr(err error) error {
	if err == nil {
		return nil
	}
	type bodyError interface {
		Body() []byte
	}
	if be, ok := err.(bodyError); ok && len(be.Body()) > 0 {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(be.Body())))
	}
	return err
}

package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/grafana-com-public-clients/go/gcom"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// This module calls plugin's /initialized endpoint to ensure the publisher token is propagated on creation.
// This is needed because terraform-provisioned stacks can't rely on app's user navigation, which is the
// regular UI-provisioning guarantee. In case of failure, we warn the user.

const pluginInitializedPath = "/api/plugins/k6-app/resources/cloud/v3/account/grafana-app/initialized"

func deliverPublisherToken(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) error {
	var stack *gcom.FormattedApiInstance
	if err := common.RetryRequest(ctx, "get stack instance", func() (*http.Response, error) {
		s, httpResp, execErr := cloudClient.InstancesAPI.GetInstance(ctx, d.Get("stack_id").(string)).Execute()
		stack = s
		return httpResp, execErr
	}); err != nil {
		return fmt.Errorf("could not resolve the stack URL: %w", err)
	}

	stackURL := strings.TrimSuffix(stack.Url, "/")
	saToken := d.Get("grafana_sa_token").(string)
	client := cloudClient.GetConfig().HTTPClient
	userAgent := cloudClient.GetConfig().UserAgent

	started := time.Now()
	state, err := forwardPublisherToken(ctx, client, stackURL, saToken, userAgent)
	if err != nil {
		return err
	}
	if state.Initialized != nil && !*state.Initialized {
		return errors.New(state.pendingReason())
	}
	tflog.Info(ctx, "k6 publisher token delivered", map[string]any{
		"stack_id":  d.Get("stack_id"),
		"confirmed": state.Initialized != nil,
		"elapsed":   time.Since(started).Round(time.Millisecond).String(),
	})
	return nil
}

type stackInitialization struct {
	Initialized           *bool `json:"initialized"`
	PublisherTokenPresent *bool `json:"publisher_token_present"`
	FoldersInitialized    *bool `json:"folders_initialized"`
	GrafanaRBACEnabled    *bool `json:"grafana_rbac_enabled"`
}

func (s stackInitialization) pendingReason() string {
	var reasons []string
	if isFalse(s.PublisherTokenPresent) {
		reasons = append(reasons, "the stack has no valid k6 publisher token")
	}
	if isTrue(s.GrafanaRBACEnabled) && isFalse(s.FoldersInitialized) {
		reasons = append(reasons, "the k6 App's Grafana folders are not set up")
	}
	if len(reasons) == 0 {
		return "the k6 App reports this stack as not initialized"
	}
	return strings.Join(reasons, ", and ")
}

func isTrue(b *bool) bool {
	return b != nil && *b
}

func isFalse(b *bool) bool {
	return b != nil && !*b
}

func forwardPublisherToken(ctx context.Context, client *http.Client, stackURL, saToken, userAgent string) (stackInitialization, error) {
	var result stackInitialization
	if err := getStack(ctx, client, stackURL+pluginInitializedPath, saToken, userAgent, &result); err != nil {
		return stackInitialization{}, err
	}
	return result, nil
}

func getStack(ctx context.Context, client *http.Client, url, saToken, userAgent string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+saToken)
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned %s: %s", url, resp.Status, string(body))
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func publisherTokenSyncWarning(err error) diag.Diagnostics {
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "could not confirm the k6 App finished initializing",
			Detail: "The k6 App is installed and usable, but test runs cannot publish metrics to " +
				"this stack until a user opens the k6 App.\n\nReason: " + err.Error(),
		},
	}
}

func syncPublisherToken(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	if err := deliverPublisherToken(ctx, d, cloudClient); err != nil {
		tflog.Warn(ctx, "could not deliver the k6 publisher token", map[string]any{
			"stack_id": d.Get("stack_id"),
			"error":    err.Error(),
		})
		return publisherTokenSyncWarning(err)
	}
	return nil
}

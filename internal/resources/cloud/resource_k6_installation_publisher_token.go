package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/grafana-com-public-clients/go/gcom"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// This module calls plugin's /initialized endpoint to ensure the publisher token is propagated on creation.
// This is needed because terraform-provisioned stacks can't rely on app's user navigation, which is the
// regular UI-provisioning guarantee. In case of failure, we warn the user.

const pluginInitializedPath = "/api/plugins/k6-app/resources/cloud/v3/account/grafana-app/initialized"

const initializedRetryTimeout = 30 * time.Second

func confirmStackInitialized(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient, timeout time.Duration) ([]string, error) {
	var stack *gcom.FormattedApiInstance
	if err := common.RetryRequest(ctx, "get stack instance", func() (*http.Response, error) {
		s, httpResp, execErr := cloudClient.InstancesAPI.GetInstance(ctx, d.Get("stack_id").(string)).Execute()
		stack = s
		return httpResp, execErr
	}); err != nil {
		return nil, fmt.Errorf("could not resolve the stack URL: %w", err)
	}

	stackURL := strings.TrimSuffix(stack.Url, "/")
	saToken := d.Get("grafana_sa_token").(string)
	client := cloudClient.GetConfig().HTTPClient
	userAgent := cloudClient.GetConfig().UserAgent

	started := time.Now()
	var (
		state   stackInitialization
		pending []string
	)
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		state, pending = stackInitialization{}, nil

		if err := getJSON(ctx, client, stackURL+pluginInitializedPath, saToken, userAgent, &state); err != nil {
			return retry.RetryableError(err)
		}
		if state.Initialized != nil && !*state.Initialized {
			pending = state.pending()
			return retry.RetryableError(fmt.Errorf("the k6 App is still setting up: %s", strings.Join(pending, "; ")))
		}
		return nil
	})
	switch {
	case len(pending) > 0:
		return pending, nil
	case err != nil:
		return nil, err
	}
	tflog.Info(ctx, "k6 App initialization confirmed", map[string]any{
		"stack_id":  d.Get("stack_id"),
		"confirmed": state.Initialized != nil,
		"elapsed":   time.Since(started).Round(time.Millisecond).String(),
	})
	return nil, nil
}

type stackInitialization struct {
	Initialized           *bool `json:"initialized"`
	PublisherTokenPresent bool  `json:"publisher_token_present"`
	FoldersInitialized    bool  `json:"folders_initialized"`
	GrafanaRBACEnabled    bool  `json:"grafana_rbac_enabled"`
}

func (s stackInitialization) pending() []string {
	var pending []string
	if !s.PublisherTokenPresent {
		pending = append(pending, "no valid k6 publisher token is stored for this stack, so test "+
			"runs cannot publish their metrics to it")
	}
	if s.GrafanaRBACEnabled && !s.FoldersInitialized {
		pending = append(pending, "the k6 App's Grafana folders are missing, so access to k6 "+
			"projects, which this stack grants through those folders' permissions, does not work")
	}
	if len(pending) == 0 {
		return []string{"the k6 App reports this stack as not set up, without saying what is missing"}
	}
	return pending
}

func getJSON(ctx context.Context, client *http.Client, url, saToken, userAgent string, out any) error {
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

func syncK6Initialization(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	return syncK6InitializationWithin(ctx, d, cloudClient, initializedRetryTimeout)
}

func syncK6InitializationWithin(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient, timeout time.Duration) diag.Diagnostics {
	pending, err := confirmStackInitialized(ctx, d, cloudClient, timeout)
	switch {
	case err != nil:
		tflog.Warn(ctx, "could not read the k6 App's setup state", map[string]any{
			"stack_id": d.Get("stack_id"),
			"error":    err.Error(),
		})
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "could not check whether the k6 App finished setting up",
				Detail: "The k6 App is installed, but Terraform could not read its setup state, so " +
					"parts of it may still be missing. Open the k6 App on this stack to check.\n\n" +
					"Reason: " + err.Error(),
			},
		}
	case len(pending) > 0:
		tflog.Warn(ctx, "the k6 App has not finished setting up", map[string]any{
			"stack_id": d.Get("stack_id"),
			"pending":  strings.Join(pending, "; "),
		})
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "the k6 App has not finished setting up on this stack",
				Detail: "The k6 App is installed, but it reports the following still missing:\n\n  - " +
					strings.Join(pending, "\n  - ") +
					"\n\nOpening the k6 App on this stack normally completes the setup.",
			},
		}
	}
	return nil
}

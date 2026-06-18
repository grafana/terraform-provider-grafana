package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/authlib/claims"
	authlib "github.com/grafana/authlib/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const bootdataRequestTimeout = 10 * time.Second

// GrafanaGet makes an authenticated HTTP GET request to the given Grafana subpath
// using the provider's configured HTTP client and base URL.
func GrafanaGet(ctx context.Context, client *common.Client, subpath string) ([]byte, error) {
	if client == nil || client.GrafanaAPIURLParsed == nil {
		return nil, fmt.Errorf("grafana HTTP client configuration is not available")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.GrafanaSubpath(subpath), nil)
	if err != nil {
		return nil, err
	}

	httpClient := client.GrafanaHTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request to %s failed with status %d: %s", subpath, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

// discoverGrafanaStackID calls /bootdata and extracts the cloud stack ID from the
// namespace field. Returns a non-zero stack ID when the Grafana instance is a
// cloud stack, or an error when the endpoint is unreachable or the instance is
// not a cloud stack.
func discoverGrafanaStackID(ctx context.Context, client *common.Client) (int64, error) {
	body, err := GrafanaGet(ctx, client, "/bootdata")
	if err != nil {
		return 0, err
	}

	var payload struct {
		Settings struct {
			Namespace string `json:"namespace"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("failed to decode /bootdata response: %w", err)
	}

	namespace := strings.TrimSpace(payload.Settings.Namespace)
	if namespace == "" {
		return 0, fmt.Errorf("bootdata returned an empty namespace")
	}

	parsed, err := authlib.ParseNamespace(namespace)
	if err != nil {
		return 0, fmt.Errorf("failed to parse namespace %q: %w", namespace, err)
	}

	if parsed.StackID == 0 {
		return 0, fmt.Errorf("bootdata namespace is not a Grafana Cloud stack namespace %q", namespace)
	}

	return parsed.StackID, nil
}

// ResolveNamespace resolves the Kubernetes namespace for App Platform resources using
// the following precedence:
//
//  1. Always try /bootdata autodiscovery first — handles cloud instances correctly
//     even when org_id is also configured (common for legacy API compatibility).
//  2. Explicit stack_id from provider config.
//  3. Fall back to org_id for local/OSS instances.
//
// An error is returned when none of the above yields a valid namespace.
func ResolveNamespace(ctx context.Context, client *common.Client) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if client == nil {
		diags.AddError("Failed to resolve namespace", "provider client is not configured")
		return "", diags
	}

	discoveryCtx, cancel := context.WithTimeout(ctx, bootdataRequestTimeout)
	defer cancel()

	stackID, discoveryErr := discoverGrafanaStackID(discoveryCtx, client)
	if discoveryErr == nil && stackID > 0 {
		if client.GrafanaStackID > 0 && client.GrafanaStackID != stackID {
			diags.AddError(
				"Stack ID mismatch",
				fmt.Sprintf(
					"The provider `stack_id` is %d but the Grafana instance reports stack %d via `/bootdata`. "+
						"Remove the provider `stack_id` to use autodiscovery, or correct it to match the instance.",
					client.GrafanaStackID,
					stackID,
				),
			)
			return "", diags
		}
		return claims.CloudNamespaceFormatter(stackID), diags
	}

	// 2. Explicit stack_id from provider config.
	if client.GrafanaStackID > 0 {
		return claims.CloudNamespaceFormatter(client.GrafanaStackID), diags
	}

	// 3. Fall back to org_id for local/OSS instances.
	if client.GrafanaOrgID > 0 {
		return claims.OrgNamespaceFormatter(client.GrafanaOrgID), diags
	}

	detail := "Set either provider-level `org_id` or `stack_id` explicitly."
	if discoveryErr != nil {
		detail = fmt.Sprintf(
			"Failed to autodiscover the Grafana Cloud stack namespace from `/bootdata`: %s. %s",
			discoveryErr.Error(),
			detail,
		)
	}
	diags.AddError("Failed to resolve namespace", detail)
	return "", diags
}

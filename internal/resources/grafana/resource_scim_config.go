package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// SCIMConfig represents the SCIM configuration structure
type SCIMConfig struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   SCIMConfigMetadata `json:"metadata"`
	Spec       SCIMConfigSpec     `json:"spec"`
}

// SCIMConfigMetadata represents the metadata for SCIM config
type SCIMConfigMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// SCIMConfigSpec represents the SCIM configuration specification
type SCIMConfigSpec struct {
	EnableUserSync            bool `json:"enableUserSync"`
	EnableGroupSync           bool `json:"enableGroupSync"`
	RejectNonProvisionedUsers bool `json:"rejectNonProvisionedUsers"`
}

func resourceSCIMConfig() *common.Resource {
	schema := &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise.

* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-scim-provisioning/)
`,
		CreateContext: CreateOrUpdateSCIMConfig,
		UpdateContext: CreateOrUpdateSCIMConfig,
		ReadContext:   ReadSCIMConfig,
		DeleteContext: DeleteSCIMConfig,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"enable_user_sync": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether user synchronization is enabled.",
			},
			"enable_group_sync": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether group synchronization is enabled.",
			},
			"reject_non_provisioned_users": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether to block non-provisioned user access to Grafana. Cloud Portal users will always be able to access Grafana, regardless of this setting.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
		"grafana_scim_config",
		common.NewResourceID(common.OptionalIntIDField("orgID")),
		schema,
	)
}

func CreateOrUpdateSCIMConfig(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	_, orgID := OAPIClientFromNewOrgResource(meta, d)

	// Get the transport configuration to access HTTP client and API key
	metaClient := meta.(*common.Client)
	transportConfig := metaClient.GrafanaAPIConfig
	if transportConfig == nil {
		return diag.Errorf("transport configuration not available")
	}

	// Determine namespace based on whether this is on-prem or cloud
	var namespace string
	switch {
	case metaClient.GrafanaStackID > 0:
		// Grafana Cloud instance - use "stacks-{stackId}" namespace
		namespace = fmt.Sprintf("stacks-%d", metaClient.GrafanaStackID)
	case metaClient.GrafanaOrgID > 0:
		// On-prem Grafana instance - use "default" namespace
		namespace = "default"
	default:
		return diag.Errorf("expected either Grafana org ID (for local Grafana) or Grafana stack ID (for Grafana Cloud) to be set")
	}

	// Create or update SCIM config
	scimConfig := SCIMConfig{
		APIVersion: "scim.grafana.app/v0alpha1",
		Kind:       "SCIMConfig",
		Metadata: SCIMConfigMetadata{
			Name:      "default",
			Namespace: namespace,
		},
		Spec: SCIMConfigSpec{
			EnableUserSync:            d.Get("enable_user_sync").(bool),
			EnableGroupSync:           d.Get("enable_group_sync").(bool),
			RejectNonProvisionedUsers: d.Get("reject_non_provisioned_users").(bool),
		},
	}

	jsonData, err := json.Marshal(scimConfig)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to marshal SCIM config: %w", err))
	}

	baseURL := fmt.Sprintf("%s://%s", transportConfig.Schemes[0], transportConfig.Host)

	apiPath, err := url.JoinPath("apis/scim.grafana.app/v0alpha1/namespaces", namespace, "config/default")
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to construct API path: %w", err))
	}
	requestURL := fmt.Sprintf("%s/%s", baseURL, apiPath)

	req, err := http.NewRequestWithContext(ctx, "PUT", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")

	if transportConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+transportConfig.APIKey)
	} else if transportConfig.BasicAuth != nil {
		username := transportConfig.BasicAuth.Username()
		password, _ := transportConfig.BasicAuth.Password()
		req.SetBasicAuth(username, password)
	}

	// Use the HTTP client from the transport configuration
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create or update SCIM config: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return diag.FromErr(fmt.Errorf("failed to create or update SCIM config, status: %d", resp.StatusCode))
	}

	// Set ID if this is a create operation (ID is empty)
	if d.Id() == "" {
		d.SetId(MakeOrgResourceID(orgID, "scim-config"))
	}

	return ReadSCIMConfig(ctx, d, meta)
}

func ReadSCIMConfig(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// Get the transport configuration to access HTTP client and API key
	metaClient := meta.(*common.Client)
	transportConfig := metaClient.GrafanaAPIConfig
	if transportConfig == nil {
		return diag.Errorf("transport configuration not available")
	}

	// Determine namespace based on whether this is on-prem or cloud
	var namespace string
	switch {
	case metaClient.GrafanaStackID > 0:
		// Grafana Cloud instance - use "stacks-{stackId}" namespace
		namespace = fmt.Sprintf("stacks-%d", metaClient.GrafanaStackID)
	case metaClient.GrafanaOrgID > 0:
		// On-prem Grafana instance - use "default" namespace
		namespace = "default"
	default:
		return diag.Errorf("expected either Grafana org ID (for local Grafana) or Grafana stack ID (for Grafana Cloud) to be set")
	}

	// Read SCIM config
	baseURL := fmt.Sprintf("%s://%s", transportConfig.Schemes[0], transportConfig.Host)

	apiPath, err := url.JoinPath("apis/scim.grafana.app/v0alpha1/namespaces", namespace, "config/default")
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to construct API path: %w", err))
	}
	requestURL := fmt.Sprintf("%s/%s", baseURL, apiPath)

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create request: %w", err))
	}

	if transportConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+transportConfig.APIKey)
	} else if transportConfig.BasicAuth != nil {
		username := transportConfig.BasicAuth.Username()
		password, _ := transportConfig.BasicAuth.Password()
		req.SetBasicAuth(username, password)
	}

	// Use the HTTP client from the transport configuration
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read SCIM config: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return diag.FromErr(fmt.Errorf("failed to read SCIM config, status: %d", resp.StatusCode))
	}

	var scimConfig SCIMConfig
	if err := json.NewDecoder(resp.Body).Decode(&scimConfig); err != nil {
		return diag.FromErr(fmt.Errorf("failed to decode SCIM config: %w", err))
	}

	err = d.Set("enable_user_sync", scimConfig.Spec.EnableUserSync)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("enable_group_sync", scimConfig.Spec.EnableGroupSync)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("reject_non_provisioned_users", scimConfig.Spec.RejectNonProvisionedUsers)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func DeleteSCIMConfig(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// Get the transport configuration to access HTTP client and API key
	metaClient := meta.(*common.Client)
	transportConfig := metaClient.GrafanaAPIConfig
	if transportConfig == nil {
		return diag.Errorf("transport configuration not available")
	}

	// Determine namespace based on whether this is on-prem or cloud
	var namespace string
	switch {
	case metaClient.GrafanaStackID > 0:
		// Grafana Cloud instance - use "stacks-{stackId}" namespace
		namespace = fmt.Sprintf("stacks-%d", metaClient.GrafanaStackID)
	case metaClient.GrafanaOrgID > 0:
		// On-prem Grafana instance - use "default" namespace
		namespace = "default"
	default:
		return diag.Errorf("expected either Grafana org ID (for local Grafana) or Grafana stack ID (for Grafana Cloud) to be set")
	}

	// Delete SCIM config
	baseURL := fmt.Sprintf("%s://%s", transportConfig.Schemes[0], transportConfig.Host)

	apiPath, err := url.JoinPath("apis/scim.grafana.app/v0alpha1/namespaces", namespace, "config/default")
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to construct API path: %w", err))
	}
	requestURL := fmt.Sprintf("%s/%s", baseURL, apiPath)

	req, err := http.NewRequestWithContext(ctx, "DELETE", requestURL, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create request: %w", err))
	}

	if transportConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+transportConfig.APIKey)
	} else if transportConfig.BasicAuth != nil {
		username := transportConfig.BasicAuth.Username()
		password, _ := transportConfig.BasicAuth.Password()
		req.SetBasicAuth(username, password)
	}

	// Use the HTTP client from the transport configuration
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete SCIM config: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return diag.FromErr(fmt.Errorf("failed to delete SCIM config, status: %d", resp.StatusCode))
	}

	return nil
}

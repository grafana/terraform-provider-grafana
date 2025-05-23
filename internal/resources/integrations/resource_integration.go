package integrations

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceIntegration() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Cloud integrations.

* [Official documentation](https://grafana.com/docs/grafana-cloud/data-configuration/integrations/)

Required access policy scopes:

* integrations:read
* integrations:write
`,

		CreateContext: createIntegration,
		ReadContext:   readIntegration,
		UpdateContext: updateIntegration,
		DeleteContext: deleteIntegration,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"slug": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The slug of the integration to install (e.g., 'docker', 'linux-node').",
			},
			"installed": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the integration is currently installed.",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the installed integration.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The display name of the integration.",
			},
			"dashboard_folder": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The dashboard folder associated with this integration.",
			},
			"configuration": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Configuration options for the integration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"configurable_logs": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Logs configuration for the integration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"logs_disabled": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "Whether to disable logs collection for this integration.",
									},
								},
							},
						},
						"configurable_alerts": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Alerts configuration for the integration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"alerts_disabled": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "Whether to disable alerts for this integration.",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_integration",
		common.NewResourceID(common.StringIDField("slug")),
		schema,
	)
}

func createIntegration(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getIntegrationsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	slug := d.Get("slug").(string)

	// Check if integration already exists and is installed
	installed, err := client.IsIntegrationInstalled(ctx, slug)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to check integration status: %w", err))
	}

	if installed {
		// Integration is already installed, just set the ID and read the state
		d.SetId(slug)
		return readIntegration(ctx, d, meta)
	}

	// Parse configuration
	config := parseInstallationConfig(d)

	// Install the integration
	err = client.InstallIntegration(ctx, slug, config)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to install integration: %w", err))
	}

	d.SetId(slug)
	return readIntegration(ctx, d, meta)
}

func readIntegration(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getIntegrationsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	slug := d.Id()

	integration, err := client.GetIntegration(ctx, slug)
	if err != nil {
		if err == ErrNotFound {
			return common.WarnMissing("integration", d)
		}
		return diag.FromErr(fmt.Errorf("failed to get integration: %w", err))
	}

	// Set computed attributes
	d.Set("slug", integration.Data.Slug)
	d.Set("name", integration.Data.Name)
	d.Set("version", integration.Data.Version)
	d.Set("dashboard_folder", integration.Data.DashboardFolder)
	d.Set("installed", integration.Data.Installation != nil)

	// Set configuration if available
	if integration.Data.Installation != nil && integration.Data.Installation.Configuration != nil {
		config := flattenInstallationConfig(integration.Data.Installation.Configuration)
		d.Set("configuration", config)
	}

	return nil
}

func updateIntegration(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getIntegrationsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	slug := d.Id()

	// For now, we handle updates by uninstalling and reinstalling
	// This is because the API doesn't seem to have an update endpoint
	if d.HasChange("configuration") {
		// Uninstall first
		err = client.UninstallIntegration(ctx, slug)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to uninstall integration for update: %w", err))
		}

		// Parse new configuration
		config := parseInstallationConfig(d)

		// Reinstall with new configuration
		err = client.InstallIntegration(ctx, slug, config)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to reinstall integration with new configuration: %w", err))
		}
	}

	return readIntegration(ctx, d, meta)
}

func deleteIntegration(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getIntegrationsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	slug := d.Id()

	err = client.UninstallIntegration(ctx, slug)
	if err != nil {
		if err == ErrNotFound {
			// Integration is already uninstalled
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to uninstall integration: %w", err))
	}

	return nil
}

func getIntegrationsClient(meta interface{}) (*Client, error) {
	client := meta.(*common.Client)

	// Get the auth token from the Grafana API config
	authToken := ""
	if client.GrafanaAPIConfig != nil {
		authToken = client.GrafanaAPIConfig.APIKey
	}

	// Create integrations client using the same pattern as frontendo11y
	integrationsClient, err := NewClient(
		client.GrafanaAPIURL,
		authToken,
		nil, // Use default HTTP client
		"terraform-provider-grafana",
		nil, // No default headers for now
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create integrations client: %w", err)
	}

	// Set the Grafana folders client for folder operations
	if client.GrafanaAPI != nil {
		integrationsClient.SetFoldersClient(client.GrafanaAPI.Folders)
	}

	return integrationsClient, nil
}

func parseInstallationConfig(d *schema.ResourceData) *InstallationConfig {
	configList := d.Get("configuration").([]interface{})
	if len(configList) == 0 {
		return nil
	}

	configMap := configList[0].(map[string]interface{})
	config := &InstallationConfig{}

	// Parse configurable_logs
	if logsConfigList, ok := configMap["configurable_logs"].([]interface{}); ok && len(logsConfigList) > 0 {
		logsConfigMap := logsConfigList[0].(map[string]interface{})
		config.ConfigurableLogs = &ConfigurableLogs{
			LogsDisabled: logsConfigMap["logs_disabled"].(bool),
		}
	}

	// Parse configurable_alerts
	if alertsConfigList, ok := configMap["configurable_alerts"].([]interface{}); ok && len(alertsConfigList) > 0 {
		alertsConfigMap := alertsConfigList[0].(map[string]interface{})
		config.ConfigurableAlerts = &ConfigurableAlerts{
			AlertsDisabled: alertsConfigMap["alerts_disabled"].(bool),
		}
	}

	return config
}

func flattenInstallationConfig(config *InstallationConfig) []interface{} {
	if config == nil {
		return nil
	}

	result := make(map[string]interface{})

	if config.ConfigurableLogs != nil {
		result["configurable_logs"] = []interface{}{
			map[string]interface{}{
				"logs_disabled": config.ConfigurableLogs.LogsDisabled,
			},
		}
	}

	if config.ConfigurableAlerts != nil {
		result["configurable_alerts"] = []interface{}{
			map[string]interface{}{
				"alerts_disabled": config.ConfigurableAlerts.AlertsDisabled,
			},
		}
	}

	return []interface{}{result}
}

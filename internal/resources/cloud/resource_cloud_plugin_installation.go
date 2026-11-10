package cloud

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	resourcePluginInstallationID = common.NewResourceID(
		common.StringIDField("stackSlug"),
		common.StringIDField("pluginSlug"),
	)
)

const LatestVersion = "latest"

func resourcePluginInstallation() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Cloud Plugin Installations.

* [Plugin management](https://grafana.com/docs/grafana/latest/administration/plugin-management/)

On stacks where Grafana Cloud manages plugin versions, Grafana Cloud keeps each managed plugin at
the version it chooses. Terraform then keeps ` + "`version`" + ` as configured, reports the real
version in ` + "`installed_version`" + `, and shows a warning when the configured version is not
applied.

Required access policy scopes:

* stack-plugins:read
* stack-plugins:write
* stack-plugins:delete
`,
		Schema: map[string]*schema.Schema{
			"stack_slug": {
				Description: "The stack id to which the plugin should be installed.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"slug": {
				Description: "Slug of the plugin to be installed.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"version": {
				Description: "Version of the plugin to be installed. Defaults to 'latest' and installs the most recent version. Terraform will detect new version as drift for plan/apply. A change updates the plugin in place. On stacks where Grafana Cloud manages plugin versions, Grafana Cloud can keep the plugin at another version, shown in `installed_version`.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     LatestVersion,
			},
			"installed_version": {
				Description: "The version of the plugin installed on the stack. On stacks where Grafana Cloud manages plugin versions, this can differ from `version`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
		CreateContext: withClient[schema.CreateContextFunc](resourcePluginInstallationCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourcePluginInstallationRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourcePluginInstallationUpdate),
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta any) error {
			if diff.Id() != "" && diff.HasChange("version") {
				return diff.SetNewComputed("installed_version")
			}
			return nil
		},
		DeleteContext: withClient[schema.DeleteContextFunc](resourcePluginInstallationDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_plugin_installation",
		resourcePluginInstallationID,
		schema,
	).WithLister(cloudListerFunction(listStackPlugins))
}

func listStackPlugins(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error) {
	stacks, err := data.Stacks(ctx, client)
	if err != nil {
		return nil, err
	}

	var pluginIDs []string
	for _, stack := range stacks {
		var plugins *gcom.GetInstancePlugins200Response
		if err := common.RetryRequest(ctx, "list instance plugins", func() (*http.Response, error) {
			r, httpResp, err := client.InstancesAPI.GetInstancePlugins(ctx, stack.Slug).Execute()
			plugins = r
			return httpResp, err
		}); err != nil {
			return nil, err
		}
		for _, plugin := range plugins.Items {
			pluginIDs = append(pluginIDs, resourcePluginInstallationID.Make(stack.Slug, plugin.PluginSlug))
		}
	}

	return pluginIDs, nil
}

func resourcePluginInstallationCreate(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)
	version := d.Get("version").(string)

	req := gcom.PostInstancePluginsRequest{
		Plugin:  pluginSlug,
		Version: common.Ref(version),
	}

	var created *gcom.FormattedApiInstancePlugin
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, _, err := client.InstancesAPI.PostInstancePlugins(ctx, stackSlug).
			PostInstancePluginsRequest(req).
			XRequestId(ClientRequestID()).Execute()
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "conflict") {
			// If the API returns a conflict error (409), it means that the plugin installation
			// is in progress or there's a temporary conflict. Retry after a delay.
			time.Sleep(10 * time.Second) // Do not retry too fast, default is 500ms
			return retry.RetryableError(err)
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		created = resp
		return nil
	})
	if err != nil {
		return apiError(err)
	}

	d.SetId(resourcePluginInstallationID.Make(stackSlug, pluginSlug))
	if created != nil {
		d.Set("installed_version", created.Version)
	}

	return nil
}

func resourcePluginInstallationRead(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	var installation *gcom.FormattedApiInstancePlugin
	installErr := common.RetryRequest(ctx, "read instance plugin", func() (*http.Response, error) {
		r, httpResp, err := client.InstancesAPI.GetInstancePlugin(ctx, stackSlug.(string), pluginSlug.(string)).Execute()
		installation = r
		return httpResp, err
	})
	if err, shouldReturn := common.CheckReadError("plugin", d, installErr); shouldReturn {
		return err
	}
	desiredVersion := d.Get("version").(string)
	catalogVersion := ""
	if desiredVersion == LatestVersion {
		var catalogPlugin *gcom.FormattedApiPlugin
		catalogErr := common.RetryRequest(ctx, "read plugin", func() (*http.Response, error) {
			r, httpResp, err := client.PluginsAPI.GetPlugin(ctx, pluginSlug.(string)).Execute()
			catalogPlugin = r
			return httpResp, err
		})
		if err, shouldReturn := common.CheckReadError("plugin", d, catalogErr); shouldReturn {
			return err
		}
		catalogVersion = catalogPlugin.Version
	}

	d.Set("stack_slug", installation.InstanceSlug)
	d.Set("slug", installation.PluginSlug)

	decision := decideStateVersion(
		stackSlug.(string),
		pluginSlug.(string),
		desiredVersion,
		installation.Version,
		catalogVersion,
		managedStatusFromAPI(installation),
	)
	d.Set("version", decision.StateVersion)
	d.Set("installed_version", installation.Version)
	d.SetId(resourcePluginInstallationID.Make(stackSlug, pluginSlug))

	if decision.Warning != nil {
		return diag.Diagnostics{*decision.Warning}
	}
	return nil
}

func resourcePluginInstallationUpdate(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	req := gcom.PostInstancePluginRequest{Version: common.Ref(d.Get("version").(string))}

	var updated *gcom.FormattedApiInstancePlugin
	err = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, _, err := client.InstancesAPI.PostInstancePlugin(ctx, stackSlug.(string), pluginSlug.(string)).
			PostInstancePluginRequest(req).
			XRequestId(ClientRequestID()).Execute()
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "conflict") {
			time.Sleep(10 * time.Second) // Do not retry too fast, default is 500ms
			return retry.RetryableError(err)
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		updated = resp
		return nil
	})
	if err != nil {
		// SDKv2 saves the proposed config into state on an update error unless Partial is set.
		d.Partial(true)
		return apiError(err)
	}

	if updated != nil {
		d.Set("installed_version", updated.Version)
	}
	return nil
}

func resourcePluginInstallationDelete(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	err = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		_, _, err := client.InstancesAPI.DeleteInstancePlugin(ctx, stackSlug.(string), pluginSlug.(string)).XRequestId(ClientRequestID()).Execute()
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "conflict") {
			// If the API returns a conflict error (409), it means that the plugin deletion
			// is in progress or there's a temporary conflict. Retry after a delay.
			time.Sleep(10 * time.Second) // Do not retry too fast, default is 500ms
			return retry.RetryableError(err)
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return apiError(err)
	}

	return nil
}

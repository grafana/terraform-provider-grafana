package cloud

import (
	"context"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
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

func resourcePluginInstallation() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Cloud Plugin Installations.

* [Plugin Catalog](https://grafana.com/grafana/plugins/)

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
				Description: "Version of the plugin to be installed.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
		CreateContext: withClient[schema.CreateContextFunc](resourcePluginInstallationCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourcePluginInstallationRead),
		UpdateContext: nil,
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
		plugins, _, err := client.InstancesAPI.GetInstancePlugins(ctx, stack.Slug).Execute()
		if err != nil {
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

	req := gcom.PostInstancePluginsRequest{
		Plugin:  pluginSlug,
		Version: common.Ref(d.Get("version").(string)),
	}

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		_, _, err := client.InstancesAPI.PostInstancePlugins(ctx, stackSlug).
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
		return nil
	})
	if err != nil {
		return apiError(err)
	}

	d.SetId(resourcePluginInstallationID.Make(stackSlug, pluginSlug))

	return nil
}

func resourcePluginInstallationRead(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	installation, _, err := client.InstancesAPI.GetInstancePlugin(ctx, stackSlug.(string), pluginSlug.(string)).Execute()
	if err, shouldReturn := common.CheckReadError("plugin", d, err); shouldReturn {
		return err
	}

	d.Set("stack_slug", installation.InstanceSlug)
	d.Set("slug", installation.PluginSlug)
	d.Set("version", installation.Version)
	d.SetId(resourcePluginInstallationID.Make(stackSlug, pluginSlug))

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

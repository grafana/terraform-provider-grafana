package cloud

import (
	"context"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var ResourcePluginInstallationID = common.NewTFIDWithLegacySeparator("grafana_cloud_plugin_installation", "_", "stackSlug", "pluginSlug") //nolint:staticcheck

func ResourcePluginInstallation() *schema.Resource {
	return &schema.Resource{
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
		CreateContext: ResourcePluginInstallationCreate,
		ReadContext:   ResourcePluginInstallationRead,
		UpdateContext: nil,
		DeleteContext: ResourcePluginInstallationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func ResourcePluginInstallationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)

	req := gcom.PostInstancePluginsRequest{
		Plugin:  pluginSlug,
		Version: common.Ref(d.Get("version").(string)),
	}
	_, _, err := client.InstancesAPI.PostInstancePlugins(ctx, stackSlug).
		PostInstancePluginsRequest(req).
		XRequestId(ClientRequestID()).Execute()
	if err != nil {
		return apiError(err)
	}

	d.SetId(ResourcePluginInstallationID.Make(stackSlug, pluginSlug))

	return nil
}

func ResourcePluginInstallationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	split, err := ResourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	installation, _, err := client.InstancesAPI.GetInstancePlugin(ctx, stackSlug, pluginSlug).Execute()
	if err, shouldReturn := common.CheckReadError("plugin", d, err); shouldReturn {
		return err
	}

	d.Set("stack_slug", installation.InstanceSlug)
	d.Set("slug", installation.PluginSlug)
	d.Set("version", installation.Version)
	d.SetId(ResourcePluginInstallationID.Make(stackSlug, pluginSlug))

	return nil
}

func ResourcePluginInstallationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	split, err := ResourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	_, _, err = client.InstancesAPI.DeleteInstancePlugin(ctx, stackSlug, pluginSlug).XRequestId(ClientRequestID()).Execute()
	return apiError(err)
}

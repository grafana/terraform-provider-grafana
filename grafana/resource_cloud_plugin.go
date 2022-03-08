package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceCloudPluginInstallation() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana Cloud Plugin Installations.

* [Plugin Catalog](https://grafana.com/grafana/plugins/)
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
		CreateContext: resourceCloudPluginInstallationCreate,
		ReadContext:   resourceCloudPluginInstallationRead,
		UpdateContext: nil,
		DeleteContext: resourceCloudPluginInstallationDelete,
		// TODO: Need an ID
		//Importer: &schema.ResourceImporter{
		//	StateContext: schema.ImportStatePassthroughContext,
		//},
	}
}

func resourceCloudPluginInstallationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)
	pluginVersion := d.Get("version").(string)

	err := client.InstallCloudPlugin(stackSlug, pluginSlug, pluginVersion)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCloudPluginInstallationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)

	ok, err := client.IsCloudPluginInstalled(stackSlug, pluginSlug)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("installed", ok)
	_ = d.Set("slug", pluginSlug)

	return nil
}

func resourceCloudPluginInstallationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)

	err := client.UninstallCloudPlugin(stackSlug, pluginSlug)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

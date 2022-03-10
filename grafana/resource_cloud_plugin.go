package grafana

import (
	"context"
	"strconv"

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
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceCloudPluginInstallationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)
	pluginVersion := d.Get("version").(string)

	installation, err := client.InstallCloudPlugin(stackSlug, pluginSlug, pluginVersion)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(installation.ID))

	return nil
}

func resourceCloudPluginInstallationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)

	installation, err := client.GetCloudPluginInstallation(stackSlug, pluginSlug)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("slug", installation.PluginSlug)
	_ = d.Set("version", installation.Version)

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

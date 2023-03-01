package cloud

import (
	"context"
	"log"
	"strings"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourcePluginInstallation() *schema.Resource {
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
	pluginVersion := d.Get("version").(string)

	_, err := client.InstallCloudPlugin(stackSlug, pluginSlug, pluginVersion)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(stackSlug + "_" + pluginSlug)

	return nil
}

func ResourcePluginInstallationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	splitID := strings.SplitN(d.Id(), "_", 2)
	stackSlug, pluginSlug := splitID[0], splitID[1]

	installation, err := client.GetCloudPluginInstallation(stackSlug, pluginSlug)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing plugin %s from state because it no longer exists in stack %s", pluginSlug, stackSlug)
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	d.Set("stack_slug", installation.InstanceSlug)
	d.Set("slug", installation.PluginSlug)
	d.Set("version", installation.Version)

	return nil
}

func ResourcePluginInstallationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	splitID := strings.SplitN(d.Id(), "_", 2)
	stackSlug, pluginSlug := splitID[0], splitID[1]

	err := client.UninstallCloudPlugin(stackSlug, pluginSlug)

	return diag.FromErr(err)
}

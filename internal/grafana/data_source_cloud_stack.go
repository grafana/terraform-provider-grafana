package grafana

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceCloudStack() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for Grafana Stack",
		ReadContext: datasourceCloudStackRead,
		Schema: common.CloneResourceSchemaForDatasource(ResourceCloudStack(), map[string]*schema.Schema{
			"slug": {
				Type:     schema.TypeString,
				Required: true,
				Description: `
Subdomain that the Grafana instance will be available at (i.e. setting slug to “<stack_slug>” will make the instance
available at “https://<stack_slug>.grafana.net".`,
			},
			"region_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The region this stack is deployed to.",
			},
			"wait_for_readiness":         nil,
			"wait_for_readiness_timeout": nil,
		}),
	}
}

func datasourceCloudStackRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	slug := d.Get("slug").(string)

	stack, err := client.StackBySlug(slug)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := FlattenStack(d, stack); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

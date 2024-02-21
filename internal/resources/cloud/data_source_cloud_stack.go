package cloud

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceStack() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for Grafana Stack",
		ReadContext: DataSourceStackRead,
		Schema: common.CloneResourceSchemaForDatasource(ResourceStack(), map[string]*schema.Schema{
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

func DataSourceStackRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId(d.Get("slug").(string))
	return ReadStack(ctx, d, meta)
}

package cloud

import (
	"context"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceStack() *common.DataSource {
	schema := &schema.Resource{
		Description: "Data source for Grafana Stack",
		ReadContext: withClient[schema.ReadContextFunc](datasourceStackRead),
		Schema: common.CloneResourceSchemaForDatasource(resourceStack().Schema, map[string]*schema.Schema{
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
	return common.NewLegacySDKDataSource(common.CategoryCloud, "grafana_cloud_stack", schema)
}

func datasourceStackRead(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	d.SetId(d.Get("slug").(string))
	return readStack(ctx, d, client)
}

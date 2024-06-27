package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceAWSCloudWatchScrapeJobs() *common.DataSource {
	schema := &schema.Resource{
		ReadContext: datasourceAWSCloudWatchScrapeJobRead,
		Schema: map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID whose scrape jobs are to be listed.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"jobs": {
				Description: "The CloudWatch Scrape Jobs associated with the StackID.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: common.CloneResourceSchemaForDatasource(resourceAWSCloudWatchScrapeJob().Schema, nil),
				},
			},
		},
	}

	return common.NewLegacySDKDataSource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_cloudwatch_scrape_jobs",
		schema,
	)
}

func datasourceAWSCloudWatchScrapeJobsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	return diags
}

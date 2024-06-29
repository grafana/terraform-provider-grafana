package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func makeDatasourceAWSCloudWatchScrapeJob() *common.DataSource {
	schema := &schema.Resource{
		ReadContext: datasourceAWSCloudWatchScrapeJobRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceAWSCloudWatchScrapeJob().Schema, map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID of the AWS CloudWatch Scrape Job resource to look up.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "The name of the CloudWatch Scrape Job resource to look up.",
				Type:        schema.TypeString,
				Required:    true,
			},
		}),
	}

	return common.NewLegacySDKDataSource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_cloudwatch_scrape_job",
		schema,
	)
}

func datasourceAWSCloudWatchScrapeJobRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.SetId(resourceAWSCWScrapeJobTerraformID.Make(d.Get("stack_id").(string), d.Get("name").(string)))
	d.Set("aws_account_resource_id", TestAWSCloudWatchScrapeJobData.AWSAccountResourceID)
	d.Set("enabled", TestAWSCloudWatchScrapeJobData.Enabled)
	d.Set("regions", TestAWSCloudWatchScrapeJobData.Regions)
	d.Set("service_configurations", TestAWSCloudWatchScrapeJobData.ServiceConfigurations)

	return diags
}

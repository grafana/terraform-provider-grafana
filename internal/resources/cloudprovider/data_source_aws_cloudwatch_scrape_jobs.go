package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func makeDatasourceAWSCloudWatchScrapeJobs() *common.DataSource {
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

	jobsResp := []cloudproviderapi.AWSCloudWatchScrapeJob{TestAWSCloudWatchScrapeJobData}

	jobs := make([]map[string]interface{}, len(jobsResp))
	for i, result := range jobsResp {
		jobs[i] = map[string]interface{}{
			"name":                    result.Name,
			"enabled":                 result.Enabled,
			"aws_account_resource_id": result.AWSAccountResourceID,
			"regions":                 result.Regions,
			"service_configurations":  result.ServiceConfigurations,
		}
	}

	if err := d.Set("jobs", jobs); err != nil {
		return diag.Errorf("error setting jobs attribute: %s", err)
	}

	return diags
}

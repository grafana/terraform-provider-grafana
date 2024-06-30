package cloudprovider_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudprovider"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceAWSCloudWatchScrapeJobs(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		IsUnitTest:               true,
		// TODO(tristan): actually check for resource existence
		CheckDestroy: func() resource.TestCheckFunc {
			return func(s *terraform.State) error {
				return nil
			}
		}(),
		Steps: []resource.TestStep{
			{
				Config: awsCloudWatchScrapeJobsDataSourceData(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "id", cloudprovider.TestAWSCloudWatchScrapeJobData.StackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "stack_id", cloudprovider.TestAWSCloudWatchScrapeJobData.StackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.stack_id", cloudprovider.TestAWSCloudWatchScrapeJobData.StackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.aws_account_resource_id", cloudprovider.TestAWSCloudWatchScrapeJobData.AWSAccountResourceID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.0", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.1", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.2", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[2]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.metric.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.metric.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.metric.0.statistics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.metric.0.statistics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.scrape_interval_seconds", fmt.Sprintf("%d", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.resource_discovery_tag_filter.0.key", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.resource_discovery_tag_filter.0.value", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.tags_to_add_to_metrics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].TagsToAddToMetrics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.0.is_custom_namespace", fmt.Sprintf("%t", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].IsCustomNamespace)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.metric.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.metric.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.metric.0.statistics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.metric.0.statistics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.scrape_interval_seconds", fmt.Sprintf("%d", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service_configuration.1.is_custom_namespace", fmt.Sprintf("%t", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].IsCustomNamespace)),
				),
			},
		},
	})
}

func awsCloudWatchScrapeJobsDataSourceData() string {
	data := fmt.Sprintf(`
data "grafana_cloud_provider_aws_cloudwatch_scrape_jobs" "test" {
	stack_id = "%[1]s"
}
`,
		cloudprovider.TestAWSCloudWatchScrapeJobData.StackID,
	)

	return data
}

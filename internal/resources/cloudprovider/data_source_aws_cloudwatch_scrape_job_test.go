package cloudprovider_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAWSCloudWatchScrapeJob(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	testCfg := makeTestConfig(t)

	var gotJob cloudproviderapi.AWSCloudWatchScrapeJobResponse

	jobName := "test-job" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					regionsString([]string{testAWSCloudWatchScrapeJobData.Regions[0], testAWSCloudWatchScrapeJobData.Regions[1]}),
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				) + awsCloudWatchScrapeJobDataSourceData(testCfg.stackID, jobName),
				Check: resource.ComposeTestCheckFunc(
					checkAWSCloudWatchScrapeJobResourceExists("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", testCfg.stackID, &gotJob),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "enabled", "false"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "disabled_reason", "disabled_by_user"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "aws_account_resource_id", testCfg.accountID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.#", "2"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.0", testAWSCloudWatchScrapeJobData.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.1", testAWSCloudWatchScrapeJobData.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override_used", "true"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "export_tags", fmt.Sprintf("%t", testAWSCloudWatchScrapeJobData.ExportTags)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.name", testAWSCloudWatchScrapeJobData.Services[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.name", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.Services[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.key", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.value", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.0", testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.0.label", "label1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.0.value", testAWSCloudWatchScrapeJobData.StaticLabels["label1"]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.1.label", "label2"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.1.value", testAWSCloudWatchScrapeJobData.StaticLabels["label2"]),
				),
			},
			// update to remove regions_subset_override so that the account's regions are used instead
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					"",
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				) + awsCloudWatchScrapeJobDataSourceData(testCfg.stackID, jobName),
				Check: resource.ComposeTestCheckFunc(
					checkAWSCloudWatchScrapeJobResourceExists("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", testCfg.stackID, &gotJob),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.0", testAWSCloudWatchScrapeJobData.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.1", testAWSCloudWatchScrapeJobData.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.2", testAWSCloudWatchScrapeJobData.Regions[2]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override_used", "false"),
				),
			},
		},
		CheckDestroy: checkAWSCloudWatchScrapeJobResourceDestroy(testCfg.stackID, &gotJob),
	})
}

func awsCloudWatchScrapeJobDataSourceData(stackID string, jobName string) string {
	data := fmt.Sprintf(`
data "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
	stack_id = "%[1]s"
	name = grafana_cloud_provider_aws_cloudwatch_scrape_job.test.name
}
`,
		stackID,
		jobName,
	)

	return data
}

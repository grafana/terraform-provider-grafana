package cloudprovider_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccDataSourceAWSCloudWatchScrapeJobs(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Uses a pre-existing account resource so that we don't need to create a new one for every test run
	accountID := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_AWS_ACCOUNT_RESOURCE_ID")
	require.NotEmpty(t, accountID, "GRAFANA_CLOUD_PROVIDER_TEST_AWS_ACCOUNT_RESOURCE_ID must be set")

	roleARN := os.Getenv("GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN")
	require.NotEmpty(t, roleARN, "GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN must be set")

	stackID := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID")
	require.NotEmpty(t, stackID, "GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID must be set")

	// Make sure the account exists and matches the role ARN we expect for testing
	client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
	gotAccount, err := client.GetAWSAccount(context.Background(), stackID, accountID)
	require.NoError(t, err)
	require.Equal(t, roleARN, gotAccount.RoleARN)

	var gotJob cloudproviderapi.AWSCloudWatchScrapeJobResponse

	jobName := "test-job" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: awsCloudWatchScrapeJobResourceData(stackID,
					jobName,
					false,
					accountID,
					regionsString(testAWSCloudWatchScrapeJobData.Regions),
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				) + awsCloudWatchScrapeJobsDataSourceData(stackID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "id", stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "stack_id", stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.stack_id", stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.name", jobName),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "scrape_job.0.enabled", "false"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "scrape_job.0.disabled_reason", "disabled_by_user"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.aws_account_resource_id", accountID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.0", testAWSCloudWatchScrapeJobData.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.1", testAWSCloudWatchScrapeJobData.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions.2", testAWSCloudWatchScrapeJobData.Regions[2]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.regions_subset_override_used", "true"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.export_tags", fmt.Sprintf("%t", testAWSCloudWatchScrapeJobData.ExportTags)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.name", testAWSCloudWatchScrapeJobData.Services[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.metric.0.name", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.Services[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.resource_discovery_tag_filter.0.key", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.resource_discovery_tag_filter.0.value", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.service.0.tags_to_add_to_metrics.0", testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.custom_namespace.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.custom_namespace.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.custom_namespace.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.custom_namespace.0.metric.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.custom_namespace.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.custom_namespace.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.0.custom_namespace.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].ScrapeIntervalSeconds)),
				),
			},
		},
		CheckDestroy: checkAWSCloudWatchScrapeJobResourceDestroy(stackID, &gotJob),
	})
}

func awsCloudWatchScrapeJobsDataSourceData(stackID string) string {
	data := fmt.Sprintf(`
data "grafana_cloud_provider_aws_cloudwatch_scrape_jobs" "test" {
	stack_id = "%[1]s"
}
`,
		stackID,
	)

	return data
}

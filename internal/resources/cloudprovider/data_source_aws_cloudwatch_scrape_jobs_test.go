package cloudprovider_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAWSCloudWatchScrapeJobs(t *testing.T) {
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
					regionsString(testAWSCloudWatchScrapeJobData.Regions),
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				) + awsCloudWatchScrapeJobsDataSourceData(testCfg.stackID),
				Check: resource.ComposeTestCheckFunc(
					checkAWSCloudWatchScrapeJobResourceExists("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", testCfg.stackID, &gotJob),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "id", testCfg.stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttrWith("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.#", func(v string) error {
						if got, err := strconv.Atoi(v); err != nil || got == 0 {
							return fmt.Errorf("expected at least one scrape job")
						}
						return nil
					}),
				),
			},
		},
		CheckDestroy: checkAWSCloudWatchScrapeJobResourceDestroy(testCfg.stackID, &gotJob),
	})
}

func awsCloudWatchScrapeJobsDataSourceData(stackID string) string {
	data := fmt.Sprintf(`
data "grafana_cloud_provider_aws_cloudwatch_scrape_jobs" "test" {
	stack_id = "%[1]s"
	depends_on = [grafana_cloud_provider_aws_cloudwatch_scrape_job.test]
}
`,
		stackID,
	)

	return data
}

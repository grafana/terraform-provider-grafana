package cloudprovider_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
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
					checkAWSCloudWatchScrapeJobResourceExists("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", stackID, &gotJob),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "id", stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "stack_id", stackID),
					resource.TestCheckResourceAttrWith("data.grafana_cloud_provider_aws_cloudwatch_scrape_jobs.test", "scrape_job.#", func(v string) error {
						if got, err := strconv.Atoi(v); err != nil || got == 0 {
							return fmt.Errorf("expected at least one scrape job")
						}
						return nil
					}),
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
	depends_on = [grafana_cloud_provider_aws_cloudwatch_scrape_job.test]
}
`,
		stackID,
	)

	return data
}

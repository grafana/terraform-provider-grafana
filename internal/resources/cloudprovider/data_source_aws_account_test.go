package cloudprovider_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourceAWSAccount(t *testing.T) {
	t.Skip("TODO(tristan): switch to CloudInstanceTestsEnabled")
	// testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// TODO(tristan): actually check for resource existence
		CheckDestroy: func() resource.TestCheckFunc {
			return func(s *terraform.State) error {
				return nil
			}
		}(),
		Steps: []resource.TestStep{
			{
				// Creates an AWS Account resource
				Config: awsAccountResourceData(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "stack_id", testAWSAccountData.StackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "role_arn", testAWSAccountData.RoleARN),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(testAWSAccountData.Regions))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.0", testAWSAccountData.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.1", testAWSAccountData.Regions[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.2", testAWSAccountData.Regions[2]),
				),
			},
			{
				// Verifies that the created AWS Account is read by the datasource read function
				Config: awsAccountDatasourceData(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "stack_id", testAWSAccountData.StackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "role_arn", testAWSAccountData.RoleARN),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(testAWSAccountData.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.0", testAWSAccountData.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.1", testAWSAccountData.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.2", testAWSAccountData.Regions[2]),
				),
			},
		},
	})
}

func awsAccountDatasourceData() string {
	return fmt.Sprintf(`
data "grafana_cloud_provider_aws_account" "test" {
	stack_id = "%[1]s"
	role_arn = "%[2]s"
}
`,
		testAWSAccountData.StackID,
		testAWSAccountData.RoleARN,
	)
}

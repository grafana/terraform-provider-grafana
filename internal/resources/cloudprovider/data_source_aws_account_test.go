package cloudprovider_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudprovider"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourceAWSAccount(t *testing.T) {
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
				Config: testAccResourceAWSAccount(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "stack_id", cloudprovider.TestAWSAccountData.StackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "role_arn", cloudprovider.TestAWSAccountData.RoleARN),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(cloudprovider.TestAWSAccountData.Regions))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.0", cloudprovider.TestAWSAccountData.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.1", cloudprovider.TestAWSAccountData.Regions[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.2", cloudprovider.TestAWSAccountData.Regions[2]),
				),
			},
			{
				// Verifies that the created AWS Account is read by the datasource read function
				Config: testAccDatasourceAWSAccount(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "stack_id", cloudprovider.TestAWSAccountData.StackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "role_arn", cloudprovider.TestAWSAccountData.RoleARN),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(cloudprovider.TestAWSAccountData.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.0", cloudprovider.TestAWSAccountData.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.1", cloudprovider.TestAWSAccountData.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.2", cloudprovider.TestAWSAccountData.Regions[2]),
				),
			},
		},
	})
}

func testAccDatasourceAWSAccount() string {
	return fmt.Sprintf(`
data "grafana_cloud_provider_aws_account" "test" {
	stack_id = "%[1]s"
	role_arn = "%[2]s"
}
`,
		cloudprovider.TestAWSAccountData.StackID,
		cloudprovider.TestAWSAccountData.RoleARN,
	)
}

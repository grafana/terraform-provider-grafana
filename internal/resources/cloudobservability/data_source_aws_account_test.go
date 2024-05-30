package cloudobservability_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudobservability"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourceAWSAccount(t *testing.T) {
	randomName := acctest.RandomWithPrefix(cloudobservability.TestAWSAccountData.Name)

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
				Config: testAccResourceAWSAccount(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "stack_id", cloudobservability.TestAWSAccountData.StackID),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.%", strconv.Itoa(len(cloudobservability.TestAWSAccountData.RoleARNs))),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.my role 1a", cloudobservability.TestAWSAccountData.RoleARNs["my role 1a"]),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.my role 1b", cloudobservability.TestAWSAccountData.RoleARNs["my role 1b"]),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.my role 2", cloudobservability.TestAWSAccountData.RoleARNs["my role 2"]),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.#", strconv.Itoa(len(cloudobservability.TestAWSAccountData.Regions))),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.0", cloudobservability.TestAWSAccountData.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.1", cloudobservability.TestAWSAccountData.Regions[1]),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.2", cloudobservability.TestAWSAccountData.Regions[2]),
				),
			},
			{
				// Verifies that the created AWS Account is read by the datasource read function
				Config: testAccDatasourceAWSAccount(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "stack_id", cloudobservability.TestAWSAccountData.StackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "name", randomName),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.%", strconv.Itoa(len(cloudobservability.TestAWSAccountData.RoleARNs))),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.my role 1a", cloudobservability.TestAWSAccountData.RoleARNs["my role 1a"]),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.my role 1b", cloudobservability.TestAWSAccountData.RoleARNs["my role 1b"]),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.my role 2", cloudobservability.TestAWSAccountData.RoleARNs["my role 2"]),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.#", strconv.Itoa(len(cloudobservability.TestAWSAccountData.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.0", cloudobservability.TestAWSAccountData.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.1", cloudobservability.TestAWSAccountData.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.2", cloudobservability.TestAWSAccountData.Regions[2]),
				),
			},
		},
	})
}

func testAccDatasourceAWSAccount(randomName string) string {
	return fmt.Sprintf(`
data "grafana_cloud_observability_aws_account" "test" {
	stack_id = "%[1]s"
	name     = "%[2]s"
}
`,
		cloudobservability.TestAWSAccountData.StackID,
		randomName,
	)
}

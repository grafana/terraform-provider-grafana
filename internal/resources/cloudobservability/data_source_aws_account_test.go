package cloudobservability_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourceAWSAccount(t *testing.T) {
	randomName := acctest.RandomWithPrefix("my-aws-account")

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
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_cloud_observability_aws_account/resource.tf", map[string]string{
					"my-aws-account": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "stack_id", "001"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.%", "3"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.my role 1a", "arn:aws:iam::123456789012:role/my-role-1a"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.my role 1b", "arn:aws:iam::123456789012:role/my-role-1b"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "role_arns.my role 2", "arn:aws:iam::210987654321:role/my-role-2"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.#", "3"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.0", "us-east-1"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.1", "us-east-2"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions.2", "us-west-1"),
				),
			},
			{
				// Verifies that the created SLO Resource is read by the Datasource Read Method
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_cloud_observability_aws_account/data-source.tf", map[string]string{
					"my-aws-account": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "stack_id", "001"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "name", randomName),
					// TODO(tristan): check other attributes from API
					/*resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.%", "3"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.my role 1a", "arn:aws:iam::123456789012:role/my-role-1a"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.my role 1b", "arn:aws:iam::123456789012:role/my-role-1b"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "role_arns.my role 2", "arn:aws:iam::210987654321:role/my-role-2"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.#", "3"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.0", "us-east-1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.1", "us-east-2"),
					resource.TestCheckResourceAttr("data.grafana_cloud_observability_aws_account.test", "regions.2", "us-west-1"),
					*/
				),
			},
		},
	})
}

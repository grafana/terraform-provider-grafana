package cloudobservability_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

/*
	resource "grafana_cloud_observability_aws_account" "my-aws-account" {
	  stack_id = data.grafana_cloud_stack.test.id
	  name     = "my-aws-account"
	  role_arns = {
	    "my role 1a" = "arn:aws:iam::123456789012:role/my-role-1a",
	    "my role 1b" = "arn:aws:iam::123456789012:role/my-role-1b",
	    "my role 2"  = "arn:aws:iam::210987654321:role/my-role-2",
	  }
	  regions = [
	    "us-east-1",
	    "us-east-2",
	    "us-west-1"
	  ]
	}
*/
func TestAccResourceAWSAccount(t *testing.T) {
	randomName := acctest.RandomWithPrefix("my-aws-account")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_cloud_observability_aws_account/resource.tf", map[string]string{
					"my-aws-account": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "stack_id", "001"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "my role 1a", "arn:aws:iam::123456789012:role/my-role-1a"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "my role 1b", "arn:aws:iam::123456789012:role/my-role-1b"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "my role 2", "arn:aws:iam::210987654321:role/my-role-2"),
					resource.TestCheckResourceAttr("grafana_cloud_observability_aws_account.test", "regions", "86.92262268066406"),
				),
			},
		},
	})
}

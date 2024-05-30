package cloudobservability_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudobservability"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceAWSAccount(t *testing.T) {
	randomName := acctest.RandomWithPrefix(cloudobservability.TestAWSAccountData.Name)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
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
		},
	})
}

func testAccResourceAWSAccount(randomName string) string {
	return fmt.Sprintf(`
resource "grafana_cloud_observability_aws_account" "test" {
	stack_id = "%[1]s"
	name     = "%[2]s"
	role_arns = {%[3]s}
	regions = [%[4]s]
}
`,
		cloudobservability.TestAWSAccountData.StackID,
		randomName,
		roleARNsString(cloudobservability.TestAWSAccountData.RoleARNs),
		regionsString(cloudobservability.TestAWSAccountData.Regions),
	)
}

func roleARNsString(roleARNs map[string]string) string {
	b := new(bytes.Buffer)
	for name, arn := range roleARNs {
		fmt.Fprintf(b, "\n\t\t\"%s\" = \"%s\",", name, arn)
	}
	fmt.Fprintf(b, "\n\t")
	return b.String()
}

func regionsString(regions []string) string {
	b := new(bytes.Buffer)
	for _, region := range regions {
		fmt.Fprintf(b, "\n\t\t\"%s\",", region)
	}
	fmt.Fprintf(b, "\n\t")
	return b.String()
}

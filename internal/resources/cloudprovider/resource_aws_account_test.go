package cloudprovider_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudprovider"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceAWSAccount(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
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
		},
	})
}

func testAccResourceAWSAccount() string {
	return fmt.Sprintf(`
resource "grafana_cloud_provider_aws_account" "test" {
	stack_id = "%[1]s"
	role_arn = "%[2]s"
	regions = [%[3]s]
}
`,
		cloudprovider.TestAWSAccountData.StackID,
		cloudprovider.TestAWSAccountData.RoleARN,
		regionsString(cloudprovider.TestAWSAccountData.Regions),
	)
}

func regionsString(regions []string) string {
	b := new(bytes.Buffer)
	for _, region := range regions {
		fmt.Fprintf(b, "\n\t\t\"%s\",", region)
	}
	fmt.Fprintf(b, "\n\t")
	return b.String()
}

package cloudprovider_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAWSAccount(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	testCfg := makeTestConfig(t)

	account := cloudproviderapi.AWSAccount{
		ID:      testCfg.accountID,
		Name:    testCfg.accountName,
		RoleARN: testCfg.roleARN,
		Regions: []string{"us-east-1", "us-east-2", "us-west-1"},
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: awsAccountDataSourceData(testCfg.stackID, account),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "resource_id", account.ID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "name", account.Name),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "role_arn", account.RoleARN),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(account.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.0", account.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.1", account.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.2", account.Regions[2]),
				),
			},
		},
	})
}

func awsAccountDataSourceData(stackID string, account cloudproviderapi.AWSAccount) string {
	return fmt.Sprintf(`
data "grafana_cloud_provider_aws_account" "test" {
	stack_id    = "%[1]s"
	resource_id = "%[2]s"
}
`,
		stackID,
		account.ID,
	)
}

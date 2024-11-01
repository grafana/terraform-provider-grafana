package cloudprovider_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccDataSourceAWSAccount(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	roleARN := os.Getenv("GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN")
	require.NotEmpty(t, roleARN, "GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN must be set")

	stackID := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID")
	require.NotEmpty(t, roleARN, "GRAFANA_CLOUD_PROVIDER_TEST_STACK_IDmust be set")

	account := cloudproviderapi.AWSAccount{
		RoleARN: roleARN,
		Regions: []string{"us-east-1", "us-east-2", "us-west-1"},
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: awsAccountDataSourceData(stackID, account),
				Check: resource.ComposeTestCheckFunc(
					checkAWSAccountResourceExists("grafana_cloud_provider_aws_account.test", stackID, &account),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "stack_id", stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "role_arn", account.RoleARN),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(account.Regions))),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.0", account.Regions[0]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.1", account.Regions[1]),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "regions.2", account.Regions[2]),
				),
			},
		},
		CheckDestroy: checkAWSAccountResourceDestroy(stackID, &account),
	})
}

func awsAccountDataSourceData(stackID string, account cloudproviderapi.AWSAccount) string {
	return fmt.Sprintf(`
resource "grafana_cloud_provider_aws_account" "test" {
	stack_id = "%[1]s"
	role_arn = "%[2]s"
	regions  = [%[3]s]
}

data "grafana_cloud_provider_aws_account" "test" {
	stack_id    = "%[1]s"
	resource_id = grafana_cloud_provider_aws_account.test.resource_id
}
`,
		stackID,
		account.RoleARN,
		regionsString(account.Regions),
	)
}

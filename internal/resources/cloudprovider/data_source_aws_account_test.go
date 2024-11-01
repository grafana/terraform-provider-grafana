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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccDataSourceAWSAccount(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Uses a pre-existing account resource so that we don't need to create a new one for every test run.
	accountID := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_AWS_ACCOUNT_RESOURCE_ID")
	require.NotEmpty(t, accountID, "GRAFANA_CLOUD_PROVIDER_TEST_AWS_ACCOUNT_RESOURCE_ID must be set")

	roleARN := os.Getenv("GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN")
	require.NotEmpty(t, roleARN, "GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN must be set")

	stackID := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID")
	require.NotEmpty(t, roleARN, "GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID must be set")

	// Make sure the account exists and matches the role ARN we expect for testing
	client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
	gotAccount, err := client.GetAWSAccount(context.Background(), stackID, accountID)
	require.NoError(t, err)
	require.Equal(t, roleARN, gotAccount.RoleARN)

	account := cloudproviderapi.AWSAccount{
		ID:      accountID,
		RoleARN: roleARN,
		Regions: []string{"us-east-1", "us-east-2", "us-west-1"},
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: awsAccountDataSourceData(stackID, account),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "stack_id", stackID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_aws_account.test", "resource_id", account.ID),
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

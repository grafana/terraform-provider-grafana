package cloudprovider_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceAWSAccount(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	testCfg := makeTestConfig(t)

	account := cloudproviderapi.AWSAccount{
		RoleARN: testCfg.roleARN,
		Regions: []string{"us-east-1", "us-east-2", "us-west-1"},
	}
	var gotAccount cloudproviderapi.AWSAccount

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: awsAccountResourceData(testCfg.stackID, account),
				Check: resource.ComposeTestCheckFunc(
					checkAWSAccountResourceExists("grafana_cloud_provider_aws_account.test", testCfg.stackID, &gotAccount),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "role_arn", account.RoleARN),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "name", account.Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(account.Regions))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.0", account.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.1", account.Regions[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.2", account.Regions[2]),
				),
			},
			{
				ResourceName:      "grafana_cloud_provider_aws_account.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: checkAWSAccountResourceDestroy(testCfg.stackID, &gotAccount),
	})
}

func checkAWSAccountResourceExists(rn string, stackID string, account *cloudproviderapi.AWSAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		parts := strings.SplitN(rs.Primary.ID, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("Invalid ID: %s", rs.Primary.ID)
		}
		accountID := parts[1]

		if accountID == "" {
			return fmt.Errorf("account id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
		gotAccount, err := client.GetAWSAccount(context.Background(), stackID, accountID)
		if err != nil {
			return fmt.Errorf("error getting account: %s", err)
		}

		*account = gotAccount

		return nil
	}
}

func checkAWSAccountResourceDestroy(stackID string, account *cloudproviderapi.AWSAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if account.ID == "" {
			return fmt.Errorf("checking deletion of empty account id")
		}

		client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
		_, err := client.GetAWSAccount(context.Background(), stackID, account.ID)
		if err == nil {
			return fmt.Errorf("account still exists")
		} else if !common.IsNotFoundError(err) {
			return fmt.Errorf("unexpected error retrieving account: %s", err)
		}

		return nil
	}
}

func awsAccountResourceData(stackID string, account cloudproviderapi.AWSAccount) string {
	return fmt.Sprintf(`
resource "grafana_cloud_provider_aws_account" "test" {
	stack_id = "%[1]s"
	role_arn = "%[2]s"
	regions = [%[3]s]
	name = [%[4]s]
}
`,
		stackID,
		account.RoleARN,
		regionsString(account.Regions),
		account.Name,
	)
}

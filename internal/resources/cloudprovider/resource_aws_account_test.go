package cloudprovider_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAWSAccountData = struct {
	StackID string
	RoleARN string
	Regions []string
}{
	StackID: "001",
	RoleARN: "arn:aws:iam::123456789012:role/my-role-1a",
	Regions: []string{"us-east-1", "us-east-2", "us-west-1"},
}

func TestAccResourceAWSAccount(t *testing.T) {
	// TODO(tristan): switch to CloudInstanceTestsEnabled
	// as part of https://github.com/grafana/grafana-aws-app/issues/381
	t.Skip("not yet implemented. see TODO comment.")
	// testutils.CheckCloudInstanceTestsEnabled(t)

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
				Config: awsAccountResourceData(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "stack_id", testAWSAccountData.StackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "role_arn", testAWSAccountData.RoleARN),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.#", strconv.Itoa(len(testAWSAccountData.Regions))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.0", testAWSAccountData.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.1", testAWSAccountData.Regions[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_account.test", "regions.2", testAWSAccountData.Regions[2]),
				),
			},
		},
	})
}

func awsAccountResourceData() string {
	return fmt.Sprintf(`
resource "grafana_cloud_provider_aws_account" "test" {
	stack_id = "%[1]s"
	role_arn = "%[2]s"
	regions = [%[3]s]
}
`,
		testAWSAccountData.StackID,
		testAWSAccountData.RoleARN,
		regionsString(testAWSAccountData.Regions),
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

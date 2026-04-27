package cloud_test

import (
	"fmt"
	"os"

	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// This test covers both the cloud_access_policy and cloud_access_policy_token resources as well as configuration of a data source with PDC.
func TestResourcePrivateDataSourceConnectNetworkToken_Basic(t *testing.T) {
	t.Parallel()
	testutils.CheckCloudAPITestsEnabled(t)

	var pdcNetwork gcom.AuthAccessPolicy
	var pdcNetworkToken gcom.AuthToken

	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)
	initialName := fmt.Sprintf("pdc-initial-%s", randomName)
	// The HCL config below names the token "token-<name>".
	initialToken := fmt.Sprintf("token-%s", initialName)

	const networkRN = "grafana_cloud_private_data_source_connect_network.test"
	const tokenRN = "grafana_cloud_private_data_source_connect_network_token.test"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCloudAccessPolicyCheckDestroy("prod-us-east-0", &pdcNetwork),
			testAccCloudAccessPolicyTokenCheckDestroy("prod-us-east-0", &pdcNetworkToken),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudPrivateDataSourceConnectNetworkConfigBasic(initialName, "", "prod-us-east-0"),
				Check: resource.ComposeTestCheckFunc(
					// PDC networks/tokens are access policies/tokens under the
					// hood, so the cloud access policy check helpers work for
					// the existence/destroy assertions.
					testAccCloudAccessPolicyCheckExists(networkRN, &pdcNetwork),
					testAccCloudAccessPolicyTokenCheckExists(tokenRN, &pdcNetworkToken),

					resource.TestCheckResourceAttr(networkRN, "name", initialName),
					resource.TestCheckResourceAttr(networkRN, "region", "prod-us-east-0"),
					resource.TestCheckResourceAttrSet(networkRN, "pdc_network_id"),
					resource.TestCheckResourceAttrSet(networkRN, "stack_identifier"),

					resource.TestCheckResourceAttr(tokenRN, "name", initialToken),
					resource.TestCheckResourceAttr(tokenRN, "region", "prod-us-east-0"),
					resource.TestCheckResourceAttrSet(tokenRN, "pdc_network_id"),
					resource.TestCheckResourceAttrSet(tokenRN, "token"),
				),
			},
		},
	})
}

func testAccCloudPrivateDataSourceConnectNetworkConfigBasic(name, displayName, region string) string {
	if displayName != "" {
		displayName = fmt.Sprintf("display_name = \"%s\"", displayName)
	}

	return fmt.Sprintf(`
	data "grafana_cloud_stack" "current" {
		slug = "%[1]s"
	}

	resource "grafana_cloud_private_data_source_connect_network" "test" {
		region       = "%[2]s"
		name         = "%[3]s"
		display_name = "%[4]s"
		stack_identifier = data.grafana_cloud_stack.current.id
	}

	resource "grafana_cloud_private_data_source_connect_network_token" "test" {
		pdc_network_id = grafana_cloud_private_data_source_connect_network.test.pdc_network_id
		region           = "%[2]s"
		name             = "token-%[3]s"
		display_name 	 = "%[4]s" 
	}
	`, os.Getenv("GRAFANA_CLOUD_ORG"), region, name, displayName)
}

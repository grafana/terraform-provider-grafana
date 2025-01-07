package cloud_test

import (
	"fmt"
	"os"

	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
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
	initialToken := fmt.Sprintf("pdc-token-%s", initialName)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCloudAccessPolicyCheckDestroy("us", &pdcNetwork),
			testAccCloudAccessPolicyTokenCheckDestroy("us", &pdcNetworkToken),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudPrivateDataSourceConnectNetworkConfigBasic(initialName, "", "us"),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &pdcNetwork),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &pdcNetworkToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "name", initialName),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "display_name", initialName),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.0", "pdc-signing:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.0.type", "stack"),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "name", initialToken),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "display_name", initialToken),
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
		stack_identifier = grafana_cloud_stack.current.id
	}

	resource "grafana_cloud_private_data_source_connect_network_token" "test" {
		pdc_network_id = grafana_cloud_private_data_source_connect_network.test.pdc_network_id
		region           = "%[2]s"
		name             = "token-%[3]s"
		display_name 	 = "%[4]s" 
	}
	`, os.Getenv("GRAFANA_CLOUD_ORG"), region, name, displayName)
}

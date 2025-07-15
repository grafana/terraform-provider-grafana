package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceSCIMConfig_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.0.0")

	resourceName := "grafana_scim_config.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSCIMConfigResourceConfig(true, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable_user_sync", "true"),
					resource.TestCheckResourceAttr(resourceName, "enable_group_sync", "false"),
					resource.TestCheckResourceAttr(resourceName, "allow_non_provisioned_users", "false"),
				),
			},
			{
				Config: testAccSCIMConfigResourceConfig(false, true, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable_user_sync", "false"),
					resource.TestCheckResourceAttr(resourceName, "enable_group_sync", "true"),
					resource.TestCheckResourceAttr(resourceName, "allow_non_provisioned_users", "true"),
				),
			},
			{
				Config: testAccSCIMConfigResourceConfig(true, true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable_user_sync", "true"),
					resource.TestCheckResourceAttr(resourceName, "enable_group_sync", "true"),
					resource.TestCheckResourceAttr(resourceName, "allow_non_provisioned_users", "false"),
				),
			},
			{
				Config: testAccSCIMConfigResourceConfig(false, false, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable_user_sync", "false"),
					resource.TestCheckResourceAttr(resourceName, "enable_group_sync", "false"),
					resource.TestCheckResourceAttr(resourceName, "allow_non_provisioned_users", "true"),
				),
			},
		},
	})
}

func TestAccResourceSCIMConfig_import(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.0.0")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSCIMConfigResourceConfig(true, false, false),
			},
			{
				ResourceName:      "grafana_scim_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSCIMConfigResourceConfig(enableUserSync, enableGroupSync, allowNonProvisionedUsers bool) string {
	return fmt.Sprintf(`resource "grafana_scim_config" "test" {
  enable_user_sync  = %t
  enable_group_sync = %t
  allow_non_provisioned_users = %t
}
`, enableUserSync, enableGroupSync, allowNonProvisionedUsers)
}

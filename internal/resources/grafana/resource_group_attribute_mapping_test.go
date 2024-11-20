package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGroupAttributeSync_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=11.4.0")

	var groupMapping models.Group
	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             groupAttrMappingCheckExists.destroyed(&groupMapping, nil),
		Steps: []resource.TestStep{
			{
				// Can create a group attribute mapping with multiple roles
				Config: testAccGroupAttributeSyncConfig("grafana_role.role_1.uid", "grafana_role.role_2.uid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					groupAttrMappingCheckExists.exists("grafana_group_attribute_mapping.test", &groupMapping),
					roleCheckExists.exists("grafana_role.role_1", &role),
					roleCheckExists.exists("grafana_role.role_2", &role),
					roleCheckExists.exists("grafana_role.role_3", &role),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "id", "1:test_group_id"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "group_id", "test_group_id"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "role_uids.#", "2"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "role_uids.0", "role_1_uid"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "role_uids.1", "role_2_uid"),
					testutils.CheckLister("grafana_group_attribute_mapping.test"),
				),
			},
			{
				// Can update a group attribute mapping with multiple roles
				Config: testAccGroupAttributeSyncConfig("grafana_role.role_1.uid", "grafana_role.role_3.uid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					groupAttrMappingCheckExists.exists("grafana_group_attribute_mapping.test", &groupMapping),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "id", "1:test_group_id"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "role_uids.#", "2"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "role_uids.0", "role_1_uid"),
					resource.TestCheckResourceAttr("grafana_group_attribute_mapping.test", "role_uids.1", "role_3_uid"),
				),
			},
			{
				// Can import a group attribute mapping
				ImportState:       true,
				ResourceName:      "grafana_group_attribute_mapping.test",
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGroupAttributeSyncConfig(roleUID1, roleUID2 string) string {
	return fmt.Sprintf(`
resource "grafana_role" "role_1" {
  name  = "role_1"
  uid = "role_1_uid"
  auto_increment_version = true
  permissions {
	action = "teams:read"
	scope = "teams:*"
  }
}

resource "grafana_role" "role_2" {
  name  = "role_2"
  uid = "role_2_uid"
  auto_increment_version = true
  permissions {
	action = "teams:create"
 }
}

resource "grafana_role" "role_3" {
  name  = "role_3"
  uid = "role_3_uid"
  auto_increment_version = true
  permissions {
	action = "teams:write"
	scope = "teams:*"
  }
}

resource "grafana_group_attribute_mapping" "test" {
  group_id = "test_group_id"
  role_uids = [%s, %s]
}
`, roleUID1, roleUID2)
}

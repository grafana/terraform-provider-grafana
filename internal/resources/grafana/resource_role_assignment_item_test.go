package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccRoleAssignmentItem(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	testName := acctest.RandString(10)
	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             roleAssignmentCheckExists.destroyed(&role, nil),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentItemConfig(testName),
				Check: resource.ComposeTestCheckFunc(
					roleAssignmentCheckExists.exists("grafana_role.test", &role),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_role_assignment_item.user1",
				ImportStateVerify: true,
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_role_assignment_item.team",
				ImportStateVerify: true,
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_role_assignment_item.service_account",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRoleAssignmentItem_inOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	testName := acctest.RandString(10)
	var org models.OrgDetailsDTO
	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentItemConfigInOrg(testName),
				Check: resource.ComposeTestCheckFunc(
					roleAssignmentCheckExists.exists("grafana_role.test", &role),

					// Check that the role is in the correct organization
					resource.TestMatchResourceAttr("grafana_role.test", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_role.test", "grafana_organization.test"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_role_assignment_item.user1",
				ImportStateVerify: true,
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_role_assignment_item.team",
				ImportStateVerify: true,
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_role_assignment_item.service_account",
				ImportStateVerify: true,
			},
			// Check destroy
			{
				Config: testutils.WithoutResource(t,
					roleAssignmentItemConfigInOrg(testName),
					"grafana_role_assignment_item.user1",
					"grafana_role_assignment_item.user2",
					"grafana_role_assignment_item.team",
					"grafana_role_assignment_item.service_account",
				),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					roleAssignmentCheckExists.destroyed(&role, nil),
				),
			},
		},
	})
}

func TestAccRoleAssignmentItem_withCloudServiceAccount(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	testName := acctest.RandString(10)
	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             roleAssignmentCheckExists.destroyed(&role, nil),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentItemConfigWithCloudServiceAccount(testName),
				Check: resource.ComposeTestCheckFunc(
					roleAssignmentCheckExists.exists("grafana_role.test", &role),
				),
			},
		},
	})
}

func roleAssignmentItemConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_role" "test" {
	name  = "%[1]s" 
	description = "test desc"
	version = 1
	uid = "%[1]s"
	global = true
	group = "testgroup"
	display_name = "testdisplay"
	hidden = true
  }

resource "grafana_team" "test_team" {
	name = "%[1]s"
}

resource "grafana_user" "test_user" {
	email = "%[1]s-1@test.com"
	login    = "%[1]s-1@test.com"
	password = "12345"
}

resource "grafana_user" "test_user2" {
	email = "%[1]s-2@test.com"
	login    = "%[1]s-2@test.com"
	password = "12345"
}

resource "grafana_service_account" "test" {
	name        = "%[1]s-terraform-test"
	role        = "Editor"
	is_disabled = false
  }

  resource grafana_role_assignment_item "user1" {
	role_uid = grafana_role.test.uid
	user_id  = grafana_user.test_user.id
}

resource grafana_role_assignment_item "user2" {
	role_uid = grafana_role.test.uid
	user_id  = grafana_user.test_user2.id
}

resource grafana_role_assignment_item "team" {
	role_uid = grafana_role.test.uid
	team_id = grafana_team.test_team.id
}

resource grafana_role_assignment_item "service_account" {
	role_uid = grafana_role.test.uid
	service_account_id = grafana_service_account.test.id
}
`, name)
}

func roleAssignmentItemConfigInOrg(name string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_role" "test" {
	org_id = grafana_organization.test.id

	name  = "%[1]s"
	description = "test desc"
	version = 1
	uid = "%[1]s"
	global = false
	group = "testgroup"
	display_name = "testdisplay"
	hidden = true
  }

resource "grafana_team" "test_team" {
	org_id = grafana_organization.test.id
	name = "%[1]s"
}

resource "grafana_user" "test_user" {
	email = "%[1]s-1@test.com"
	login    = "%[1]s-1@test.com"
	password = "12345"
}

resource "grafana_user" "test_user2" {
	email = "%[1]s-2@test.com"
	login    = "%[1]s-2@test.com"
	password = "12345"
}

resource "grafana_service_account" "test" {
	org_id = grafana_organization.test.id

	name        = "%[1]s-terraform-test"
	role        = "Editor"
	is_disabled = false
  }

resource grafana_role_assignment_item "user1" {
	org_id = grafana_organization.test.id
	role_uid = grafana_role.test.uid
	user_id  = grafana_user.test_user.id
}

resource grafana_role_assignment_item "user2" {
	org_id = grafana_organization.test.id
	role_uid = grafana_role.test.uid
	user_id  = grafana_user.test_user2.id
}

resource grafana_role_assignment_item "team" {
	org_id = grafana_organization.test.id
	role_uid = grafana_role.test.uid
	team_id = grafana_team.test_team.id
}

resource grafana_role_assignment_item "service_account" {
	org_id = grafana_organization.test.id
	role_uid = grafana_role.test.uid
	service_account_id = grafana_service_account.test.id
}
`, name)
}

func roleAssignmentItemConfigWithCloudServiceAccount(name string) string {
	return fmt.Sprintf(`
resource "grafana_role" "test" {
	name  = "%[1]s" 
	description = "test desc"
	version = 1
	uid = "%[1]s"
	global = true
	group = "testgroup"
	display_name = "testdisplay"
	hidden = true
}

// Use a terraform_data resource to simulate a cloud service account ID
resource "terraform_data" "mock_cloud_sa_id" {
    input = "mockstack:123"
}

resource "grafana_role_assignment_item" "cloud_service_account" {
	role_uid = grafana_role.test.uid
	service_account_id = terraform_data.mock_cloud_sa_id.output
}
`, name)
}

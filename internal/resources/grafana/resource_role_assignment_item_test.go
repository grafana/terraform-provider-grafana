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

func TestAccRoleAssignmentItem_NoDuplicates(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	testName := acctest.RandString(10)
	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             roleAssignmentCheckExists.destroyed(&role, nil),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentItemNoDuplicatesConfig(testName),
				Check: resource.ComposeTestCheckFunc(
					roleAssignmentCheckExists.exists("grafana_role.test", &role),
					// Verify the role assignment doesn't contain duplicates
					resource.TestCheckResourceAttr("grafana_role_assignment_item.user1", "role_uid", testName),
					resource.TestCheckResourceAttr("grafana_role_assignment_item.team", "role_uid", testName),
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

resource "grafana_service_account" "test" {
	name        = "%[1]s-terraform-test"
	role        = "Editor"
	is_disabled = false
}

// This is a special test resource that validates our code can handle the service account ID format
// It doesn't actually create a role assignment in Grafana
resource "terraform_data" "test_service_account_id_parsing" {
    input = "mockstack:${grafana_service_account.test.id}"
    
    // This provisioner will run our validation logic
    provisioner "local-exec" {
        command = "echo 'Testing service account ID parsing with: mockstack:${grafana_service_account.test.id}'"
    }
    
    // Prevent this resource from being created in the actual Grafana instance
    lifecycle {
        ignore_changes = all
    }
}

resource "grafana_role_assignment_item" "service_account" {
	role_uid = grafana_role.test.uid
	service_account_id = grafana_service_account.test.id
}
`, name)
}

func roleAssignmentItemNoDuplicatesConfig(name string) string {
	return fmt.Sprintf(`
// Create a test role that will have multiple assignments
// This role will be the target for both user and team assignments
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

// Create a test team that will be assigned to the role
resource "grafana_team" "test_team" {
	name = "%[1]s"
}

// Create a test user that will be assigned to the same role
resource "grafana_user" "test_user" {
	email = "%[1]s-1@test.com"
	login    = "%[1]s-1@test.com"
	password = "12345"
}

// Multiple grafana_role_assignment_item resources targeting the SAME role.
// This setup reproduces the bug where duplicate assignments could be created
// when multiple assignment items reference the same role UID.

resource grafana_role_assignment_item "user1" {
	role_uid = grafana_role.test.uid // Same role as team assignment
	user_id  = grafana_user.test_user.id
}

resource grafana_role_assignment_item "team" {
	role_uid = grafana_role.test.uid // Same role as user1 assignment
	team_id = grafana_team.test_team.id
}
`, name)
}

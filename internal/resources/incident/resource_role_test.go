package incident_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccIncidentRole_basic(t *testing.T) {
	testutils.CheckIncidentTestsEnabled(t)

	// Captured during the run so CheckDestroy can look the role up after the
	// resource is gone from state.
	var roleID string

	// Not parallel: the Incident API caps an organization at 5 active roles, so
	// concurrent role tests would exhaust the limit.
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkIncidentRoleDestroyed(&roleID),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_role/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					captureIncidentRoleID("grafana_incident_role.test", &roleID),
					resource.TestCheckResourceAttrSet("grafana_incident_role.test", "id"),
					resource.TestCheckResourceAttr("grafana_incident_role.test", "name", "tf-acc-test-role"),
					resource.TestCheckResourceAttr("grafana_incident_role.test", "description", "Terraform acceptance test role."),
					resource.TestCheckResourceAttr("grafana_incident_role.test", "important", "false"),
					resource.TestCheckResourceAttr("grafana_incident_role.test", "mandatory", "false"),
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "false"),
					testutils.CheckLister("grafana_incident_role.test"),
				),
			},
			{
				ResourceName:      "grafana_incident_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_role/_acc_basic.tf", map[string]string{
					"tf-acc-test-role": "tf-acc-test-role-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "name", "tf-acc-test-role-updated"),
				),
			},
			// Archiving and unarchiving go through ArchiveRole/UnarchiveRole
			// rather than UpdateRole. Both require the organization to be
			// within the active-role bounds: archiving needs more than 2 active
			// roles, unarchiving needs fewer than 5.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_role/_acc_archived.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "true"),
					resource.TestCheckResourceAttr("grafana_incident_role.test", "name", "tf-acc-test-role"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_role/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "false"),
				),
			},
		},
	})
}

// captureIncidentRoleID records the role's ID from Terraform state so it can be
// checked after destroy, when the resource is no longer in state.
func captureIncidentRoleID(resourceName string, dest *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		id := rs.Primary.Attributes["id"]
		if id == "" {
			return fmt.Errorf("resource %s has no id attribute", resourceName)
		}
		*dest = id
		return nil
	}
}

// checkIncidentRoleDestroyed asserts the role is really gone from the API. The
// roles API has no get-by-ID, so this lists and filters.
func checkIncidentRoleDestroyed(roleID *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if *roleID == "" {
			return errors.New("no incident role ID was captured during the test")
		}
		id, err := strconv.Atoi(*roleID)
		if err != nil {
			return fmt.Errorf("captured incident role ID %q is not numeric: %w", *roleID, err)
		}

		client := testutils.Provider.Meta().(*common.Client).IncidentClient
		roles, err := incident.NewRolesService(client).GetRoles(context.Background(), incident.GetRolesRequest{})
		if err != nil {
			return fmt.Errorf("listing incident roles: %w", err)
		}
		for _, role := range roles.Roles {
			if role.RoleID == id {
				return fmt.Errorf("incident role %d still exists after destroy", id)
			}
		}
		return nil
	}
}

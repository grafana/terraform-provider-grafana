package incident_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// roleDescriptionConfig renders the role under test with the given description
// line, so a step can set it to a value, to "", or leave it out entirely.
func roleDescriptionConfig(description string) string {
	return fmt.Sprintf(`
resource "grafana_incident_role" "test" {
  name = "tf-unit-test-role"
  %s
}
`, description)
}

// TestUnitIncidentRole_EmptyDescription pins how an empty description is
// mapped. The API stores an unset description as "", so the resource has to
// keep "" when the practitioner wrote description = "" and null when they left
// it out. Getting either wrong fails the apply with an inconsistent result.
func TestUnitIncidentRole_EmptyDescription(t *testing.T) {
	api := newFakeRolesAPI(activeRoles("commander", "investigator", "observer")...)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleDescriptionConfig(`description = ""`),
				Check:  resource.TestCheckResourceAttr("grafana_incident_role.test", "description", ""),
			},
			{
				Config: roleDescriptionConfig(`description = "Terraform unit test role."`),
				Check:  resource.TestCheckResourceAttr("grafana_incident_role.test", "description", "Terraform unit test role."),
			},
			{
				Config: roleDescriptionConfig(`description = ""`),
				Check:  resource.TestCheckResourceAttr("grafana_incident_role.test", "description", ""),
			},
			{
				Config: roleDescriptionConfig(""),
				Check:  resource.TestCheckNoResourceAttr("grafana_incident_role.test", "description"),
			},
			// Import has no configuration to go on, so an empty description
			// comes back as null, matching the config above.
			{
				ResourceName:      "grafana_incident_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

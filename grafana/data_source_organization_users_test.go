package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganizationUsers(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var organization gapi.Org
	orgName := "datasource-org-users-test-org"
	email := "foo@example.com"
	checks := []resource.TestCheckFunc{
		testAccOrganizationCheckExists("grafana_organization.test", &organization),
		resource.TestCheckResourceAttr(
			"data.grafana_organization_users.from_name", "organization_name", orgName,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization_users.from_name", "emails.#", "2",
		),
		resource.TestCheckResourceAttr(
			// Grafana automatically adds the admin user to all orgs
			"data.grafana_organization_users.from_name", "emails.0", "admin@localhost",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization_users.from_name", "emails.1", email,
		),
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationCheckDestroy(&organization),
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourceOrganizationUsersConfig(orgName, email),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func testAccDatasourceOrganizationUsersConfig(orgName, email string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
  name         = "%s"
  admin_user   = "admin"
	create_users = true
  viewers = [
    "%s"
  ]
}

data "grafana_organization_users" "from_name" {
  organization_name = grafana_organization.test.name
}
`, orgName, email)
}

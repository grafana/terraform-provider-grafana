package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceUser(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var user gapi.User
	checks := []resource.TestCheckFunc{
		testAccUserCheckExists("grafana_user.test", &user),
	}
	for _, rName := range []string{"from_email", "from_login", "from_id"} {
		checks = append(checks,
			resource.TestMatchResourceAttr(
				"data.grafana_user."+rName, "user_id", idRegexp,
			),
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "email", "test.datasource@example.com",
			),
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "name", "Testing Datasource",
			),
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "login", "test-datasource",
			),
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "is_admin", "true",
			),
		)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccUserCheckDestroy(&user),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_user/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccDatasourceUserPagination(t *testing.T) {
	CheckOSSTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccLotsOfUsers},
			{
				Config: testAccLotsOfUsersWithDatasource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.grafana_user.from_login", "user_id", idRegexp,
					),
					resource.TestCheckResourceAttr(
						"data.grafana_user.from_login", "email", "user-1200@example.com",
					),
					resource.TestCheckResourceAttr(
						"data.grafana_user.from_login", "name", "user-1200",
					),
					resource.TestCheckResourceAttr(
						"data.grafana_user.from_login", "login", "user-1200@example.com",
					),
					resource.TestCheckResourceAttr(
						"data.grafana_user.from_login", "is_admin", "true",
					),
				),
			},
		},
	})

}

const testAccLotsOfUsers = `
resource "grafana_user" "users" {
	count    = 1500
	name     = "user-${count.index}"
	login    = "user-${count.index}@example.com"
	email    = "user-${count.index}@example.com"
	is_admin = true
	password = "anything"
}
`

var testAccLotsOfUsersWithDatasource = testAccLotsOfUsers + `
data "grafana_user" "from_login" {
  login = "user-1200@example.com"
}
`

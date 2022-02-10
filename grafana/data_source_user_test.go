package grafana

import (
	"testing"

	gapi "github.com/albeego/grafana-api-golang-client"
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
		PreCheck:          func() { testAccPreCheck(t) },
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

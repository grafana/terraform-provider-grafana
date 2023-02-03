package provider

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceUser(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

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

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccUserCheckDestroy(&user),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_user/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

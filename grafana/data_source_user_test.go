package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceUser(t *testing.T) {
	checks := []resource.TestCheckFunc{}
	for _, rName := range []string{"from_email", "from_login", "from_id"} {
		checks = append(checks,
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "email", "staff.name@example.com",
			),
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "name", "Staff Name",
			),
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "login", "staff",
			),
			resource.TestCheckResourceAttr(
				"data.grafana_user."+rName, "is_admin", "false",
			),
		)
	}

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_user/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

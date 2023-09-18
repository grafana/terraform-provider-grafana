package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPublicDashboard_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=10.2.0") // Dashboard UIDs are only available as references in Grafana 9+

	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard_public/resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccPublicDashboardCheckExistsUID("grafana_dashboard_public.my_public_dashboard"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "uid", "my-custom-public-uid"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "dashboard_uid", "my-dashboard-uid"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "access_token", "e99e4275da6f410d83760eefa934d8d2"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "is_enabled", "true"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "share", "public"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "time_selection_enabled", "true"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "annotations_enabled", "true"),
					checkResourceIsInOrg("grafana_dashboard_public.my_public_dashboard", "grafana_organization.my_org"),

					// my_public_dashboard2 belong to a different org_id
					checkResourceIsInOrg("grafana_dashboard_public.my_public_dashboard2", "grafana_organization.my_org2"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard2", "dashboard_uid", "my-dashboard-uid2"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard2", "is_enabled", "false"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard2", "share", "public"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard2", "time_selection_enabled", "false"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard2", "annotations_enabled", "false"),
				),
			},
			{
				ResourceName:      "grafana_dashboard_public.my_public_dashboard",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_dashboard_public.my_public_dashboard2",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPublicDashboardCheckExistsUID(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource id not set")
		}

		orgID, dashboardUID, _ := grafana.SplitPublicDashboardID(rs.Primary.ID)

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)
		pd, err := client.PublicDashboardbyUID(dashboardUID)
		if pd == nil || err != nil {
			return fmt.Errorf("Error getting public dashboard: %s", err)
		}

		return nil
	}
}

package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPublicDashboard_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.2.0")

	var publicDashboard models.PublicDashboard
	var org models.OrgDetailsDTO
	var publicDashboardOrg models.PublicDashboard

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard_public/resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardPublicCheckExists.exists("grafana_dashboard_public.my_public_dashboard", &publicDashboard),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "uid", "my-custom-public-uid"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "dashboard_uid", "my-dashboard-uid"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "access_token", "e99e4275da6f410d83760eefa934d8d2"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "is_enabled", "true"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "share", "public"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "time_selection_enabled", "true"),
					resource.TestCheckResourceAttr("grafana_dashboard_public.my_public_dashboard", "annotations_enabled", "true"),
					checkResourceIsInOrg("grafana_dashboard_public.my_public_dashboard", "grafana_organization.my_org"),

					// my_public_dashboard2 belong to a different org_id
					dashboardPublicCheckExists.exists("grafana_dashboard_public.my_public_dashboard2", &publicDashboardOrg),
					orgCheckExists.exists("grafana_organization.my_org2", &org),
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
			// Destroy both public dashboards
			{
				Config: testutils.WithoutResource(t, testutils.TestAccExample(t, "resources/grafana_dashboard_public/resource.tf"),
					"grafana_dashboard_public.my_public_dashboard",
					"grafana_dashboard_public.my_public_dashboard2",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardPublicCheckExists.destroyed(&publicDashboard, nil),
					dashboardPublicCheckExists.destroyed(&publicDashboardOrg, &org),
				),
			},
		},
	})
}

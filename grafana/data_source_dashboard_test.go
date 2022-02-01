package grafana

import (
	"regexp"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceDashboardBasicID(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard
	checks := []resource.TestCheckFunc{
		testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
		resource.TestCheckResourceAttr(
			"data.grafana_dashboard.from_id", "title", "Production Overview",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_dashboard.from_id", "dashboard_id", idRegexp,
		),
		resource.TestMatchResourceAttr(
			"data.grafana_dashboard.from_id", "uid", uidRegexp,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_dashboard.from_uid", "title", "Production Overview",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_dashboard.from_uid", "dashboard_id", idRegexp,
		),
		resource.TestMatchResourceAttr(
			"data.grafana_dashboard.from_uid", "uid", uidRegexp,
		),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_dashboard/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccDatasourceDashboardBasicUID(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			{
				Config:      testAccExample(t, "data-sources/grafana_dashboard/bad-ExactlyOneOf.tf"),
				ExpectError: regexp.MustCompile(".*one of.*must be specified.*"),
			},
		},
	})
}

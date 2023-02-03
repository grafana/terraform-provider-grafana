package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganizationPreferences(t *testing.T) {
	CheckOSSTestsEnabled(t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(
			"data.grafana_organization_preferences.test", "theme", "",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization_preferences.test", "timezone", "",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization_preferences.test", "id", "organization_preferences",
		),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_organization_preferences/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

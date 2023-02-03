package provider

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganizationPreferences(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

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
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_organization_preferences/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

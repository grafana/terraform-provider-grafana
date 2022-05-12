package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganization(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var organization gapi.Org
	checks := []resource.TestCheckFunc{
		testAccOrganizationCheckExists("grafana_organization.test", &organization),
		resource.TestCheckResourceAttr(
			"data.grafana_organization.from_name", "name", "test-org",
		),
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationCheckDestroy(&organization),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_organization/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

package slo_test

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var slo gapi.Slo
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccSloCheckDestroy(&slo),
		Steps: []resource.TestStep{
			{
				// Creates a SLO Resource
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "id"),
					resource.TestCheckResourceAttr("grafana_slo.test", "name", "Terraform Testing"),
					resource.TestCheckResourceAttr("grafana_slo.test", "description", "Terraform Description"),
				),
			},
			{
				// Verifies that the created SLO Resource is read by the Datasource Read Method
				Config: testutils.TestAccExample(t, "data-sources/grafana_slos/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(

					resource.TestCheckResourceAttrSet("data.grafana_slos.slos", "slos.0.uuid"),
					resource.TestCheckResourceAttr("data.grafana_slos.slos", "slos.0.name", "Terraform Testing"),
					resource.TestCheckResourceAttr("data.grafana_slos.slos", "slos.0.description", "Terraform Description"),
				),
			},
		},
	})
}

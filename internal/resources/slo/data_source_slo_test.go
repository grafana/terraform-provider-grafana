package slo_test

import (
	"testing"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := acctest.RandomWithPrefix("SLO Terraform Testing")

	var slo slo.SloV00Slo
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccSloCheckDestroy(&slo),
		Steps: []resource.TestStep{
			{
				// Creates a SLO Resource
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_slo/resource.tf", map[string]string{
					"Terraform Testing": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "id"),
					resource.TestCheckResourceAttr("grafana_slo.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_slo.test", "description", "Terraform Description"),
				),
			},
			{
				// Verifies that the created SLO Resource is read by the Datasource Read Method
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_slos/data-source.tf", map[string]string{
					"Terraform Testing": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_slos.slos", "slos.0.uuid"),
					resource.TestCheckResourceAttrSet("data.grafana_slos.slos", "slos.0.name"),
				),
			},
		},
	})
}

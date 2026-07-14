package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccDataSourceK6ProjectAllowedLoadZones_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	projectName := "Terraform Project Test Allowed Load Zones " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_project_allowed_load_zones/data-source.tf", map[string]string{
					"Terraform Project Test Allowed Load Zones": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					// Check that allowed_load_zones is an empty array
					// Private load zone cannot be created using terraform, that's the only thing we can test for
					resource.TestCheckResourceAttr("data.grafana_k6_project_allowed_load_zones.from_project_id", "allowed_load_zones.#", "0"),
				),
			},
		},
	})
}

package grafana_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceDataSources_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet("data.grafana_data_sources.all", "data_sources.#"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "grafana_data_source" "test" {
	type = "prometheus"
	name = "test-data-sources-ds"
	url  = "http://localhost:9090"
}
data "grafana_data_sources" "all" {
    depends_on = [grafana_data_source.test]
}
				`,
				Check: resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

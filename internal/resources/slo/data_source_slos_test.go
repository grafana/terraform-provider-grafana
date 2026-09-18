package slo_test

import (
	"fmt"
	"testing"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
			{
				// The data source must surface source_datasource_uid. It shares
				// freeformQueryModel with the resource, so a schema that omits the
				// attribute fails at read time rather than producing a wrong value.
				Config: sloWithSourceDatasourceAndDataSource(randomName + " - DS Source"),
				Check: resource.ComposeTestCheckFunc(
					checkSlosDataSourceAttrByName(
						"data.grafana_slos.slos",
						randomName+" - DS Source",
						"query.0.freeform.0.source_datasource_uid",
					),
				),
			},
		},
	})
}

func sloWithSourceDatasourceAndDataSource(name string) string {
	return fmt.Sprintf(`
resource "grafana_data_source" "slo_source" {
  type = "prometheus"
  name = "%[1]s"
  url  = "https://prometheus.example.com/"
}

resource "grafana_slo" "ds_source" {
  description = "%[1]s"
  name        = "%[1]s"
  objectives {
    value  = 0.995
    window = "28d"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  query {
    type = "freeform"
    freeform {
      query                 = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
      source_datasource_uid = grafana_data_source.slo_source.uid
    }
  }
}

data "grafana_slos" "slos" {
  depends_on = [grafana_slo.ds_source]
}
`, name)
}

// checkSlosDataSourceAttrByName locates an SLO in the grafana_slos list by name
// rather than by index — the API returns every SLO in the instance in no
// guaranteed order, so slos.0 is not reliably the one the test created — and
// asserts the given nested attribute is set on it.
func checkSlosDataSourceAttrByName(dataSourceName, sloName, attrSuffix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("data source not found: %s", dataSourceName)
		}

		count := rs.Primary.Attributes["slos.#"]
		if count == "" || count == "0" {
			return fmt.Errorf("%s returned no SLOs", dataSourceName)
		}

		for i := 0; ; i++ {
			key := fmt.Sprintf("slos.%d.name", i)
			got, ok := rs.Primary.Attributes[key]
			if !ok {
				break
			}
			if got != sloName {
				continue
			}

			attrKey := fmt.Sprintf("slos.%d.%s", i, attrSuffix)
			value, ok := rs.Primary.Attributes[attrKey]
			if !ok || value == "" {
				return fmt.Errorf("%s: %s is not set on SLO %q", dataSourceName, attrSuffix, sloName)
			}
			return nil
		}

		return fmt.Errorf("%s: no SLO named %q in the %s returned", dataSourceName, sloName, count)
	}
}

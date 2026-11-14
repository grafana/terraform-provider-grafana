package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceDatasource_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource models.DataSource
	checks := []resource.TestCheckFunc{
		datasourceCheckExists.exists("grafana_data_source.prometheus", &dataSource),

		resource.TestMatchResourceAttr("data.grafana_data_source.from_name", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_name", "name", "prometheus-ds-test"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_name", "uid", "prometheus-ds-test-uid"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_name", "json_data_encoded", `{"httpMethod":"POST","prometheusType":"Mimir","prometheusVersion":"2.4.0"}`),

		resource.TestMatchResourceAttr("data.grafana_data_source.from_uid", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_uid", "name", "prometheus-ds-test"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_uid", "uid", "prometheus-ds-test-uid"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_uid", "json_data_encoded", `{"httpMethod":"POST","prometheusType":"Mimir","prometheusVersion":"2.4.0"}`),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_data_source/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccDataSources_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	name := acctest.RandString(10)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_organization" "test" {
  name = "%s"
}

resource "grafana_data_source" "metrics" {
  org_id = grafana_organization.test.id
  name   = "Metrics"
  uid    = "metrics"
  type   = "prometheus"
  url    = "http://prometheus:9090"
}

resource "grafana_data_source" "logs" {
  org_id = grafana_organization.test.id
  name   = "Logs"
  uid    = "logs"
  type   = "loki"
  url    = "http://loki:3100"
}

data "grafana_data_sources" "test" {
  org_id     = grafana_organization.test.id
  depends_on = [grafana_data_source.metrics, grafana_data_source.logs]
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.grafana_data_sources.test", "org_id", "grafana_organization.test", "id"),
					resource.TestCheckResourceAttr("data.grafana_data_sources.test", "data_sources.#", "2"),
					resource.TestCheckResourceAttr("data.grafana_data_sources.test", "data_sources.0.name", "Logs"),
					resource.TestCheckResourceAttr("data.grafana_data_sources.test", "data_sources.0.uid", "logs"),
					resource.TestCheckResourceAttr("data.grafana_data_sources.test", "data_sources.1.name", "Metrics"),
					resource.TestCheckResourceAttr("data.grafana_data_sources.test", "data_sources.1.uid", "metrics"),
				),
			},
		},
	})
}

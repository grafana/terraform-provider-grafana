package provider

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceDatasource(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var dataSource gapi.DataSource
	checks := []resource.TestCheckFunc{
		testAccDataSourceCheckExists("grafana_data_source.prometheus", &dataSource),

		resource.TestMatchResourceAttr("data.grafana_data_source.from_name", "id", idRegexp),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_name", "name", "prometheus-ds-test"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_name", "uid", "prometheus-ds-test-uid"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_name", "json_data_encoded", `{"httpMethod":"POST","prometheusType":"Mimir","prometheusVersion":"2.4.0"}`),

		resource.TestMatchResourceAttr("data.grafana_data_source.from_uid", "id", idRegexp),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_uid", "name", "prometheus-ds-test"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_uid", "uid", "prometheus-ds-test-uid"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_uid", "json_data_encoded", `{"httpMethod":"POST","prometheusType":"Mimir","prometheusVersion":"2.4.0"}`),

		resource.TestMatchResourceAttr("data.grafana_data_source.from_id", "id", idRegexp),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_id", "name", "prometheus-ds-test"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_id", "uid", "prometheus-ds-test-uid"),
		resource.TestCheckResourceAttr("data.grafana_data_source.from_id", "json_data_encoded", `{"httpMethod":"POST","prometheusType":"Mimir","prometheusVersion":"2.4.0"}`),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_data_source/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

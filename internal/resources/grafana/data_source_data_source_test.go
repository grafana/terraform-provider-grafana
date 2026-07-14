package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
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

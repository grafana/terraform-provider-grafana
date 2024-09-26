package connections_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_DataSourceMetricsEndpointScrapeJob(t *testing.T) {
	// Run this test by removing t.Skip and set env variables TF_ACC=1;GRAFANA_CONNECTIONS_ACCESS_TOKEN=whatever
	// in order to test the resource code "scaffolding".
	// Expected result: test fails and the error message should be "failed to create metrics endpoint scrape job: failed to do request"
	// because the Connections API is not yet available.

	// t.Skip("will be enabled after Connections API is available in prod")

	// testutils.CheckCloudInstanceTestsEnabled(t) // TODO: enable after Connections API is available
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create this resource
				Config: testutils.TestAccExample(t, "resources/grafana_connections_metrics_endpoint_scrape_job/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "stack_id", "test-stack-id"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "name", "my-scrape-job"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_method", "basic"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_username", "my_username"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_password", "my_password"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "url", "https://dev.my-metrics-endpoint-url.com:9000/metrics"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "scrape_interval_seconds", "60"),
					testutils.CheckLister("grafana_connections_metrics_endpoint_scrape_job.test"),
				),
			},
			{
				// Verifies that the created SLO Resource is read by the Datasource Read Method
				// TODO: work on after other Test Step passes
				// Config:       testutils.TestAccExample(t, "data-sources/grafana_connections_metrics_endpoint_scrape_job/resource.tf"),
				// RefreshState: true,
				// Check: resource.ComposeTestCheckFunc(
				// 	resource.TestCheckResourceAttrSet("data.grafana_connections_metrics_endpoint_scrape_job.test", "test.0.name"),
				// ),
			},
		},
	})
}

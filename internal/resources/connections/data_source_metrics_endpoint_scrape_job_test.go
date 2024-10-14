package connections_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAcc_DataSourceMetricsEndpointScrapeJob2(t *testing.T) {
	// Mock the Connections API response for Create, Get, and Delete
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/metrics-endpoint/stacks/1/jobs/scrape-job-name", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`
				{
				  "data":{
					"name":"scrape-job-name",
					"authentication_method":"basic",
					"basic_username":"my-username",
					"basic_password":"my-password",
					"url":"https://dev.my-metrics-endpoint-url.com:9000/metrics",
					"scrape_interval_seconds":60,
					"flavor":"default",
					"enabled":true
				  }
				}`))
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				{
				  "data":{
				    "name":"scrape-job-name",
				    "authentication_method":"basic",
				    "url":"https://dev.my-metrics-endpoint-url.com:9000/metrics",
				    "scrape_interval_seconds":60,
				    "flavor":"default",
				    "enabled":true
				  }
				}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	require.NoError(t, os.Setenv("GRAFANA_CONNECTIONS_ACCESS_TOKEN", "some token"))
	require.NoError(t, os.Setenv("GRAFANA_CONNECTIONS_URL", server.URL))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_connections_metrics_endpoint_scrape_job/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_connections_metrics_endpoint_scrape_job.test", "stack_id"),
					resource.TestCheckResourceAttrSet("data.grafana_connections_metrics_endpoint_scrape_job.test", "name"),
					resource.TestCheckResourceAttrSet("data.grafana_connections_metrics_endpoint_scrape_job.test", "authentication_method"),
					resource.TestCheckNoResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_username"),
					resource.TestCheckNoResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_password"),
					resource.TestCheckResourceAttrSet("data.grafana_connections_metrics_endpoint_scrape_job.test", "url"),
					resource.TestCheckResourceAttrSet("data.grafana_connections_metrics_endpoint_scrape_job.test", "enabled"),
					resource.TestCheckResourceAttrSet("data.grafana_connections_metrics_endpoint_scrape_job.test", "scrape_interval_seconds"),
				),
			},
		},
	})
}

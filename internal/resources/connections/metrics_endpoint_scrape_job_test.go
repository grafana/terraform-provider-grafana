package connections_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// Tests both managed resource and data source
func TestAcc_MetricsEndpointScrapeJob(t *testing.T) {
	// Mock the Connections API response for Create, Get, and Delete
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stacks/1/metrics-endpoint/jobs/my-scrape-job", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`
				{
				  "data":{
					"name":"my-scrape-job",
					"authentication_method":"basic",
					"basic_username":"my-username",
					"basic_password":"my-password",
					"url":"https://grafana.com/metrics",
					"scrape_interval_seconds":120,
					"flavor":"default",
					"enabled":true
				  }
				}`))
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				{
				  "data":{
				    "name":"my-scrape-job",
				    "authentication_method":"basic",
				    "url":"https://grafana.com/metrics",
				    "scrape_interval_seconds":120,
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

	require.NoError(t, os.Setenv("GRAFANA_CONNECTIONS_API_ACCESS_TOKEN", "some token"))
	require.NoError(t, os.Setenv("GRAFANA_CONNECTIONS_API_URL", server.URL))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:       invalidScrapeJobBothAuthTypesUsed,
				PlanOnly:     true,
				RefreshState: false,
				ExpectError:  regexp.MustCompile(`These attributes cannot be configured together`),
			},
			{
				Config:       invalidScrapeJobMissingBasicPassword,
				PlanOnly:     true,
				RefreshState: false,
				ExpectError:  regexp.MustCompile(`Missing Required Field`),
			},
			{
				Config:       invalidScrapeJobMissingBasicUsernameAndPassword,
				PlanOnly:     true,
				RefreshState: false,
				ExpectError:  regexp.MustCompile(`Missing Required Field`),
			},
			{
				Config:       invalidScrapeJobUsingBasicWithToken,
				PlanOnly:     true,
				RefreshState: false,
				ExpectError:  regexp.MustCompile(`Missing Required Field`),
			},
			{
				Config:             resourceWithForEachValidURL,
				PlanOnly:           true,
				RefreshState:       false,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:       resourceWithForEachInvalidURL,
				PlanOnly:     true,
				RefreshState: false,
				ExpectError:  regexp.MustCompile(`A valid URL is required`),
			},
			{
				// Creates a managed resource
				Config: testutils.TestAccExample(t, "resources/grafana_connections_metrics_endpoint_scrape_job/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "stack_id", "1"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "name", "my-scrape-job"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_method", "basic"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_username", "my-username"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_password", "my-password"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "url", "https://grafana.com/metrics"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "scrape_interval_seconds", "120"),
				),
			},
			{
				// Tests data source resource
				Config: testutils.TestAccExample(t, "data-sources/grafana_connections_metrics_endpoint_scrape_job/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "stack_id", "1"),
					resource.TestCheckResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "name", "my-scrape-job"),
					resource.TestCheckResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "enabled", "true"),
					resource.TestCheckResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "authentication_method", "basic"),
					resource.TestCheckNoResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "authentication_basic_username"),
					resource.TestCheckNoResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "authentication_basic_password"),
					resource.TestCheckResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "url", "https://grafana.com/metrics"),
					resource.TestCheckResourceAttr("data.grafana_connections_metrics_endpoint_scrape_job.ds_test", "scrape_interval_seconds", "120"),
				),
			},
		},
	})
}

var invalidScrapeJobUsingBasicWithToken = `
resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                      = "1"
  name                          = "my-scrape-job"
  enabled                       = true
  authentication_method         = "basic"
  authentication_bearer_token   = "some-token"
  url                           = "https://grafana.com/metrics"
  scrape_interval_seconds       = 120
}
`

var invalidScrapeJobMissingBasicUsernameAndPassword = `
resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                      = "1"
  name                          = "my-scrape-job"
  enabled                       = true
  authentication_method         = "basic"
  url                           = "https://grafana.com/metrics"
  scrape_interval_seconds       = 120
}
`

var invalidScrapeJobMissingBasicPassword = `
resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                      = "1"
  name                          = "my-scrape-job"
  enabled                       = true
  authentication_method         = "basic"
  authentication_basic_username = "my-username"
  url                           = "https://grafana.com/metrics"
  scrape_interval_seconds       = 120
}
`

var invalidScrapeJobBothAuthTypesUsed = `
resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                      = "1"
  name                          = "my-scrape-job"
  enabled                       = true
  authentication_method         = "basic"
  authentication_basic_username = "my-username"
  authentication_basic_password = "my-password"
  authentication_bearer_token   = "my-token"
  url                           = "https://grafana.com/metrics"
  scrape_interval_seconds       = 120
}
`

var resourceWithForEachValidURL = `
locals {
  jobs = [
    { name = "test", url = "https://google.com" }
  ]
}

resource "grafana_connections_metrics_endpoint_scrape_job" "valid_url" {
  for_each = { for j in local.jobs : j.name => j.url }
  stack_id = "......"
  name = each.key
  enabled = false
  authentication_method = "bearer"
  authentication_bearer_token = "test"
  url = each.value
}
`

var resourceWithForEachInvalidURL = `
locals {
  jobs = [
    { name = "test", url = "" }
  ]
}

resource "grafana_connections_metrics_endpoint_scrape_job" "invalid_url" {
  for_each = { for j in local.jobs : j.name => j.url }
  stack_id = "......"
  name = each.key
  enabled = false
  authentication_method = "bearer"
  authentication_bearer_token = "test"
  url = each.value
}
`

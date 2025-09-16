package asserts_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccAssertsLogConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "envs_for_log.0", "test-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "false"),
				),
			},
			{
				ResourceName:            "grafana_asserts_log_config.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"log_config"},
			},
		},
	})
}

func TestAccAssertsLogConfig_update(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "envs_for_log.0", "test-env"),
				),
			},
			{
				Config: testAccAssertsLogConfigConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "true"),
				),
			},
		},
	})
}

func TestAccAssertsLogConfig_logConfigFull(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfigFull,
				Check: resource.ComposeTestCheckFunc(
					// top-level
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "name", "full-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "envs_for_log.0", "full-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "sites_for_log.0", "us-east-1"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "sites_for_log.1", "us-west-2"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "default_config", "true"),
					// log_config nested
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.tool", "loki"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.url", "https://logs.example.com"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.date_format", "RFC3339"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.correlation_labels", "trace_id,span_id"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.default_search_text", "error"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.error_filter", "level=error"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.columns.0", "timestamp"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.columns.1", "level"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.columns.2", "message"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.index", "logs-*"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.interval", "1h"),
					// map flattens to keys
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.query.job", "app"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.query.level", "error"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.sort.0", "timestamp desc"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.http_response_code_field", "status_code"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.org_id", "1"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.data_source", "loki"),
				),
			},
		},
	})
}

const testAccAssertsLogConfigConfig = `
resource "grafana_asserts_log_config" "test" {
  name           = "test-env"
  envs_for_log   = ["test-env"]
  default_config = false
  log_config {
    tool  = "loki"
    url   = "https://logs.example.com"
    index = "logs-*"
  }
}
`

const testAccAssertsLogConfigConfigUpdated = `
resource "grafana_asserts_log_config" "test" {
  name           = "test-env"
  envs_for_log   = ["test-env"]
  default_config = true
  log_config {
    tool  = "loki"
    url   = "https://logs.example.com"
    index = "logs-*"
  }
}
`

const testAccAssertsLogConfigConfigFull = `
resource "grafana_asserts_log_config" "full" {
  name           = "full-env"
  envs_for_log   = ["full-env"]
  sites_for_log  = ["us-east-1", "us-west-2"]
  default_config = true
  log_config {
    tool                 = "loki"
    url                  = "https://logs.example.com"
    date_format          = "RFC3339"
    correlation_labels   = "trace_id,span_id"
    default_search_text  = "error"
    error_filter         = "level=error"
    columns              = ["timestamp", "level", "message"]
    index                = "logs-*"
    interval             = "1h"
    query = {
      job   = "app"
      level = "error"
    }
    sort                   = ["timestamp desc"]
    http_response_code_field = "status_code"
    org_id                = "1"
    data_source           = "loki"
  }
}
`

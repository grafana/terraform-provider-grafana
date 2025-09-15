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
					resource.TestCheckResourceAttrSet("grafana_asserts_log_config.test", "config"),
				),
			},
			{
				ResourceName:      "grafana_asserts_log_config.test",
				ImportState:       true,
				ImportStateVerify: true,
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
				),
			},
			{
				Config: testAccAssertsLogConfigConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
				),
			},
		},
	})
}

const testAccAssertsLogConfigConfig = `
resource "grafana_asserts_log_config" "test" {
  name = "test-env"
  config = <<-EOT
    name: test-env
    logConfig:
      enabled: true
      retention: "7d"
  EOT
}
`

const testAccAssertsLogConfigConfigUpdated = `
resource "grafana_asserts_log_config" "test" {
  name = "test-env"
  config = <<-EOT
    name: test-env
    logConfig:
      enabled: true
      retention: "30d"
  EOT
}
`

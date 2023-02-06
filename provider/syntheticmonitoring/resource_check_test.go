package syntheticmonitoring

import (
	"context"
	"regexp"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceCheck_dns(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/dns_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "job", "DNS Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "target", "grafana.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "probes.0", "1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.server", "8.8.8.8"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.port", "53"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.record_type", "A"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.protocol", "UDP"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/dns_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "job", "DNS Updated"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "target", "grafana.net"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "probes.0", "14"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "probes.1", "19"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.ip_version", "Any"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.server", "8.8.4.4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.port", "8600"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.record_type", "CNAME"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.valid_r_codes.0", "NOERROR"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.valid_r_codes.1", "NOTAUTH"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.validate_answer_rrs.0.fail_if_matches_regexp.0", ".+-bad-stuff*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.validate_answer_rrs.0.fail_if_not_matches_regexp.0", ".+-good-stuff*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.validate_authority_rrs.0.fail_if_matches_regexp.0", ".+-bad-stuff*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.validate_authority_rrs.0.fail_if_not_matches_regexp.0", ".+-good-stuff*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.validate_additional_rrs.0.fail_if_matches_regexp.0", ".+-bad-stuff*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.validate_additional_rrs.0.fail_if_not_matches_regexp.0", ".+-good-stuff*"),
				),
			},
		},
	})
}

func TestAccResourceCheck_http(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "job", "HTTP Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "target", "https://grafana.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "probes.0", "1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.method", "GET"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.no_follow_redirects", "false"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/http_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "job", "HTTP Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "target", "https://grafana.org"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "probes.0", "15"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.ip_version", "V6"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.method", "TRACE"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.no_follow_redirects", "true"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.body", "and spirit"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.bearer_token", "asdfjkl;"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.proxy_url", "https://almost-there"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_ssl", "true"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.cache_busting_query_param_name", "pineapple"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.tls_config.0.server_name", "grafana.org"),
					resource.TestMatchResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.tls_config.0.client_cert", regexp.MustCompile((`^-{5}BEGIN CERTIFICATE`))),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.headers.0", "Content-Type: multipart/form-data; boundary=something"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.basic_auth.0.username", "open"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.basic_auth.0.password", "sesame"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.valid_status_codes.0", "200"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.valid_status_codes.1", "201"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.valid_http_versions.0", "HTTP/1.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.valid_http_versions.1", "HTTP/1.1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.valid_http_versions.2", "HTTP/2"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_body_matches_regexp.0", "*bad stuff*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_body_not_matches_regexp.0", "*good stuff*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_header_matches_regexp.0.header", "Content-Type"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_header_matches_regexp.0.regexp", "application/soap*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_header_matches_regexp.0.allow_missing", "true"),
				),
			},
		},
	})
}

func TestAccResourceCheck_ping(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/ping_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "job", "Ping Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "target", "grafana.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "probes.0", "1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.ip_version", "V4"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/ping_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "job", "Ping Updated"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "target", "grafana.net"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "probes.0", "14"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "probes.1", "19"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.ip_version", "Any"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.payload_size", "20"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.dont_fragment", "true"),
				),
			},
		},
	})
}

func TestAccResourceCheck_tcp(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/tcp_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "job", "TCP Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "target", "grafana.com:80"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "probes.0", "1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.tls", "false"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/tcp_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "job", "TCP Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "target", "grafana.com:443"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "probes.0", "14"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "probes.1", "19"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.ip_version", "V6"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.tls", "true"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.query_response.0.send", "howdy"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.query_response.0.expect", "hi"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.query_response.0.start_tls", "false"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.query_response.1.send", "like this"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.query_response.1.expect", "like that"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.query_response.1.start_tls", "true"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.tls_config.0.server_name", "grafana.com"),
					resource.TestMatchResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.tls_config.0.ca_cert", regexp.MustCompile((`^-{5}BEGIN CERTIFICATE`))),
				),
			},
		},
	})
}

func TestAccResourceCheck_traceroute(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/traceroute_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "job", "Traceroute defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "target", "grafana.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "probes.0", "1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_hops", "64"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_unknown_hops", "15"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.ptr_lookup", "true"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/traceroute_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "job", "Traceroute complex"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "target", "grafana.net"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "probes.0", "14"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "probes.1", "19"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_hops", "25"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_unknown_hops", "10"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.ptr_lookup", "false"),
				),
			},
		},
	})
}

// Test that a check is recreated if deleted outside the Terraform process
func TestAccResourceCheck_recreate(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf"),
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["grafana_synthetic_monitoring_check.http"]
					id, _ := strconv.ParseInt(rs.Primary.ID, 10, 64)
					return testutils.GetProvider().Meta().(*common.Client).SMAPI.DeleteCheck(context.Background(), id)
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "job", "HTTP Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "target", "https://grafana.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "probes.0", "1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.method", "GET"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.no_follow_redirects", "false"),
				),
			},
		},
	})
}

func TestAccResourceCheck_noSettings(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceCheck_noSettings,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("at least one check setting must be defined"),
			},
		},
	})
}

func TestAccResourceCheck_multiple(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceCheck_multiple,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("exactly one check setting must be defined, got 2"),
			},
		},
	})
}

const testAccResourceCheck_noSettings = `
data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "no_settings" {
  job       = "No Settings"
  target    = "grafana.com"
  enabled   = false
  frequency = 120000
  timeout   = 30000
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Atlanta,
  ]
  labels = {
    foo = "bar"
  }
  settings {

  }
}`

const testAccResourceCheck_multiple = `
data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "multiple" {
  job       = "No Settings"
  target    = "grafana.com"
  enabled   = false
  frequency = 120000
  timeout   = 30000
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Atlanta,
  ]
  labels = {
    foo = "bar"
  }
  settings {
	traceroute {}
	http {}
  }
}`

package grafana

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceSyntheticMonitoringCheck_dns(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/dns_basic.tf"),
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
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/dns_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "job", "DNS Updated"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "target", "grafana.net"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "probes.0", "2"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "probes.1", "3"),
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

func TestAccResourceSyntheticMonitoringCheck_http(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf"),
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
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/http_complex.tf"),
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
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.valid_http_versions.0", "HTTP/2"),
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

func TestAccResourceSyntheticMonitoringCheck_ping(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/ping_basic.tf"),
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
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/ping_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "job", "Ping Updated"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "target", "grafana.net"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "probes.0", "2"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "probes.1", "3"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.ip_version", "Any"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.payload_size", "20"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.dont_fragment", "true"),
				),
			},
		},
	})
}

func TestAccResourceSyntheticMonitoringCheck_tcp(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/tcp_basic.tf"),
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
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/tcp_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "job", "TCP Defaults"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "target", "grafana.com:443"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "probes.0", "2"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "probes.1", "3"),
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

func TestAccResourceSyntheticMonitoringCheck_traceroute(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/traceroute_basic.tf"),
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
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_check/traceroute_complex.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "job", "Traceroute complex"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "target", "grafana.net"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "probes.0", "2"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "probes.1", "3"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_hops", "25"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_unknown_hops", "10"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.ptr_lookup", "false"),
				),
			},
		},
	})
}

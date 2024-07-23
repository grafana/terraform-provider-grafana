package syntheticmonitoring_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAccResourceCheck_dns(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("dns")
	jobNameUpdated := acctest.RandomWithPrefix("dns")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/dns_basic.tf", map[string]string{
					`"DNS Defaults"`: strconv.Quote(jobName),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "target", "grafana.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "timeout", "3000"), // default
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.server", "8.8.8.8"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.port", "53"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.record_type", "A"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "settings.0.dns.0.protocol", "UDP"),
					testutils.CheckLister("grafana_synthetic_monitoring_check.dns"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/dns_complex.tf", map[string]string{
					`"DNS Updated"`: strconv.Quote(jobNameUpdated),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "job", jobNameUpdated),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.dns", "target", "grafana.net"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "probes.0"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.dns", "probes.1"),
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
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.dns",
			},
		},
	})
}

func TestAccResourceCheck_http(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("http")
	nameReplaceMap := map[string]string{
		`"HTTP Defaults"`: strconv.Quote(jobName),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "target", "https://grafana.com"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.method", "GET"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.no_follow_redirects", "false"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/http_complex.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "target", "https://grafana.org"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "probes.0"),
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
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.valid_http_versions.2", "HTTP/2.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_body_matches_regexp.0", ".*bad stuff.*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_body_not_matches_regexp.0", ".*good stuff.*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_header_matches_regexp.0.header", "Content-Type"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_header_matches_regexp.0.regexp", "application/soap*"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "settings.0.http.0.fail_if_header_matches_regexp.0.allow_missing", "true"),
				),
			},
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.http",
			},
		},
	})
}

func TestAccResourceCheck_ping(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("ping")
	jobNameUpdated := acctest.RandomWithPrefix("ping")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/ping_basic.tf", map[string]string{
					`"Ping Defaults"`: strconv.Quote(jobName),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "target", "grafana.com"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.ip_version", "V4"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/ping_complex.tf", map[string]string{
					`"Ping Updated"`: strconv.Quote(jobNameUpdated),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "job", jobNameUpdated),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "target", "grafana.net"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "probes.0"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.ping", "probes.1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.ip_version", "Any"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.payload_size", "20"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.ping", "settings.0.ping.0.dont_fragment", "true"),
				),
			},
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.ping",
			},
		},
	})
}

func TestAccResourceCheck_tcp(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("tcp")
	nameReplaceMap := map[string]string{
		`"TCP Defaults"`: strconv.Quote(jobName),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/tcp_basic.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "target", "grafana.com:80"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "settings.0.tcp.0.tls", "false"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/tcp_complex.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.tcp", "target", "grafana.com:443"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "probes.0"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.tcp", "probes.1"),
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
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.tcp",
			},
		},
	})
}

func TestAccResourceCheck_traceroute(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("traceroute")
	jobNameUpdated := acctest.RandomWithPrefix("traceroute")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/traceroute_basic.tf", map[string]string{
					`"Traceroute defaults"`: strconv.Quote(jobName),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "target", "grafana.com"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_hops", "64"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_unknown_hops", "15"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.ptr_lookup", "true"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/traceroute_complex.tf", map[string]string{
					`"Traceroute complex"`: strconv.Quote(jobNameUpdated),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "job", jobNameUpdated),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "target", "grafana.net"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "probes.0"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.traceroute", "probes.1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_hops", "25"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.max_unknown_hops", "10"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.traceroute", "settings.0.traceroute.0.ptr_lookup", "false"),
				),
			},
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.traceroute",
			},
		},
	})
}

func TestAccResourceCheck_multihttp(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("multihttp")
	jobNameUpdated := acctest.RandomWithPrefix("multihttp")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/multihttp_basic.tf", map[string]string{
					`"multihttp basic"`: strconv.Quote(jobName),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.multihttp", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.multihttp", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "target", "https://www.grafana-dev.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "timeout", "5000"), // multihttp has a default timeout of 5000
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.multihttp", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.method", "GET"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.url", "https://www.grafana-dev.com"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/multihttp_complex.tf", map[string]string{
					`"multihttp complex"`: strconv.Quote(jobNameUpdated),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.multihttp", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.multihttp", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "job", jobNameUpdated),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "target", "https://www.an-auth-endpoint.com"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.multihttp", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.method", "POST"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.url", "https://www.an-auth-endpoint.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.query_fields.1.name", "username"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.query_fields.1.value", "steve"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.query_fields.0.name", "password"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.query_fields.0.value", "top_secret"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.request.0.body.0.content_type", "application/json"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.assertions.0.type", "TEXT"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.assertions.0.subject", "HTTP_STATUS_CODE"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.assertions.0.condition", "EQUALS"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.assertions.0.value", "200"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.variables.0.type", "JSON_PATH"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.variables.0.name", "accessToken"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.0.variables.0.expression", "data.accessToken"),

					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.request.0.method", "GET"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.request.0.url", "https://www.an-endpoint-that-requires-auth.com"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.request.0.headers.0.name", "Authorization"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.request.0.headers.0.value", "Bearer ${accessToken}"),

					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.0.type", "TEXT"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.0.subject", "RESPONSE_BODY"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.0.condition", "CONTAINS"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.0.value", "foobar"),

					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.1.type", "TEXT"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.1.subject", "RESPONSE_BODY"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.1.condition", "NOT_CONTAINS"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.1.value", "xyyz"),

					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.2.type", "JSON_PATH_VALUE"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.2.condition", "EQUALS"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.2.expression", "$.slideshow.author"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.2.value", "Yours Truly"),

					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.3.type", "JSON_PATH_VALUE"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.3.condition", "STARTS_WITH"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.3.expression", "$.slideshow.date"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.3.value", "date of "),

					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.4.type", "JSON_PATH_ASSERTION"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.multihttp", "settings.0.multihttp.0.entries.1.assertions.4.expression", "$.slideshow.slides"),
				),
			},
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.multihttp",
			},
		},
	})
}

func TestAccResourceCheck_scripted(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Find and replace the path.module since it's not available in the test environment
	scriptFilepathAbs, err := filepath.Abs("../../../examples/resources/grafana_synthetic_monitoring_check")
	require.NoError(t, err)
	scriptFileContent, err := os.ReadFile(filepath.Join(scriptFilepathAbs, "script.js"))
	require.NoError(t, err)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("scripted")
	nameReplaceMap := map[string]string{
		`"Validate homepage"`: strconv.Quote(jobName),
		"${path.module}":      scriptFilepathAbs,
	}
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/scripted_basic.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.scripted", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.scripted", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.scripted", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.scripted", "target", "https://grafana.com/"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.scripted", "timeout", "5000"), // scripted has a default timeout of 5000
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.scripted", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.scripted", "labels.environment", "production"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.scripted", "settings.0.scripted.0.script", string(scriptFileContent)),
				),
			},
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.scripted",
			},
		},
	})
}

func TestAccResourceCheck_grpc(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("grpc")
	nameReplaceMap := map[string]string{
		`"gRPC Defaults"`: strconv.Quote(jobName),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/grpc_basic.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.grpc", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.grpc", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "target", "host.docker.internal:50051"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.grpc", "probes.0"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "labels.foo", "bar"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "settings.0.grpc.0.ip_version", "V4"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "settings.0.grpc.0.tls", "false"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/grpc_complex.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.grpc", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.grpc", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "target", "host.docker.internal:50051"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.grpc", "probes.0"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.grpc", "probes.1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "labels.foo", "baz"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "settings.0.grpc.0.service", "health-check-test"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "settings.0.grpc.0.ip_version", "V6"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "settings.0.grpc.0.tls", "true"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.grpc", "settings.0.grpc.0.tls_config.0.server_name", "grafana.com"),
					resource.TestMatchResourceAttr("grafana_synthetic_monitoring_check.grpc", "settings.0.grpc.0.tls_config.0.ca_cert", regexp.MustCompile((`^-{5}BEGIN CERTIFICATE`))),
				),
			},
			{
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "grafana_synthetic_monitoring_check.grpc",
			},
		},
	})
}

// Test that a check is recreated if deleted outside the Terraform process
func TestAccResourceCheck_recreate(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Inject random job names to avoid conflicts with other tests
	jobName := acctest.RandomWithPrefix("http")
	nameReplaceMap := map[string]string{
		`"HTTP Defaults"`: strconv.Quote(jobName),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf", nameReplaceMap),
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["grafana_synthetic_monitoring_check.http"]
					id, _ := strconv.ParseInt(rs.Primary.ID, 10, 64)
					return testutils.Provider.Meta().(*common.Client).SMAPI.DeleteCheck(context.Background(), id)
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf", nameReplaceMap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "tenant_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "job", jobName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check.http", "target", "https://grafana.com"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "probes.0"),
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
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
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
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
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

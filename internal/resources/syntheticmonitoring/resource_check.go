package syntheticmonitoring

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
)

const (
	checkDefaultTimeout          = 3000
	checkMultiHTTPDefaultTimeout = 5000
)

var (

	// Set variables for schemas used in multiple fields and/or used to transform
	// API client types back to schemas.

	// All check types set IP version.
	syntheticMonitoringCheckIPVersion = &schema.Schema{
		Description: "Options are `V4`, `V6`, `Any`. Specifies whether the corresponding check will be performed using IPv4 or IPv6. " +
			"The `Any` value indicates that IPv6 should be used, falling back to IPv4 if that's not available.",
		Type:     schema.TypeString,
		Optional: true,
		Default:  "V4",
	}

	// HTTP, TCP and gRPC Health checks can set TLS config.
	syntheticMonitoringCheckTLSConfig = &schema.Schema{
		Description: "TLS config.",
		Type:        schema.TypeSet,
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"insecure_skip_verify": {
					Description: "Disable target certificate validation.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"ca_cert": {
					Description: "CA certificate in PEM format.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"client_cert": {
					Description: "Client certificate in PEM format.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"client_key": {
					Description: "Client key in PEM format.",
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
				},
				"server_name": {
					Description: "Used to verify the hostname for the targets.",
					Type:        schema.TypeString,
					Optional:    true,
				},
			},
		},
	}

	syntheticMonitoringCheckSettings = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"dns": {
				Description: "Settings for DNS check. The target must be a valid hostname (or IP address for `PTR` records).",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsDNS,
			},
			"http": {
				Description: "Settings for HTTP check. The target must be a URL (http or https).",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsHTTP,
			},
			"ping": {
				Description: "Settings for ping (ICMP) check. The target must be a valid hostname or IP address.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsPing,
			},
			"tcp": {
				Description: "Settings for TCP check. The target must be of the form `<host>:<port>`, where the host portion must be a valid hostname or IP address.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsTCP,
			},
			"traceroute": {
				Description: "Settings for traceroute check. The target must be a valid hostname or IP address",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsTraceroute,
			},
			"multihttp": {
				Description: "Settings for MultiHTTP check. The target must be a URL (http or https)",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsMultiHTTP,
			},
			"scripted": {
				Description: "Settings for scripted check. See https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/create-checks/checks/k6/.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsScripted,
			},
			"grpc": {
				Description: "Settings for gRPC Health check. The target must be of the form `<host>:<port>`, where the host portion must be a valid hostname or IP address.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsGRPC,
			},
			"browser": {
				Description: "Settings for browser check. See https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/create-checks/checks/k6-browser/.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsBrowser,
			},
		},
	}

	syntheticMonitoringCheckSettingsScripted = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"script": {
				Type:     schema.TypeString,
				Optional: false,
				Required: true,
			},
		},
	}

	syntheticMonitoringCheckSettingsBrowser = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"script": {
				Type:     schema.TypeString,
				Optional: false,
				Required: true,
			},
		},
	}

	syntheticMonitoringCheckSettingsDNS = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip_version": syntheticMonitoringCheckIPVersion,
			"source_ip_address": {
				Description: "Source IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"server": {
				Description: "DNS server address to target.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "8.8.8.8",
			},
			"port": {
				Description: "Port to target.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     53,
			},
			"record_type": {
				Description: "One of `ANY`, `A`, `AAAA`, `CNAME`, `MX`, `NS`, `PTR`, `SOA`, `SRV`, `TXT`.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "A",
			},
			"protocol": {
				Description: "`TCP` or `UDP`.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "UDP",
			},
			"valid_r_codes": {
				Description: "List of valid response codes. Options include `NOERROR`, `BADALG`, `BADMODE`, `BADKEY`, `BADCOOKIE`, `BADNAME`, `BADSIG`, `BADTIME`, `BADTRUNC`, " +
					"`BADVERS`, `FORMERR`, `NOTIMP`, `NOTAUTH`, `NOTZONE`, `NXDOMAIN`, `NXRRSET`, `REFUSED`, `SERVFAIL`, `YXDOMAIN`, `YXRRSET`.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"validate_answer_rrs": {
				Description: "Validate response answer.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsDNSValidate,
			},
			"validate_authority_rrs": {
				Description: "Validate response authority.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsDNSValidate,
			},
			"validate_additional_rrs": {
				Description: "Validate additional matches.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        syntheticMonitoringCheckSettingsDNSValidate,
			},
		},
	}

	syntheticMonitoringCheckSettingsDNSValidate = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"fail_if_matches_regexp": {
				Description: "Fail if value matches regex.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"fail_if_not_matches_regexp": {
				Description: "Fail if value does not match regex.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}

	syntheticMonitoringCheckSettingsHTTP = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip_version": syntheticMonitoringCheckIPVersion,
			"tls_config": syntheticMonitoringCheckTLSConfig,
			"method": {
				Description: "Request method. One of `GET`, `CONNECT`, `DELETE`, `HEAD`, `OPTIONS`, `POST`, `PUT`, `TRACE`",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "GET",
			},
			"headers": {
				Description: "The HTTP headers set for the probe.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"body": {
				Description: "The body of the HTTP request used in probe.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"no_follow_redirects": {
				Description: "Do not follow redirects.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"basic_auth": {
				Description: "Basic auth settings.",
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettingsHTTPBasicAuth,
			},
			"bearer_token": {
				Description: "Token for use with bearer authorization header.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			"proxy_url": {
				Description: "Proxy URL.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"proxy_connect_headers": {
				Description: "The HTTP headers sent to the proxy URL",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"fail_if_ssl": {
				Description: "Fail if SSL is present.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"fail_if_not_ssl": {
				Description: "Fail if SSL is not present.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"valid_status_codes": {
				Description: "Accepted status codes. If unset, defaults to 2xx.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"valid_http_versions": {
				Description: "List of valid HTTP versions. Options include `HTTP/1.0`, `HTTP/1.1`, `HTTP/2.0`",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"fail_if_body_matches_regexp": {
				Description: "List of regexes. If any match the response body, the check will fail.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"fail_if_body_not_matches_regexp": {
				Description: "List of regexes. If any do not match the response body, the check will fail.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"fail_if_header_matches_regexp": {
				Description: "Check fails if headers match.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        syntheticMonitoringCheckSettingsHTTPHeaderMatch,
			},
			"fail_if_header_not_matches_regexp": {
				Description: "Check fails if headers do not match.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        syntheticMonitoringCheckSettingsHTTPHeaderMatch,
			},
			"compression": {
				Description:  "Check fails if the response body is not compressed using this compression algorithm. One of `none`, `identity`, `br`, `gzip`, `deflate`.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(slices.Collect(maps.Keys(sm.CompressionAlgorithm_value)), false),
			},
			"cache_busting_query_param_name": {
				Description: "The name of the query parameter used to prevent the server from using a cached response. Each probe will assign a random value to this parameter each time a request is made.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}

	syntheticMonitoringCheckSettingsHTTPBasicAuth = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"username": {
				Description: "Basic auth username.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"password": {
				Description: "Basic auth password.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
		},
	}

	syntheticMonitoringCheckSettingsHTTPHeaderMatch = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"header": {
				Description: "Header name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regexp": {
				Description: "Regex that header value should match.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"allow_missing": {
				Description: "Allow header to be missing from responses.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}

	syntheticMonitoringCheckSettingsPing = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip_version": syntheticMonitoringCheckIPVersion,
			"source_ip_address": {
				Description: "Source IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"payload_size": {
				Description: "Payload size.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
			},
			"dont_fragment": {
				Description: "Set the DF-bit in the IP-header. Only works with ipV4.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}

	syntheticMonitoringCheckSettingsTCP = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip_version": syntheticMonitoringCheckIPVersion,
			"tls_config": syntheticMonitoringCheckTLSConfig,
			"source_ip_address": {
				Description: "Source IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"tls": {
				Description: "Whether or not TLS is used when the connection is initiated.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"query_response": {
				Description: "The query sent in the TCP probe and the expected associated response.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        syntheticMonitoringCheckSettingsTCPQueryResponse,
			},
		},
	}

	syntheticMonitoringCheckSettingsTCPQueryResponse = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"send": {
				Description: "Data to send.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"expect": {
				Description: "Response to expect.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"start_tls": {
				Description: "Upgrade TCP connection to TLS.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}

	syntheticMonitoringCheckSettingsTraceroute = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"max_hops": {
				Description: "Maximum TTL for the trace",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     64,
			},
			"max_unknown_hops": {
				Description: "Maximum number of hosts to travers that give no response",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     15,
			},
			"ptr_lookup": {
				Description: "Reverse lookup hostnames from IP addresses",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
		},
	}

	syntheticMonitoringCheckSettingsMultiHTTP = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"entries": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     syntheticMonitoringCheckSettingsMultiHTTPEntry,
			},
		},
	}

	syntheticMonitoringCheckSettingsMultiHTTPEntry = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"request":    syntheticMonitoringMultiHTTPRequest,
			"assertions": syntheticMonitoringMultiHTTPAssertion,
			"variables":  syntheticMonitoringMultiHTTPVariable,
		},
	}

	syntheticMonitoringMultiHTTPRequestHeader = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the header to send",
				Type:        schema.TypeString,
				Required:    true,
			},
			"value": {
				Description: "Value of the header to send",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}

	syntheticMonitoringMultiHTTPRequestQueryField = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the query field to send",
				Type:        schema.TypeString,
				Required:    true,
			},
			"value": {
				Description: "Value of the query field to send",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}

	syntheticMonitoringMultiHTTPRequestBody = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"content_type": {
				Description: "The content type of the body",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"content_encoding": {
				Description: "The content encoding of the body",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"payload": {
				Description: "The body payload",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}

	syntheticMonitoringMultiHTTPRequest = &schema.Schema{
		Description: "An individual MultiHTTP request",
		Type:        schema.TypeSet,
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"method": {
					Description: "The HTTP method to use",
					Type:        schema.TypeString,
					Required:    true,
				},
				"url": {
					Description: "The URL for the request",
					Type:        schema.TypeString,
					Required:    true,
				},
				"headers": {
					Description: "The headers to send with the request",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem:        syntheticMonitoringMultiHTTPRequestHeader,
				},
				"query_fields": {
					Description: "Query fields to send with the request",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem:        syntheticMonitoringMultiHTTPRequestQueryField,
				},
				"body": {
					Description: "The body of the HTTP request used in probe.",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem:        syntheticMonitoringMultiHTTPRequestBody,
				},
			},
		},
	}

	syntheticMonitoringMultiHTTPAssertion = &schema.Schema{
		Description: "Assertions to make on the request response",
		Type:        schema.TypeList,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Description: "The type of assertion to make: TEXT, JSON_PATH_VALUE, JSON_PATH_ASSERTION, REGEX_ASSERTION",
					Type:        schema.TypeString,
					Required:    true,
				},
				"subject": {
					Description: "The subject of the assertion: RESPONSE_HEADERS, HTTP_STATUS_CODE, RESPONSE_BODY",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"condition": {
					Description: "The condition of the assertion: NOT_CONTAINS, EQUALS, STARTS_WITH, ENDS_WITH, TYPE_OF, CONTAINS",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"expression": {
					Description: "The expression of the assertion. Should start with $.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"value": {
					Description: "The value of the assertion",
					Type:        schema.TypeString,
					Optional:    true,
				},
			},
		},
	}

	syntheticMonitoringMultiHTTPVariable = &schema.Schema{
		Description: "Variables to extract from the request response",
		Type:        schema.TypeList,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Description: "The method of finding the variable value to extract. JSON_PATH, REGEX, CSS_SELECTOR",
					Type:        schema.TypeString,
					Required:    true,
				},
				"name": {
					Description: "The name of the variable to extract",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"expression": {
					Description: "The expression to when finding the variable. Should start with $. Only use when type is JSON_PATH or REGEX",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"attribute": {
					Description: "The attribute to use when finding the variable value. Only used when type is CSS_SELECTOR",
					Type:        schema.TypeString,
					Optional:    true,
				},
			},
		},
	}

	syntheticMonitoringCheckSettingsGRPC = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip_version": syntheticMonitoringCheckIPVersion,
			"tls": {
				Description: "Whether or not TLS is used when the connection is initiated.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"tls_config": syntheticMonitoringCheckTLSConfig,
			"service": {
				Description: "gRPC service.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}

	resourceCheckID = common.NewResourceID(common.IntIDField("id"))
)

func resourceCheck() *common.Resource {
	schema := &schema.Resource{
		Description: `
Synthetic Monitoring checks are tests that run on selected probes at defined
intervals and report metrics and logs back to your Grafana Cloud account. The
target for checks can be a domain name, a server, or a website, depending on
what information you would like to gather about your endpoint. You can define
multiple checks for a single endpoint to check different capabilities.

* [Official documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/create-checks/checks/)
`,

		CreateContext: withClient[schema.CreateContextFunc](resourceCheckCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceCheckRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceCheckUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceCheckDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: resourceCheckCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of the check.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"tenant_id": {
				Description: "The tenant ID of the check.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"job": {
				Description: "Name used for job label.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"target": {
				Description: "Hostname to ping.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"frequency": {
				Description: "How often the check runs in milliseconds (the value is not truly a \"frequency\" but a \"period\"). " +
					"The minimum acceptable value is 1 second (1000 ms), and the maximum is 1 hour (3600000 ms).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60000,
				ValidateFunc: validation.IntBetween(1000, 3600000),
			},
			"timeout": {
				Description: "Specifies the maximum running time for the check in milliseconds. " +
					"The minimum acceptable value is 1 second (1000 ms), and the maximum 10 seconds (10000 ms).",
				Type:     schema.TypeInt,
				Optional: true,
				Default:  checkDefaultTimeout,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diff if it's a multihttp check with a timeout of 5000 (default timeout for those)
					// and it's being changed to 3000 (default timeout set here).
					doSuppress := d.Get("settings.0.multihttp.0") != nil || d.Get("settings.0.scripted") != nil || d.Get("settings.0.browser") != nil
					if doSuppress &&
						old == strconv.Itoa(checkMultiHTTPDefaultTimeout) &&
						new == strconv.Itoa(checkDefaultTimeout) {
						return true
					}

					return old == new
				},
			},
			"enabled": {
				Description: "Whether to enable the check.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"alert_sensitivity": {
				Description: "Can be set to `none`, `low`, `medium`, or `high` to correspond to the check [alert levels](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/configure-alerts/synthetic-monitoring-alerting/).",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "none",
			},
			"basic_metrics_only": {
				Description: "Metrics are reduced by default. Set this to `false` if you'd like to publish all metrics. " +
					"We maintain a [full list of metrics](https://github.com/grafana/synthetic-monitoring-agent/tree/main/internal/scraper/testdata) collected for each.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"probes": {
				Description: "List of probe location IDs where this target will be checked from.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"labels": {
				Description: "Custom labels to be included with collected metrics and logs. " +
					"The maximum number of labels that can be specified per check is 5. " +
					"These are applied, along with the probe-specific labels, to the outgoing metrics. " +
					"The names and values of the labels cannot be empty, and the maximum length is 32 bytes.",
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"settings": {
				Description: "Check settings. Should contain exactly one nested block.",
				Type:        schema.TypeSet,
				Required:    true,
				MaxItems:    1,
				Elem:        syntheticMonitoringCheckSettings,
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategorySyntheticMonitoring,
		"grafana_synthetic_monitoring_check",
		resourceCheckID,
		schema,
	).
		WithLister(listChecks).
		WithPreferredResourceNameField("job")
}

func listChecks(ctx context.Context, client *client.Client, data any) ([]string, error) {
	smClient := client.SMAPI
	if smClient == nil {
		return nil, fmt.Errorf("client not configured for SM API")
	}

	checkList, err := smClient.ListChecks(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, check := range checkList {
		ids = append(ids, strconv.FormatInt(check.Id, 10))
	}
	return ids, nil
}

func resourceCheckCreate(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	chk, err := makeCheck(d)
	if err != nil {
		return diag.FromErr(err)
	}
	res, err := c.AddCheck(ctx, *chk)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(res.Id, 10))
	d.Set("tenant_id", res.TenantId)
	return resourceCheckRead(ctx, d, c)
}

//nolint:gocyclo
func resourceCheckRead(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	id, err := resourceCheckID.Single(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	chk, err := c.GetCheck(ctx, id.(int64))
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return common.WarnMissing("check", d)
		}
		return diag.FromErr(err)
	}

	d.Set("tenant_id", chk.TenantId)
	d.Set("job", chk.Job)
	d.Set("target", chk.Target)
	d.Set("frequency", chk.Frequency)
	d.Set("timeout", chk.Timeout)
	d.Set("enabled", chk.Enabled)
	d.Set("alert_sensitivity", chk.AlertSensitivity)
	d.Set("basic_metrics_only", chk.BasicMetricsOnly)
	d.Set("probes", chk.Probes)

	if len(chk.Labels) > 0 {
		// Convert []sm.Label into a map before set.
		labels := make(map[string]string, len(chk.Labels))
		for _, l := range chk.Labels {
			labels[l.Name] = l.Value
		}
		d.Set("labels", labels)
	}

	// Convert sm.Settings...

	settings := schema.NewSet(
		schema.HashResource(syntheticMonitoringCheckSettings),
		[]interface{}{},
	)

	tlsConfig := func(t *sm.TLSConfig) *schema.Set {
		if t == nil {
			return &schema.Set{}
		}
		return schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckTLSConfig.Elem.(*schema.Resource)),
			[]interface{}{
				map[string]interface{}{
					"insecure_skip_verify": t.InsecureSkipVerify,
					"ca_cert":              string(t.CACert),
					"client_cert":          string(t.ClientCert),
					"client_key":           string(t.ClientKey),
					"server_name":          t.ServerName,
				},
			})
	}

	switch {
	case chk.Settings.Dns != nil:
		dns := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsDNS),
			[]interface{}{},
		)
		dnsValidator := func(v *sm.DNSRRValidator) *schema.Set {
			if v == nil {
				return &schema.Set{}
			}
			return schema.NewSet(
				schema.HashResource(syntheticMonitoringCheckSettingsDNSValidate),
				[]interface{}{
					map[string]interface{}{
						"fail_if_matches_regexp":     common.StringSliceToSet(v.FailIfMatchesRegexp),
						"fail_if_not_matches_regexp": common.StringSliceToSet(v.FailIfNotMatchesRegexp),
					},
				},
			)
		}
		dns.Add(map[string]interface{}{
			"ip_version":              chk.Settings.Dns.IpVersion.String(),
			"source_ip_address":       chk.Settings.Dns.SourceIpAddress,
			"server":                  chk.Settings.Dns.Server,
			"port":                    int(chk.Settings.Dns.Port),
			"record_type":             chk.Settings.Dns.RecordType.String(),
			"protocol":                chk.Settings.Dns.Protocol.String(),
			"valid_r_codes":           common.StringSliceToSet(chk.Settings.Dns.ValidRCodes),
			"validate_answer_rrs":     dnsValidator(chk.Settings.Dns.ValidateAnswer),
			"validate_authority_rrs":  dnsValidator(chk.Settings.Dns.ValidateAuthority),
			"validate_additional_rrs": dnsValidator(chk.Settings.Dns.ValidateAdditional),
		})
		settings.Add(map[string]interface{}{
			"dns": dns,
		})
	case chk.Settings.Http != nil:
		http := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsPing),
			[]interface{}{},
		)
		basicAuth := schema.Set{}
		if chk.Settings.Http.BasicAuth != nil {
			basicAuth = *schema.NewSet(schema.HashResource(syntheticMonitoringCheckSettingsHTTPBasicAuth),
				[]interface{}{
					map[string]interface{}{
						"username": chk.Settings.Http.BasicAuth.Username,
						"password": chk.Settings.Http.BasicAuth.Password,
					},
				},
			)
		}
		// The default compression "none" is the same as omitting the value.
		// Since this value is usually not explicitly set, omit when set to "none"
		var compression string
		if chk.Settings.Http.Compression != sm.CompressionAlgorithm_none {
			compression = chk.Settings.Http.Compression.String()
		}
		headerMatch := func(hms []sm.HeaderMatch) *schema.Set {
			hmSet := schema.NewSet(
				schema.HashResource(syntheticMonitoringCheckSettingsTCPQueryResponse),
				[]interface{}{},
			)
			for _, hm := range hms {
				hmSet.Add(map[string]interface{}{
					"header":        hm.Header,
					"regexp":        hm.Regexp,
					"allow_missing": hm.AllowMissing,
				})
			}
			return hmSet
		}
		http.Add(map[string]interface{}{
			"ip_version":                        chk.Settings.Http.IpVersion.String(),
			"tls_config":                        tlsConfig(chk.Settings.Http.TlsConfig),
			"method":                            chk.Settings.Http.Method.String(),
			"headers":                           common.StringSliceToSet(chk.Settings.Http.Headers),
			"body":                              chk.Settings.Http.Body,
			"no_follow_redirects":               chk.Settings.Http.NoFollowRedirects,
			"basic_auth":                        &basicAuth,
			"bearer_token":                      chk.Settings.Http.BearerToken,
			"proxy_url":                         chk.Settings.Http.ProxyURL,
			"proxy_connect_headers":             common.StringSliceToSet(chk.Settings.Http.ProxyConnectHeaders),
			"fail_if_ssl":                       chk.Settings.Http.FailIfSSL,
			"fail_if_not_ssl":                   chk.Settings.Http.FailIfNotSSL,
			"valid_status_codes":                common.Int32SliceToSet(chk.Settings.Http.ValidStatusCodes),
			"valid_http_versions":               common.StringSliceToSet(chk.Settings.Http.ValidHTTPVersions),
			"fail_if_body_matches_regexp":       common.StringSliceToSet(chk.Settings.Http.FailIfBodyMatchesRegexp),
			"fail_if_body_not_matches_regexp":   common.StringSliceToSet(chk.Settings.Http.FailIfBodyNotMatchesRegexp),
			"fail_if_header_matches_regexp":     headerMatch(chk.Settings.Http.FailIfHeaderMatchesRegexp),
			"fail_if_header_not_matches_regexp": headerMatch(chk.Settings.Http.FailIfHeaderNotMatchesRegexp),
			"compression":                       compression,
			"cache_busting_query_param_name":    chk.Settings.Http.CacheBustingQueryParamName,
		})

		settings.Add(map[string]interface{}{
			"http": http,
		})
	case chk.Settings.Ping != nil:
		ping := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsPing),
			[]interface{}{},
		)
		ping.Add(map[string]interface{}{
			"ip_version":        chk.Settings.Ping.IpVersion.String(),
			"source_ip_address": chk.Settings.Ping.SourceIpAddress,
			"payload_size":      int(chk.Settings.Ping.PayloadSize),
			"dont_fragment":     chk.Settings.Ping.DontFragment,
		})
		settings.Add(map[string]interface{}{
			"ping": ping,
		})
	case chk.Settings.Tcp != nil:
		tcp := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsTCP),
			[]interface{}{},
		)
		queryResponse := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsTCPQueryResponse),
			[]interface{}{},
		)
		for _, qr := range chk.Settings.Tcp.QueryResponse {
			queryResponse.Add(map[string]interface{}{
				"send":      string(qr.Send),
				"expect":    string(qr.Expect),
				"start_tls": qr.StartTLS,
			})
		}
		tcp.Add(map[string]interface{}{
			"ip_version":        chk.Settings.Tcp.IpVersion.String(),
			"tls_config":        tlsConfig(chk.Settings.Tcp.TlsConfig),
			"source_ip_address": chk.Settings.Tcp.SourceIpAddress,
			"tls":               chk.Settings.Tcp.Tls,
			"query_response":    queryResponse,
		})
		settings.Add(map[string]interface{}{
			"tcp": tcp,
		})
	case chk.Settings.Traceroute != nil:
		traceroute := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsTraceroute),
			[]interface{}{},
		)

		traceroute.Add(map[string]interface{}{
			"max_hops":         int(chk.Settings.Traceroute.MaxHops),
			"max_unknown_hops": int(chk.Settings.Traceroute.MaxUnknownHops),
			"ptr_lookup":       chk.Settings.Traceroute.PtrLookup,
		})
		settings.Add(map[string]interface{}{
			"traceroute": traceroute,
		})
	case chk.Settings.Multihttp != nil:
		entries := []any{}
		for _, e := range chk.Settings.Multihttp.Entries {
			requestSet := schema.NewSet(
				schema.HashResource(syntheticMonitoringMultiHTTPRequest.Elem.(*schema.Resource)),
				[]any{},
			)
			request := map[string]any{
				"method": e.Request.Method.String(),
				"url":    e.Request.Url,
			}
			if len(e.Request.Headers) > 0 {
				headerSet := schema.NewSet(
					schema.HashResource(syntheticMonitoringMultiHTTPRequestHeader),
					[]any{},
				)
				for _, h := range e.Request.Headers {
					headerSet.Add(map[string]any{
						"name":  h.Name,
						"value": h.Value,
					})
				}
				request["headers"] = headerSet
			}
			if len(e.Request.QueryFields) > 0 {
				querySet := schema.NewSet(
					schema.HashResource(syntheticMonitoringMultiHTTPRequestQueryField),
					[]any{},
				)
				for _, q := range e.Request.QueryFields {
					querySet.Add(map[string]any{
						"name":  q.Name,
						"value": q.Value,
					})
				}
				request["query_fields"] = querySet
			}
			if e.Request.Body != nil {
				bodySet := schema.NewSet(
					schema.HashResource(syntheticMonitoringMultiHTTPRequestBody),
					[]any{},
				)
				bodySet.Add(map[string]any{
					"content_type":     e.Request.Body.ContentType,
					"content_encoding": e.Request.Body.ContentEncoding,
					"payload":          string(e.Request.Body.Payload),
				})
				request["body"] = bodySet
			}
			requestSet.Add(request)
			checks := []any{}
			for _, assertion := range e.Assertions {
				check := map[string]any{
					"type":       assertion.Type.String(),
					"expression": assertion.Expression,
					"value":      assertion.Value,
				}
				if assertion.Subject != sm.MultiHttpEntryAssertionSubjectVariant_DEFAULT_SUBJECT {
					check["subject"] = assertion.Subject.String()
				}
				if assertion.Condition != sm.MultiHttpEntryAssertionConditionVariant_DEFAULT_CONDITION {
					check["condition"] = assertion.Condition.String()
				}
				checks = append(checks, check)
			}
			variables := []any{}
			for _, v := range e.Variables {
				variables = append(variables, map[string]any{
					"type":       v.Type.String(),
					"name":       v.Name,
					"expression": v.Expression,
					"attribute":  v.Attribute,
				})
			}
			entries = append(entries, map[string]any{
				"request":    requestSet,
				"assertions": checks,
				"variables":  variables,
			})
		}

		multiHTTP := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsMultiHTTP),
			[]any{},
		)
		multiHTTP.Add(map[string]any{
			"entries": entries,
		})
		settings.Add(map[string]any{
			"multihttp": multiHTTP,
		})
	case chk.Settings.Scripted != nil:
		scripted := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsScripted),
			[]any{},
		)
		scripted.Add(map[string]any{
			"script": string(chk.Settings.Scripted.Script),
		})
		settings.Add(map[string]any{
			"scripted": scripted,
		})
	case chk.Settings.Grpc != nil:
		grpc := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsGRPC),
			[]interface{}{},
		)
		grpc.Add(map[string]interface{}{
			"ip_version": chk.Settings.Grpc.IpVersion.String(),
			"tls_config": tlsConfig(chk.Settings.Grpc.TlsConfig),
			"tls":        chk.Settings.Grpc.Tls,
			"service":    chk.Settings.Grpc.Service,
		})
		settings.Add(map[string]interface{}{
			"grpc": grpc,
		})
	case chk.Settings.Browser != nil:
		browser := schema.NewSet(
			schema.HashResource(syntheticMonitoringCheckSettingsBrowser),
			[]any{},
		)
		browser.Add(map[string]any{
			"script": string(chk.Settings.Browser.Script),
		})
		settings.Add(map[string]any{
			"browser": browser,
		})
	}

	d.Set("settings", settings)

	return nil
}

func resourceCheckUpdate(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	chk, err := makeCheck(d)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.UpdateCheck(ctx, *chk)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceCheckRead(ctx, d, c)
}

func resourceCheckDelete(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	id, err := resourceCheckID.Single(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	err = c.DeleteCheck(ctx, id.(int64))
	return diag.FromErr(err)
}

// makeCheck populates an instance of sm.Check. We need this for create and
// update calls with the SM API client.
func makeCheck(d *schema.ResourceData) (*sm.Check, error) {
	var id int64
	if d.Id() != "" {
		id, _ = strconv.ParseInt(d.Id(), 10, 64)
	}

	var probes []int64
	for _, p := range d.Get("probes").(*schema.Set).List() {
		probes = append(probes, int64(p.(int)))
	}

	var labels []sm.Label
	for name, value := range d.Get("labels").(map[string]interface{}) {
		labels = append(labels, sm.Label{
			Name:  name,
			Value: value.(string),
		})
	}

	settings, err := makeCheckSettings(d.Get("settings").(*schema.Set).List()[0].(map[string]interface{}))
	if err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	timeout := int64(d.Get("timeout").(int))
	if timeout == checkDefaultTimeout && (settings.Multihttp != nil || settings.Scripted != nil || settings.Browser != nil) {
		timeout = checkMultiHTTPDefaultTimeout
	}

	return &sm.Check{
		Id:               id,
		TenantId:         int64(d.Get("tenant_id").(int)),
		Job:              d.Get("job").(string),
		Target:           d.Get("target").(string),
		Frequency:        int64(d.Get("frequency").(int)),
		Timeout:          timeout,
		Enabled:          d.Get("enabled").(bool),
		AlertSensitivity: d.Get("alert_sensitivity").(string),
		BasicMetricsOnly: d.Get("basic_metrics_only").(bool),
		Probes:           probes,
		Labels:           labels,
		Settings:         settings,
	}, nil
}

func makeMultiHTTPSettings(settings map[string]any, cs *sm.CheckSettings) error {
	var out sm.MultiHttpSettings

	entries := settings["entries"]
	if entries == nil {
		return errors.New("entries not set")
	}

	for _, entry := range entries.([]any) {
		entry, err := makeMultiHTTPEntry(entry.(map[string]any))
		if err != nil {
			return err
		}
		out.Entries = append(out.Entries, entry)
	}

	// Everything is OK, modify the structure.
	cs.Multihttp = &out

	return nil
}

func makeMultiHTTPEntry(entry map[string]any) (*sm.MultiHttpEntry, error) {
	var request map[string]any

	if r := entry["request"]; r == nil {
		return nil, errors.New("request not set")
	} else if requests := r.(*schema.Set).List(); len(requests) == 0 {
		return nil, errors.New("request is empty")
	} else {
		request = requests[0].(map[string]any)
	}

	method := sm.HttpMethod(sm.HttpMethod_value[request["method"].(string)])

	e := sm.MultiHttpEntry{
		Request: &sm.MultiHttpEntryRequest{
			Method: method,
			Url:    request["url"].(string),
		},
	}

	if headers := request["headers"]; headers != nil {
		for _, header := range headers.(*schema.Set).List() {
			header := header.(map[string]any)
			e.Request.Headers = append(e.Request.Headers, &sm.HttpHeader{
				Name:  header["name"].(string),
				Value: header["value"].(string),
			})
		}
	}

	if queryFields := request["query_fields"]; queryFields != nil {
		for _, queryField := range queryFields.(*schema.Set).List() {
			queryField := queryField.(map[string]any)
			e.Request.QueryFields = append(e.Request.QueryFields, &sm.QueryField{
				Name:  queryField["name"].(string),
				Value: queryField["value"].(string),
			})
		}
	}

	if reqBody := request["body"]; reqBody != nil {
		reqBody := reqBody.(*schema.Set).List()
		if len(reqBody) > 0 {
			r := reqBody[0].(map[string]any)
			e.Request.Body = &sm.HttpRequestBody{
				ContentType:     r["content_type"].(string),
				ContentEncoding: r["content_encoding"].(string),
				Payload:         []byte(r["payload"].(string)),
			}
		}
	}

	if variables := entry["variables"]; variables != nil {
		for _, variable := range variables.([]any) {
			c := variable.(map[string]any)

			typeValue, ok := c["type"].(string)
			if !ok {
				return nil, errors.New("variable type not set")
			}

			variableType, err := sm.MultiHttpEntryVariableTypeString(typeValue)
			if err != nil {
				return nil, fmt.Errorf("invalid variable type %q", typeValue)
			}

			e.Variables = append(e.Variables, &sm.MultiHttpEntryVariable{
				Name:       c["name"].(string),
				Type:       variableType,
				Expression: c["expression"].(string),
				Attribute:  c["attribute"].(string),
			})
		}
	}

	entries := entry["assertions"]
	if entries == nil {
		// no entries, we are done
		return &e, nil
	}

	for _, settings := range entries.([]any) {
		assertion, err := makeMultiHTTPAssertion(settings.(map[string]any))
		if err != nil {
			return nil, err
		}

		e.Assertions = append(e.Assertions, assertion)
	}

	return &e, nil
}

func makeMultiHTTPAssertion(settings map[string]any) (*sm.MultiHttpEntryAssertion, error) {
	var assertion sm.MultiHttpEntryAssertion

	if settings["type"] == nil {
		return nil, errors.New("assertion type not set")
	}

	v, ok := settings["type"].(string)
	if !ok {
		return nil, fmt.Errorf("unsupported assertion type value %v", v)
	}

	if len(v) > 0 {
		value, err := sm.MultiHttpEntryAssertionTypeString(v)
		if err != nil {
			return nil, fmt.Errorf("invalid assertion type %q", v)
		}
		assertion.Type = value
	}

	if !assertion.Type.IsAMultiHttpEntryAssertionType() {
		return nil, fmt.Errorf("invalid assertion type %s", v)
	}

	if settings["subject"] != nil {
		v, ok := settings["subject"].(string)
		if !ok {
			return nil, fmt.Errorf("unsupported assertion type value %v", v)
		}

		if len(v) > 0 {
			value, err := sm.MultiHttpEntryAssertionSubjectVariantString(v)
			if err != nil {
				return nil, fmt.Errorf("invalid assertion subject %q", v)
			}
			assertion.Subject = value
		}

		if !assertion.Subject.IsAMultiHttpEntryAssertionSubjectVariant() {
			return nil, fmt.Errorf("invalid assertion subject %s", v)
		}
	}

	if settings["condition"] != nil {
		v, ok := settings["condition"].(string)
		if !ok {
			return nil, fmt.Errorf("unsupported assertion condition value %v", v)
		}

		if len(v) > 0 {
			value, err := sm.MultiHttpEntryAssertionConditionVariantString(v)
			if err != nil {
				return nil, fmt.Errorf("invalid assertion condition %q", v)
			}
			assertion.Condition = value
		}

		if !assertion.Condition.IsAMultiHttpEntryAssertionConditionVariant() {
			return nil, fmt.Errorf("invalid assertion condition %s", v)
		}
	}

	if settings["expression"] != nil {
		assertion.Expression = settings["expression"].(string)
	}

	if settings["value"] != nil {
		assertion.Value = settings["value"].(string)
	}

	return &assertion, nil
}

// makeCheckSettings populates an instance of sm.CheckSettings. This is called
// by makeCheck. It's isolated from makeCheck to hopefully make it all more
// human readable.
func makeCheckSettings(settings map[string]interface{}) (sm.CheckSettings, error) {
	cs := sm.CheckSettings{}

	tlsConfig := func(t *schema.Set) *sm.TLSConfig {
		tc := t.List()[0].(map[string]interface{})
		return &sm.TLSConfig{
			InsecureSkipVerify: tc["insecure_skip_verify"].(bool),
			CACert:             []byte(tc["ca_cert"].(string)),
			ClientCert:         []byte(tc["client_cert"].(string)),
			ClientKey:          []byte(tc["client_key"].(string)),
			ServerName:         tc["server_name"].(string),
		}
	}

	dns := settings["dns"].(*schema.Set).List()
	if len(dns) > 0 {
		d := dns[0].(map[string]interface{})
		cs.Dns = &sm.DnsSettings{
			IpVersion:       sm.IpVersion(sm.IpVersion_value[d["ip_version"].(string)]),
			SourceIpAddress: d["source_ip_address"].(string),
			Server:          d["server"].(string),
			Port:            int32(d["port"].(int)), //nolint:gosec
			RecordType:      sm.DnsRecordType(sm.DnsRecordType_value[d["record_type"].(string)]),
			Protocol:        sm.DnsProtocol(sm.DnsProtocol_value[d["protocol"].(string)]),
			ValidRCodes:     common.SetToStringSlice(d["valid_r_codes"].(*schema.Set)),
		}
		dnsValidator := func(validation string) *sm.DNSRRValidator {
			val := sm.DNSRRValidator{}
			for _, v := range d[validation].(*schema.Set).List() {
				val.FailIfMatchesRegexp = common.SetToStringSlice(v.(map[string]interface{})["fail_if_matches_regexp"].(*schema.Set))
				val.FailIfNotMatchesRegexp = common.SetToStringSlice(v.(map[string]interface{})["fail_if_not_matches_regexp"].(*schema.Set))
			}
			return &val
		}
		if d["validate_answer_rrs"].(*schema.Set).Len() > 0 {
			cs.Dns.ValidateAnswer = dnsValidator("validate_answer_rrs")
		}
		if d["validate_authority_rrs"].(*schema.Set).Len() > 0 {
			cs.Dns.ValidateAuthority = dnsValidator("validate_authority_rrs")
		}
		if d["validate_additional_rrs"].(*schema.Set).Len() > 0 {
			cs.Dns.ValidateAdditional = dnsValidator("validate_additional_rrs")
		}
	}

	http := settings["http"].(*schema.Set).List()
	if len(http) > 0 {
		h := http[0].(map[string]interface{})
		cs.Http = &sm.HttpSettings{
			IpVersion:                  sm.IpVersion(sm.IpVersion_value[h["ip_version"].(string)]),
			Method:                     sm.HttpMethod(sm.HttpMethod_value[h["method"].(string)]),
			Headers:                    common.SetToStringSlice(h["headers"].(*schema.Set)),
			Body:                       h["body"].(string),
			NoFollowRedirects:          h["no_follow_redirects"].(bool),
			BearerToken:                h["bearer_token"].(string),
			ProxyURL:                   h["proxy_url"].(string),
			ProxyConnectHeaders:        common.SetToStringSlice(h["proxy_connect_headers"].(*schema.Set)),
			FailIfSSL:                  h["fail_if_ssl"].(bool),
			FailIfNotSSL:               h["fail_if_not_ssl"].(bool),
			ValidHTTPVersions:          common.SetToStringSlice(h["valid_http_versions"].(*schema.Set)),
			FailIfBodyMatchesRegexp:    common.SetToStringSlice(h["fail_if_body_matches_regexp"].(*schema.Set)),
			FailIfBodyNotMatchesRegexp: common.SetToStringSlice(h["fail_if_body_not_matches_regexp"].(*schema.Set)),
			CacheBustingQueryParamName: h["cache_busting_query_param_name"].(string),
		}
		compression, ok := h["compression"].(string)
		if ok {
			cs.Http.Compression = sm.CompressionAlgorithm(sm.CompressionAlgorithm_value[compression])
		}
		if h["tls_config"].(*schema.Set).Len() > 0 {
			cs.Http.TlsConfig = tlsConfig(h["tls_config"].(*schema.Set))
		}
		if h["basic_auth"].(*schema.Set).Len() > 0 {
			ba := h["basic_auth"].(*schema.Set).List()[0].(map[string]interface{})
			cs.Http.BasicAuth = &sm.BasicAuth{
				Username: ba["username"].(string),
				Password: ba["password"].(string),
			}
		}
		if h["valid_status_codes"].(*schema.Set).Len() > 0 {
			for _, v := range h["valid_status_codes"].(*schema.Set).List() {
				cs.Http.ValidStatusCodes = append(cs.Http.ValidStatusCodes, int32(v.(int))) //nolint:gosec
			}
		}
		headerMatch := func(hms *schema.Set) []sm.HeaderMatch {
			smhm := []sm.HeaderMatch{}
			for _, hm := range hms.List() {
				smhm = append(smhm, sm.HeaderMatch{
					Header:       hm.(map[string]interface{})["header"].(string),
					Regexp:       hm.(map[string]interface{})["regexp"].(string),
					AllowMissing: hm.(map[string]interface{})["allow_missing"].(bool),
				})
			}
			return smhm
		}
		if h["fail_if_header_matches_regexp"].(*schema.Set).Len() > 0 {
			cs.Http.FailIfHeaderMatchesRegexp = headerMatch(h["fail_if_header_matches_regexp"].(*schema.Set))
		}
		if h["fail_if_header_not_matches_regexp"].(*schema.Set).Len() > 0 {
			cs.Http.FailIfHeaderNotMatchesRegexp = headerMatch(h["fail_if_header_not_matches_regexp"].(*schema.Set))
		}
	}

	ping := settings["ping"].(*schema.Set).List()
	if len(ping) > 0 {
		p := ping[0].(map[string]interface{})
		cs.Ping = &sm.PingSettings{
			IpVersion:       sm.IpVersion(sm.IpVersion_value[p["ip_version"].(string)]),
			SourceIpAddress: p["source_ip_address"].(string),
			PayloadSize:     int64(p["payload_size"].(int)),
			DontFragment:    p["dont_fragment"].(bool),
		}
	}

	tcp := settings["tcp"].(*schema.Set).List()
	if len(tcp) > 0 {
		t := tcp[0].(map[string]interface{})
		cs.Tcp = &sm.TcpSettings{
			IpVersion:       sm.IpVersion(sm.IpVersion_value[t["ip_version"].(string)]),
			SourceIpAddress: t["source_ip_address"].(string),
			Tls:             t["tls"].(bool),
		}
		if t["tls_config"].(*schema.Set).Len() > 0 {
			cs.Tcp.TlsConfig = tlsConfig(t["tls_config"].(*schema.Set))
		}
		if t["query_response"].(*schema.Set).Len() > 0 {
			for _, qr := range t["query_response"].(*schema.Set).List() {
				cs.Tcp.QueryResponse = append(cs.Tcp.QueryResponse, sm.TCPQueryResponse{
					Send:     []byte(qr.(map[string]interface{})["send"].(string)),
					Expect:   []byte(qr.(map[string]interface{})["expect"].(string)),
					StartTLS: qr.(map[string]interface{})["start_tls"].(bool),
				})
			}
		}
	}

	traceroute := settings["traceroute"].(*schema.Set).List()
	if len(traceroute) > 0 {
		t := traceroute[0].(map[string]interface{})
		cs.Traceroute = &sm.TracerouteSettings{
			MaxHops:        int64(t["max_hops"].(int)),
			MaxUnknownHops: int64(t["max_unknown_hops"].(int)),
			PtrLookup:      t["ptr_lookup"].(bool),
		}
	}

	multihttp := settings["multihttp"].(*schema.Set).List()
	if len(multihttp) > 0 {
		m := multihttp[0].(map[string]interface{})
		err := makeMultiHTTPSettings(m, &cs)
		if err != nil {
			return cs, fmt.Errorf("invalid MultiHTTP settings: %w", err)
		}
	}

	scripted := settings["scripted"].(*schema.Set).List()
	if len(scripted) > 0 {
		s := scripted[0].(map[string]interface{})
		cs.Scripted = &sm.ScriptedSettings{
			Script: []byte(s["script"].(string)),
		}
	}

	grpc := settings["grpc"].(*schema.Set).List()
	if len(grpc) > 0 {
		t := grpc[0].(map[string]interface{})
		cs.Grpc = &sm.GrpcSettings{
			Service:   t["service"].(string),
			IpVersion: sm.IpVersion(sm.IpVersion_value[t["ip_version"].(string)]),
			Tls:       t["tls"].(bool),
		}
		if t["tls_config"].(*schema.Set).Len() > 0 {
			cs.Grpc.TlsConfig = tlsConfig(t["tls_config"].(*schema.Set))
		}
	}

	browser := settings["browser"].(*schema.Set).List()
	if len(browser) > 0 {
		s := browser[0].(map[string]interface{})
		cs.Browser = &sm.BrowserSettings{
			Script: []byte(s["script"].(string)),
		}
	}

	return cs, nil
}

// Check if the user provider exactly one setting
// Ideally, we'd use `ExactlyOneOf` here but it doesn't support TypeSet.
// Also, TypeSet doesn't support ValidateFunc.
// To maintain backwards compatibility, we do a custom validation in the CustomizeDiff function.
func resourceCheckCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	settingsList := diff.Get("settings").(*schema.Set).List()
	if len(settingsList) == 0 {
		return fmt.Errorf("at least one check setting must be defined")
	}
	settings, ok := settingsList[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("at least one check setting must be defined")
	}

	count := 0
	for k := range syntheticMonitoringCheckSettings.Schema {
		count += len(settings[k].(*schema.Set).List())
	}

	if count != 1 {
		return fmt.Errorf("exactly one check setting must be defined, got %d", count)
	}

	return nil
}

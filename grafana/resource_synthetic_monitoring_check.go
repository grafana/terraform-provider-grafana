package grafana

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
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

	// HTTP and TCP checks can set TLS config.
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
			},
			"proxy_url": {
				Description: "Proxy URL.",
				Type:        schema.TypeString,
				Optional:    true,
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
				Description: "List of valid HTTP versions. Options include `HTTP/1.0`, `HTTP/1.1`, `HTTP/2`",
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
)

func ResourceSyntheticMonitoringCheck() *schema.Resource {
	return &schema.Resource{

		Description: `
Synthetic Monitoring checks are tests that run on selected probes at defined
intervals and report metrics and logs back to your Grafana Cloud account. The
target for checks can be a domain name, a server, or a website, depending on
what information you would like to gather about your endpoint. You can define
multiple checks for a single endpoint to check different capabilities.

* [Official documentation](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/checks/)
`,

		CreateContext: resourceSyntheticMonitoringCheckCreate,
		ReadContext:   resourceSyntheticMonitoringCheckRead,
		UpdateContext: resourceSyntheticMonitoringCheckUpdate,
		DeleteContext: resourceSyntheticMonitoringCheckDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: resourceSyntheticMonitoringCheckCustomizeDiff,

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
					"The minimum acceptable value is 1 second (1000 ms), and the maximum is 120 seconds (120000 ms).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60000,
				ValidateFunc: validation.IntBetween(1000, 120000),
			},
			"timeout": {
				Description: "Specifies the maximum running time for the check in milliseconds. " +
					"The minimum acceptable value is 1 second (1000 ms), and the maximum 10 seconds (10000 ms).",
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3000,
			},
			"enabled": {
				Description: "Whether to enable the check.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"alert_sensitivity": {
				Description: "Can be set to `none`, `low`, `medium`, or `high` to correspond to the check [alert levels](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/synthetic-monitoring-alerting/).",
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
}

func resourceSyntheticMonitoringCheckCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	chk := makeCheck(d)
	res, err := c.AddCheck(ctx, *chk)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(res.Id, 10))
	d.Set("tenant_id", res.TenantId)
	return resourceSyntheticMonitoringCheckRead(ctx, d, meta)
}

func resourceSyntheticMonitoringCheckRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	chk, err := c.GetCheck(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			log.Printf("[WARN] removing check %s from state because it no longer exists", d.Id())
			d.SetId("")
			return nil
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
						"fail_if_matches_regexp":     stringSliceToSet(v.FailIfMatchesRegexp),
						"fail_if_not_matches_regexp": stringSliceToSet(v.FailIfNotMatchesRegexp),
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
			"valid_r_codes":           stringSliceToSet(chk.Settings.Dns.ValidRCodes),
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
			"headers":                           stringSliceToSet(chk.Settings.Http.Headers),
			"body":                              chk.Settings.Http.Body,
			"no_follow_redirects":               chk.Settings.Http.NoFollowRedirects,
			"basic_auth":                        &basicAuth,
			"bearer_token":                      chk.Settings.Http.BearerToken,
			"proxy_url":                         chk.Settings.Http.ProxyURL,
			"fail_if_ssl":                       chk.Settings.Http.FailIfSSL,
			"fail_if_not_ssl":                   chk.Settings.Http.FailIfNotSSL,
			"valid_status_codes":                int32SliceToSet(chk.Settings.Http.ValidStatusCodes),
			"valid_http_versions":               stringSliceToSet(chk.Settings.Http.ValidHTTPVersions),
			"fail_if_body_matches_regexp":       stringSliceToSet(chk.Settings.Http.FailIfBodyMatchesRegexp),
			"fail_if_body_not_matches_regexp":   stringSliceToSet(chk.Settings.Http.FailIfBodyNotMatchesRegexp),
			"fail_if_header_matches_regexp":     headerMatch(chk.Settings.Http.FailIfHeaderMatchesRegexp),
			"fail_if_header_not_matches_regexp": headerMatch(chk.Settings.Http.FailIfHeaderNotMatchesRegexp),
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
	}

	d.Set("settings", settings)

	return nil
}

func resourceSyntheticMonitoringCheckUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	chk := makeCheck(d)
	_, err := c.UpdateCheck(ctx, *chk)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceSyntheticMonitoringCheckRead(ctx, d, meta)
}

func resourceSyntheticMonitoringCheckDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	var diags diag.Diagnostics
	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	err := c.DeleteCheck(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}

// makeCheck populates an instance of sm.Check. We need this for create and
// update calls with the SM API client.
func makeCheck(d *schema.ResourceData) *sm.Check {
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

	return &sm.Check{
		Id:               id,
		TenantId:         int64(d.Get("tenant_id").(int)),
		Job:              d.Get("job").(string),
		Target:           d.Get("target").(string),
		Frequency:        int64(d.Get("frequency").(int)),
		Timeout:          int64(d.Get("timeout").(int)),
		Enabled:          d.Get("enabled").(bool),
		AlertSensitivity: d.Get("alert_sensitivity").(string),
		BasicMetricsOnly: d.Get("basic_metrics_only").(bool),
		Probes:           probes,
		Labels:           labels,
		Settings:         makeCheckSettings(d.Get("settings").(*schema.Set).List()[0].(map[string]interface{})),
	}
}

// makeCheckSettings populates an instance of sm.CheckSettings. This is called
// by makeCheck. It's isolated from makeCheck to hopefully make it all more
// human readable.
func makeCheckSettings(settings map[string]interface{}) sm.CheckSettings {
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
			Port:            int32(d["port"].(int)),
			RecordType:      sm.DnsRecordType(sm.DnsRecordType_value[d["record_type"].(string)]),
			Protocol:        sm.DnsProtocol(sm.DnsProtocol_value[d["protocol"].(string)]),
			ValidRCodes:     setToStringSlice(d["valid_r_codes"].(*schema.Set)),
		}
		dnsValidator := func(validation string) *sm.DNSRRValidator {
			val := sm.DNSRRValidator{}
			for _, v := range d[validation].(*schema.Set).List() {
				val.FailIfMatchesRegexp = setToStringSlice(v.(map[string]interface{})["fail_if_matches_regexp"].(*schema.Set))
				val.FailIfNotMatchesRegexp = setToStringSlice(v.(map[string]interface{})["fail_if_not_matches_regexp"].(*schema.Set))
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
			Headers:                    setToStringSlice(h["headers"].(*schema.Set)),
			Body:                       h["body"].(string),
			NoFollowRedirects:          h["no_follow_redirects"].(bool),
			BearerToken:                h["bearer_token"].(string),
			ProxyURL:                   h["proxy_url"].(string),
			FailIfSSL:                  h["fail_if_ssl"].(bool),
			FailIfNotSSL:               h["fail_if_not_ssl"].(bool),
			ValidHTTPVersions:          setToStringSlice(h["valid_http_versions"].(*schema.Set)),
			FailIfBodyMatchesRegexp:    setToStringSlice(h["fail_if_body_matches_regexp"].(*schema.Set)),
			FailIfBodyNotMatchesRegexp: setToStringSlice(h["fail_if_body_not_matches_regexp"].(*schema.Set)),
			CacheBustingQueryParamName: h["cache_busting_query_param_name"].(string),
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
				cs.Http.ValidStatusCodes = append(cs.Http.ValidStatusCodes, int32(v.(int)))
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

	return cs
}

// Check if the user provider exactly one setting
// Ideally, we'd use `ExactlyOneOf` here but it doesn't support TypeSet.
// Also, TypeSet doesn't support ValidateFunc.
// To maintain backwards compatibility, we do a custom validation in the CustomizeDiff function.
func resourceSyntheticMonitoringCheckCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	settingsList := diff.Get("settings").(*schema.Set).List()
	if len(settingsList) == 0 {
		return fmt.Errorf("at least one check setting must be defined")
	}
	settings := settingsList[0].(map[string]interface{})

	count := 0
	for k := range syntheticMonitoringCheckSettings.Schema {
		count += len(settings[k].(*schema.Set).List())
	}

	if count != 1 {
		return fmt.Errorf("exactly one check setting must be defined, got %d", count)
	}

	return nil
}

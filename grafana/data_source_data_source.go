package grafana

import (
	"regexp"
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceDatasource() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [DataSource HTTP API](https://grafana.com/docs/grafana/latest/http_api/data_source/)
`,
		ReadContext: dataSourceDataSourceRead,
		Schema: map[string]*schema.Schema{
			"access_mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "proxy",
				Description: "The method by which Grafana will access the data source: `proxy` or `direct`.",
			},
			"basic_auth_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to enable basic auth for the data source.",
			},
			"basic_auth_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "Basic auth password. Deprecated: Use secure_json_data.basic_auth_password instead. This attribute is removed in Grafana 9.0+.",
				Deprecated:  "Use secure_json_data.basic_auth_password instead. This attribute is removed in Grafana 9.0+.",
			},
			"basic_auth_username": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Basic auth username.",
			},
			"database_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "(Required by some data source types) The name of the database to use on the selected data source server.",
			},
			"http_headers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Description: "Custom HTTP headers",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"is_default": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to set the data source as default. This should only be `true` to a single data source.",
			},
			"uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Unique identifier. If unset, this will be automatically generated.",
			},

			"json_data": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "(Required by some data source types). Deprecated: Use json_data_encoded instead. It supports arbitrary JSON data, and therefore all attributes.",
				Deprecated:  "Use json_data_encoded instead. It supports arbitrary JSON data, and therefore all attributes.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"assume_role_arn": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(CloudWatch, Athena) The ARN of the role to be assumed by Grafana when using the CloudWatch or Athena data source.",
						},
						"auth_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(CloudWatch, Athena) The authentication type used to access the data source.",
						},
						"authentication_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Stackdriver) The authentication type: `jwt` or `gce`.",
						},
						"catalog": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Athena) Athena catalog.",
						},
						"client_email": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Stackdriver) Service account email address.",
						},
						"client_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Azure Monitor) The service account client id.",
						},
						"cloud_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Azure Monitor) The cloud name.",
						},
						"conn_max_lifetime": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(MySQL, PostgreSQL, and MSSQL) Maximum amount of time in seconds a connection may be reused (Grafana v5.4+).",
						},
						"custom_metrics_namespaces": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(CloudWatch) A comma-separated list of custom namespaces to be queried by the CloudWatch data source.",
						},
						"database": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Athena) Name of the database within the catalog.",
						},
						"default_bucket": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(InfluxDB) The default bucket for the data source.",
						},
						"default_project": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Stackdriver) The default project for the data source.",
						},
						"default_region": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(CloudWatch, Athena) The default region for the data source.",
						},
						"derived_field": {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"matcher_regex": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"url": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"datasource_uid": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							Optional:    true,
							Description: "(Loki) See https://grafana.com/docs/grafana/latest/datasources/loki/#derived-fields",
						},
						"encrypt": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(MSSQL) Connection SSL encryption handling: 'disable', 'false' or 'true'.",
						},
						"es_version": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch) Elasticsearch semantic version (Grafana v8.0+).",
							ValidateDiagFunc: func(v interface{}, p cty.Path) diag.Diagnostics {
								var diags diag.Diagnostics
								r := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
								if !r.MatchString(v.(string)) {
									diags = append(diags, diag.Diagnostic{
										Severity: diag.Warning,
										Summary:  "Expected semantic version",
										Detail:   "As of Grafana 8.0, the Elasticsearch plugin expects es_version to be set as a semantic version (E.g. 7.0.0, 7.6.1).",
									})
								}
								return diags
							},
						},
						"external_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(CloudWatch, Athena) If you are assuming a role in another account, that has been created with an external ID, specify the external ID here.",
						},
						"github_url": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Github) Github URL",
						},
						"graphite_version": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Graphite) Graphite version.",
						},
						"http_method": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Prometheus) HTTP method to use for making requests.",
						},
						"interval": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch) Index date time format. nil(No Pattern), 'Hourly', 'Daily', 'Weekly', 'Monthly' or 'Yearly'.",
						},
						"implementation": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Alertmanager) Implementation of Alertmanager. Either 'cortex' or 'prometheus'",
						},
						"log_level_field": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch) Which field should be used to indicate the priority of the log message.",
						},
						"log_message_field": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch) Which field should be used as the log message.",
						},
						"manage_alerts": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "(Prometheus) Manage alerts.",
						},
						"max_concurrent_shard_requests": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(Elasticsearch) Maximum number of concurrent shard requests.",
						},
						"max_idle_conns": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(MySQL, PostgreSQL and MSSQL) Maximum number of connections in the idle connection pool (Grafana v5.4+).",
						},
						"max_lines": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(Loki) Upper limit for the number of log lines returned by Loki ",
						},
						"max_open_conns": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(MySQL, PostgreSQL and MSSQL) Maximum number of open connections to the database (Grafana v5.4+).",
						},
						"organization": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(InfluxDB) An organization is a workspace for a group of users. All dashboards, tasks, buckets, members, etc., belong to an organization.",
						},
						"org_slug": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Sentry) Organization slug.",
						},
						"output_location": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Athena) AWS S3 bucket to store execution outputs. If not specified, the default query result location from the Workgroup configuration will be used.",
						},
						"postgres_version": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(PostgreSQL) Postgres version as a number (903/904/905/906/1000) meaning v9.3, v9.4, etc.",
						},
						"profile": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(CloudWatch, Athena) The credentials profile name to use when authentication type is set as 'Credentials file'.",
						},
						"query_timeout": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Prometheus) Timeout for queries made to the Prometheus data source in seconds.",
						},
						"sigv4_assume_role_arn": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch and Prometheus) Specifies the ARN of an IAM role to assume.",
						},
						"sigv4_auth": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "(Elasticsearch and Prometheus) Enable usage of SigV4.",
						},
						"sigv4_auth_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch and Prometheus) The Sigv4 authentication provider to use: 'default', 'credentials' or 'keys' (AMG: 'workspace-iam-role').",
						},
						"sigv4_external_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch and Prometheus) When assuming a role in another account use this external ID.",
						},
						"sigv4_profile": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch and Prometheus) Credentials profile name, leave blank for default.",
						},
						"sigv4_region": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch and Prometheus) AWS region to use for Sigv4.",
						},
						"ssl_mode": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(PostgreSQL) SSLmode. 'disable', 'require', 'verify-ca' or 'verify-full'.",
						},
						"subscription_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Azure Monitor) The subscription id",
						},
						"tenant_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Azure Monitor) Service account tenant ID.",
						},
						"timescaledb": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "(PostgreSQL) Enable usage of TimescaleDB extension.",
						},
						"time_field": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Elasticsearch) Which field that should be used as timestamp.",
						},
						"time_interval": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: `(Prometheus, Elasticsearch, InfluxDB, MySQL, PostgreSQL, and MSSQL) Lowest interval/step value that should be used for this data source. Sometimes called "Scrape Interval" in the Grafana UI.`,
						},
						"tls_auth": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "(All) Enable TLS authentication using client cert configured in secure json data.",
						},
						"tls_auth_with_ca_cert": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "(All) Enable TLS authentication using CA cert.",
						},
						"tls_configuration_method": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "(All) SSL Certificate configuration, either by ‘file-path’ or ‘file-content’.",
							ValidateFunc: validation.StringInSlice([]string{"file-path", "file-content"}, false),
						},
						"tls_skip_verify": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "(All) Controls whether a client verifies the server’s certificate chain and host name.",
						},
						"token_uri": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Stackdriver) The token URI used, provided in the service account key.",
						},
						"tracing_datasource_uid": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Cloudwatch) The X-Ray datasource uid to associate to this Cloudwatch datasource.",
						},
						"tsdb_resolution": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(OpenTSDB) Resolution.",
						},
						"tsdb_version": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "(OpenTSDB) Version.",
						},
						"version": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(InfluxDB) InfluxQL or Flux.",
						},
						"workgroup": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Athena) Workgroup to use.",
						},
						"xpack_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "(Elasticsearch) Enable X-Pack support.",
						},
					},
				},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A unique name for the data source.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "(Required by some data source types) The password to use to authenticate to the data source. Deprecated: Use secure_json_data.password instead. This attribute is removed in Grafana 9.0+.",
				Deprecated:  "Use secure_json_data.password instead. This attribute is removed in Grafana 9.0+.",
			},
			"secure_json_data": {
				Type:        schema.TypeList,
				Optional:    true,
				Sensitive:   true,
				Description: "Deprecated: Use secure_json_data instead. It supports arbitrary JSON data, and therefore all attributes.",
				Deprecated:  "Use secure_json_data instead. It supports arbitrary JSON data, and therefore all attributes.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(CloudWatch, Athena) The access key used to access the data source.",
						},
						"access_token": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(Github) The access token used to access the data source.",
						},
						"auth_token": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(Sentry) Authorization token.",
						},
						"basic_auth_password": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(All) Password to use for basic authentication.",
						},
						"client_secret": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(Azure Monitor) Client secret for authentication.",
						},
						"password": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(All) Password to use for authentication.",
						},
						"private_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(Stackdriver) The service account key `private_key` to use to access the data source.",
						},
						"secret_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(CloudWatch, Athena) The secret key to use to access the data source.",
						},
						"sigv4_access_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(Elasticsearch and Prometheus) SigV4 access key. Required when using 'keys' auth provider.",
						},
						"sigv4_secret_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(Elasticsearch and Prometheus) SigV4 secret key. Required when using 'keys' auth provider.",
						},
						"tls_ca_cert": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(All) CA cert for out going requests.",
						},
						"tls_client_cert": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(All) TLS Client cert for outgoing requests.",
						},
						"tls_client_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(All) TLS Client key for outgoing requests.",
						},
					},
				},
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The data source type. Must be one of the supported data source keywords.",
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The URL for the data source. The type of URL required varies depending on the chosen data source type.",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "(Required by some data source types) The username to use to authenticate to the data source.",
			},
			"json_data_encoded": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"json_data", "secure_json_data"},
				Description:   "Serialized JSON string containing the json data. Replaces the json_data attribute, this attribute can be used to pass configuration options to the data source. To figure out what options a datasource has available, see its docs or inspect the network data when saving it from the Grafana UI.",
				ValidateFunc:  validation.StringIsJSON,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				DiffSuppressFunc: SuppressEquivalentJSONDiffs,
			},
			"secure_json_data_encoded": {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"json_data", "secure_json_data"},
				Description:   "Serialized JSON string containing the secure json data. Replaces the secure_json_data attribute, this attribute can be used to pass secure configuration options to the data source. To figure out what options a datasource has available, see its docs or inspect the network data when saving it from the Grafana UI.",
				ValidateFunc:  validation.StringIsJSON,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				DiffSuppressFunc: SuppressEquivalentJSONDiffs,
			},
		},
	}
}

// search DataSource by Name
func dataSourceDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	gapiURL := meta.(*client).gapiURL
	var dataspource *gapi.Datasource
	client := meta.(*client).gapi

	id, err := client.DataSourceIDByName(name)
	if err != nil {
		return nil, err
	}
	
	return client.DataSource(id)
}

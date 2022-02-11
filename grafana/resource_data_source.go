package grafana

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceDataSource() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/data_source/)

The required arguments for this resource vary depending on the type of data
source selected (via the 'type' argument).
`,

		CreateContext: CreateDataSource,
		UpdateContext: UpdateDataSource,
		DeleteContext: DeleteDataSource,
		ReadContext:   ReadDataSource,

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
				Description: "Basic auth password.",
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
				Description: "(Required by some data source types)",
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
							Description: "(Prometheus, Elasticsearch, InfluxDB, MySQL, PostgreSQL, and MSSQL) Lowest interval/step value that should be used for this data source.",
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
						"tsdb_resolution": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(OpenTSDB) Resolution.",
						},
						"tsdb_version": {
							Type:        schema.TypeString,
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
				Description: "(Required by some data source types) The password to use to authenticate to the data source.",
			},
			"secure_json_data": {
				Type:        schema.TypeList,
				Optional:    true,
				Sensitive:   true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(CloudWatch, Athena) The access key to use to access the data source.",
						},
						"access_token": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "(Github) The access token to use to access the data source",
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
		},
	}
}

// CreateDataSource creates a Grafana datasource
func CreateDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	dataSource, err := makeDataSource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := client.NewDataSource(dataSource)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(id, 10))

	return ReadDataSource(ctx, d, meta)
}

// UpdateDataSource updates a Grafana datasource
func UpdateDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	dataSource, err := makeDataSource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = client.UpdateDataSource(dataSource); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

// ReadDataSource reads a Grafana datasource
func ReadDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("Invalid id: %#v", idStr)
	}

	dataSource, err := client.DataSource(id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing datasource %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(dataSource.ID, 10))
	d.Set("access_mode", dataSource.Access)
	d.Set("database_name", dataSource.Database)
	d.Set("is_default", dataSource.IsDefault)
	d.Set("name", dataSource.Name)
	d.Set("type", dataSource.Type)
	d.Set("url", dataSource.URL)
	d.Set("username", dataSource.User)
	d.Set("uid", dataSource.UID)

	// TODO: these fields should be migrated to SecureJSONData.
	d.Set("basic_auth_enabled", dataSource.BasicAuth)
	d.Set("basic_auth_username", dataSource.BasicAuthUser)     //nolint:staticcheck // deprecated
	d.Set("basic_auth_password", dataSource.BasicAuthPassword) //nolint:staticcheck // deprecated
	d.Set("password", dataSource.Password)                     //nolint:staticcheck // deprecated

	return nil
}

// DeleteDataSource deletes a Grafana datasource
func DeleteDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("Invalid id: %#v", idStr)
	}

	if err = client.DeleteDataSource(id); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func makeDataSource(d *schema.ResourceData) (*gapi.DataSource, error) {
	idStr := d.Id()
	var id int64
	var err error
	if idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	httpHeaders := make(map[string]string)
	for key, value := range d.Get("http_headers").(map[string]interface{}) {
		httpHeaders[key] = fmt.Sprintf("%v", value)
	}

	return &gapi.DataSource{
		ID:                id,
		Name:              d.Get("name").(string),
		Type:              d.Get("type").(string),
		URL:               d.Get("url").(string),
		Access:            d.Get("access_mode").(string),
		Database:          d.Get("database_name").(string),
		User:              d.Get("username").(string),
		Password:          d.Get("password").(string),
		IsDefault:         d.Get("is_default").(bool),
		BasicAuth:         d.Get("basic_auth_enabled").(bool),
		BasicAuthUser:     d.Get("basic_auth_username").(string),
		BasicAuthPassword: d.Get("basic_auth_password").(string),
		UID:               d.Get("uid").(string),
		HTTPHeaders:       httpHeaders,
		JSONData:          makeJSONData(d),
		SecureJSONData:    makeSecureJSONData(d),
	}, err
}

func makeJSONData(d *schema.ResourceData) gapi.JSONData {
	return gapi.JSONData{
		AssumeRoleArn:              d.Get("json_data.0.assume_role_arn").(string),
		AuthType:                   d.Get("json_data.0.auth_type").(string),
		AuthenticationType:         d.Get("json_data.0.authentication_type").(string),
		Catalog:                    d.Get("json_data.0.catalog").(string),
		ClientEmail:                d.Get("json_data.0.client_email").(string),
		ConnMaxLifetime:            int64(d.Get("json_data.0.conn_max_lifetime").(int)),
		CustomMetricsNamespaces:    d.Get("json_data.0.custom_metrics_namespaces").(string),
		Database:                   d.Get("json_data.0.database").(string),
		DefaultBucket:              d.Get("json_data.0.default_bucket").(string),
		DefaultProject:             d.Get("json_data.0.default_project").(string),
		DefaultRegion:              d.Get("json_data.0.default_region").(string),
		Encrypt:                    d.Get("json_data.0.encrypt").(string),
		EsVersion:                  d.Get("json_data.0.es_version").(string),
		ExternalID:                 d.Get("json_data.0.external_id").(string),
		GraphiteVersion:            d.Get("json_data.0.graphite_version").(string),
		HTTPMethod:                 d.Get("json_data.0.http_method").(string),
		Interval:                   d.Get("json_data.0.interval").(string),
		LogLevelField:              d.Get("json_data.0.log_level_field").(string),
		LogMessageField:            d.Get("json_data.0.log_message_field").(string),
		MaxConcurrentShardRequests: int64(d.Get("json_data.0.max_concurrent_shard_requests").(int)),
		MaxIdleConns:               int64(d.Get("json_data.0.max_idle_conns").(int)),
		MaxOpenConns:               int64(d.Get("json_data.0.max_open_conns").(int)),
		Organization:               d.Get("json_data.0.organization").(string),
		OrgSlug:                    d.Get("json_data.0.org_slug").(string),
		OutputLocation:             d.Get("json_data.0.output_location").(string),
		PostgresVersion:            int64(d.Get("json_data.0.postgres_version").(int)),
		Profile:                    d.Get("json_data.0.profile").(string),
		QueryTimeout:               d.Get("json_data.0.query_timeout").(string),
		SigV4AssumeRoleArn:         d.Get("json_data.0.sigv4_assume_role_arn").(string),
		SigV4Auth:                  d.Get("json_data.0.sigv4_auth").(bool),
		SigV4AuthType:              d.Get("json_data.0.sigv4_auth_type").(string),
		SigV4ExternalID:            d.Get("json_data.0.sigv4_external_id").(string),
		SigV4Profile:               d.Get("json_data.0.sigv4_profile").(string),
		SigV4Region:                d.Get("json_data.0.sigv4_region").(string),
		Sslmode:                    d.Get("json_data.0.ssl_mode").(string),
		Timescaledb:                d.Get("json_data.0.timescaledb").(bool),
		TimeField:                  d.Get("json_data.0.time_field").(string),
		TimeInterval:               d.Get("json_data.0.time_interval").(string),
		TLSAuth:                    d.Get("json_data.0.tls_auth").(bool),
		TLSAuthWithCACert:          d.Get("json_data.0.tls_auth_with_ca_cert").(bool),
		TLSSkipVerify:              d.Get("json_data.0.tls_skip_verify").(bool),
		TokenURI:                   d.Get("json_data.0.token_uri").(string),
		TsdbResolution:             d.Get("json_data.0.tsdb_resolution").(string),
		TsdbVersion:                d.Get("json_data.0.tsdb_version").(string),
		Version:                    d.Get("json_data.0.version").(string),
		Workgroup:                  d.Get("json_data.0.workgroup").(string),
	}
}

func makeSecureJSONData(d *schema.ResourceData) gapi.SecureJSONData {
	return gapi.SecureJSONData{
		AccessKey:         d.Get("secure_json_data.0.access_key").(string),
		AuthToken:         d.Get("secure_json_data.0.auth_token").(string),
		BasicAuthPassword: d.Get("secure_json_data.0.basic_auth_password").(string),
		Password:          d.Get("secure_json_data.0.password").(string),
		PrivateKey:        d.Get("secure_json_data.0.private_key").(string),
		SecretKey:         d.Get("secure_json_data.0.secret_key").(string),
		SigV4AccessKey:    d.Get("secure_json_data.0.sigv4_access_key").(string),
		SigV4SecretKey:    d.Get("secure_json_data.0.sigv4_secret_key").(string),
		TLSCACert:         d.Get("secure_json_data.0.tls_ca_cert").(string),
		TLSClientCert:     d.Get("secure_json_data.0.tls_client_cert").(string),
		TLSClientKey:      d.Get("secure_json_data.0.tls_client_key").(string),
	}
}

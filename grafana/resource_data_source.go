package grafana

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	gapi "github.com/nytm/go-grafana-api"
)

func ResourceDataSource() *schema.Resource {
	return &schema.Resource{
		Create: CreateDataSource,
		Update: UpdateDataSource,
		Delete: DeleteDataSource,
		Read:   ReadDataSource,

		Schema: map[string]*schema.Schema{
			"access_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "proxy",
			},
			"basic_auth_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"basic_auth_password": {
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},
			"basic_auth_username": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"database_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"is_default": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"json_data": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"assume_role_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"auth_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"conn_max_lifetime": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"custom_metrics_namespaces": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"default_region": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"encrypt": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"es_version": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"graphite_version": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"http_method": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"interval": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"log_level_field": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"log_message_field": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"max_idle_conns": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"max_open_conns": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"postgres_version": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"query_timeout": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ssl_mode": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"timescaledb": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"time_field": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"time_interval": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tls_auth": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"tls_auth_with_ca_cert": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"tls_skip_verify": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"tsdb_resolution": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tsdb_version": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},
			"secure_json_data": {
				Type:      schema.TypeList,
				Optional:  true,
				Sensitive: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_key": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"basic_auth_password": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"password": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"private_key": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"secret_key": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"tls_ca_cert": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"tls_client_cert": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"tls_client_key": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
					},
				},
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"username": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
		},
	}
}

// CreateDataSource creates a Grafana datasource
func CreateDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dataSource, err := makeDataSource(d)
	if err != nil {
		return err
	}

	id, err := client.NewDataSource(dataSource)
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(id, 10))

	return ReadDataSource(d, meta)
}

// UpdateDataSource updates a Grafana datasource
func UpdateDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dataSource, err := makeDataSource(d)
	if err != nil {
		return err
	}

	return client.UpdateDataSource(dataSource)
}

// ReadDataSource reads a Grafana datasource
func ReadDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid id: %#v", idStr)
	}

	dataSource, err := client.DataSource(id)
	if err != nil {
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] removing datasource %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("id", dataSource.Id)
	d.Set("access_mode", dataSource.Access)
	d.Set("basic_auth_enabled", dataSource.BasicAuth)
	d.Set("basic_auth_username", dataSource.BasicAuthUser)
	d.Set("basic_auth_password", dataSource.BasicAuthPassword)
	d.Set("database_name", dataSource.Database)
	d.Set("is_default", dataSource.IsDefault)
	d.Set("name", dataSource.Name)
	d.Set("password", dataSource.Password)
	d.Set("type", dataSource.Type)
	d.Set("url", dataSource.URL)
	d.Set("username", dataSource.User)

	return nil
}

// DeleteDataSource deletes a Grafana datasource
func DeleteDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid id: %#v", idStr)
	}

	return client.DeleteDataSource(id)
}

func makeDataSource(d *schema.ResourceData) (*gapi.DataSource, error) {
	idStr := d.Id()
	var id int64
	var err error
	if idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	return &gapi.DataSource{
		Id:                id,
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
		JSONData:          makeJSONData(d),
		SecureJSONData:    makeSecureJSONData(d),
	}, err
}

func makeJSONData(d *schema.ResourceData) gapi.JSONData {
	return gapi.JSONData{
		AssumeRoleArn:           d.Get("json_data.0.assume_role_arn").(string),
		AuthType:                d.Get("json_data.0.auth_type").(string),
		ConnMaxLifetime:         int64(d.Get("json_data.0.conn_max_lifetime").(int)),
		CustomMetricsNamespaces: d.Get("json_data.0.custom_metrics_namespaces").(string),
		DefaultRegion:           d.Get("json_data.0.default_region").(string),
		Encrypt:                 d.Get("json_data.0.encrypt").(string),
		EsVersion:               int64(d.Get("json_data.0.es_version").(int)),
		GraphiteVersion:         d.Get("json_data.0.graphite_version").(string),
		HttpMethod:              d.Get("json_data.0.http_method").(string),
		Interval:                d.Get("json_data.0.interval").(string),
		LogLevelField:           d.Get("json_data.0.log_level_field").(string),
		LogMessageField:         d.Get("json_data.0.log_message_field").(string),
		MaxIdleConns:            int64(d.Get("json_data.0.max_idle_conns").(int)),
		MaxOpenConns:            int64(d.Get("json_data.0.max_open_conns").(int)),
		PostgresVersion:         int64(d.Get("json_data.0.postgres_version").(int)),
		QueryTimeout:            d.Get("json_data.0.query_timeout").(string),
		Sslmode:                 d.Get("json_data.0.ssl_mode").(string),
		Timescaledb:             d.Get("json_data.0.timescaledb").(bool),
		TimeField:               d.Get("json_data.0.time_field").(string),
		TimeInterval:            d.Get("json_data.0.time_interval").(string),
		TlsAuth:                 d.Get("json_data.0.tls_auth").(bool),
		TlsAuthWithCACert:       d.Get("json_data.0.tls_auth_with_ca_cert").(bool),
		TlsSkipVerify:           d.Get("json_data.0.tls_skip_verify").(bool),
		TsdbResolution:          d.Get("json_data.0.tsdb_resolution").(string),
		TsdbVersion:             d.Get("json_data.0.tsdb_version").(string),
	}
}

func makeSecureJSONData(d *schema.ResourceData) gapi.SecureJSONData {
	return gapi.SecureJSONData{
		AccessKey:         d.Get("secure_json_data.0.access_key").(string),
		BasicAuthPassword: d.Get("secure_json_data.0.basic_auth_password").(string),
		Password:          d.Get("secure_json_data.0.password").(string),
		PrivateKey:        d.Get("secure_json_data.0.private_key").(string),
		SecretKey:         d.Get("secure_json_data.0.secret_key").(string),
		TlsCACert:         d.Get("secure_json_data.0.tls_ca_cert").(string),
		TlsClientCert:     d.Get("secure_json_data.0.tls_client_cert").(string),
		TlsClientKey:      d.Get("secure_json_data.0.tls_client_key").(string),
	}
}

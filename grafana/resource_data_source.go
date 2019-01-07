package grafana

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	gapi "github.com/nytm/go-grafana-api"
)

func ResourceDataSource() *schema.Resource {
	return &schema.Resource{
		Create: CreateDataSource,
		Update: UpdateDataSource,
		Delete: DeleteDataSource,
		Read:   ReadDataSource,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"url": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"is_default": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"basic_auth_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"basic_auth_username": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"basic_auth_password": {
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},

			"username": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},

			"json_data": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"default_region": {
							Type:     schema.TypeString,
							Required: true,
						},
						"custom_metrics_namespaces": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"assume_role_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"secure_json_data": {
				Type:      schema.TypeList,
				Optional:  true,
				Sensitive: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"secret_key": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"database_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"access_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "proxy",
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
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] removing datasource %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Invalid id: %#v", idStr)
	}

	dataSource, err := client.DataSource(id)
	if err != nil {

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
		AuthType:                d.Get("json_data.0.auth_type").(string),
		DefaultRegion:           d.Get("json_data.0.default_region").(string),
		CustomMetricsNamespaces: d.Get("json_data.0.custom_metrics_namespaces").(string),
		AssumeRoleArn:           d.Get("json_data.0.assume_role_arn").(string),
	}
}

func makeSecureJSONData(d *schema.ResourceData) gapi.SecureJSONData {
	return gapi.SecureJSONData{
		AccessKey: d.Get("secure_json_data.0.access_key").(string),
		SecretKey: d.Get("secure_json_data.0.secret_key").(string),
	}
}

package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceDataSource() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/data_source/)

The required arguments for this resource vary depending on the type of data
source selected (via the 'type' argument).
`,

		CreateContext: CreateDataSource,
		UpdateContext: UpdateDataSource,
		DeleteContext: DeleteDataSource,
		ReadContext:   ReadDataSource,
		SchemaVersion: 1,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Unique identifier. If unset, this will be automatically generated.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A unique name for the data source.",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The data source type. Must be one of the supported data source keywords.",
			},
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
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Serialized JSON string containing the json data. This attribute can be used to pass configuration options to the data source. To figure out what options a datasource has available, see its docs or inspect the network data when saving it from the Grafana UI. Note that keys in this map are usually camelCased.",
				ValidateFunc: validation.StringIsJSON,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					if oldValue == "{}" && newValue == "" {
						return true
					}
					return common.SuppressEquivalentJSONDiffs(k, oldValue, newValue, d)
				},
			},
			"secure_json_data_encoded": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Description:  "Serialized JSON string containing the secure json data. This attribute can be used to pass secure configuration options to the data source. To figure out what options a datasource has available, see its docs or inspect the network data when saving it from the Grafana UI. Note that keys in this map are usually camelCased.",
				ValidateFunc: validation.StringIsJSON,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					if oldValue == "{}" && newValue == "" {
						return true
					}
					return common.SuppressEquivalentJSONDiffs(k, oldValue, newValue, d)
				},
			},
		},
	}
}

// CreateDataSource creates a Grafana datasource
func CreateDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromNewOrgResource(meta, d)

	dataSource, err := makeDataSource("", d)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := client.NewDataSource(dataSource)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))
	return ReadDataSource(ctx, d, meta)
}

// UpdateDataSource updates a Grafana datasource
func UpdateDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	dataSource, err := makeDataSource(idStr, d)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(client.UpdateDataSource(dataSource))
}

// ReadDataSource reads a Grafana datasource
func ReadDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	var dataSource *gapi.DataSource
	var err error
	// Support both numerical and UID IDs, so that we can import an existing datasource with either.
	// Following the read, it's normalized to a numerical ID.
	if numericalID, parseErr := strconv.ParseInt(idStr, 10, 64); parseErr == nil {
		dataSource, err = client.DataSource(numericalID)
	} else {
		dataSource, err = client.DataSourceByUID(idStr)
	}

	if err, shouldReturn := common.CheckReadError("datasource", d, err); shouldReturn {
		return err
	}

	return readDatasource(d, dataSource)
}

// DeleteDataSource deletes a Grafana datasource
func DeleteDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("Invalid id: %#v", idStr)
	}

	return diag.FromErr(client.DeleteDataSource(id))
}

func readDatasource(d *schema.ResourceData, dataSource *gapi.DataSource) diag.Diagnostics {
	d.SetId(MakeOrgResourceID(dataSource.OrgID, dataSource.ID))
	d.Set("access_mode", dataSource.Access)
	d.Set("database_name", dataSource.Database)
	d.Set("is_default", dataSource.IsDefault)
	d.Set("name", dataSource.Name)
	d.Set("type", dataSource.Type)
	d.Set("url", dataSource.URL)
	d.Set("username", dataSource.User)
	d.Set("uid", dataSource.UID)
	d.Set("org_id", strconv.FormatInt(dataSource.OrgID, 10))

	gottenJSONData, _, gottenHeaders := gapi.ExtractHeadersFromJSONData(dataSource.JSONData, dataSource.SecureJSONData)
	encodedJSONData, err := json.Marshal(gottenJSONData)
	if err != nil {
		return diag.Errorf("Failed to marshal JSON data: %s", err)
	}
	d.Set("json_data_encoded", string(encodedJSONData))

	// For headers, we do not know the value (the API does not return secret data)
	// so we only remove keys from the state that are no longer present in the API.
	if currentHeadersInterface, ok := d.GetOk("http_headers"); ok {
		currentHeaders := currentHeadersInterface.(map[string]interface{})
		for key := range currentHeaders {
			if _, ok := gottenHeaders[key]; !ok {
				delete(currentHeaders, key)
			}
		}
		d.Set("http_headers", currentHeaders)
	}

	d.Set("basic_auth_enabled", dataSource.BasicAuth)
	d.Set("basic_auth_username", dataSource.BasicAuthUser)

	return nil
}

func makeDataSource(idStr string, d *schema.ResourceData) (*gapi.DataSource, error) {
	var id int64
	var err error
	if idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	httpHeaders := make(map[string]string)
	for key, value := range d.Get("http_headers").(map[string]interface{}) {
		httpHeaders[key] = fmt.Sprintf("%v", value)
	}

	jd, err := makeJSONData(d)
	if err != nil {
		return nil, err
	}
	sd, err := makeSecureJSONData(d)
	if err != nil {
		return nil, err
	}

	jd, sd = gapi.JSONDataWithHeaders(jd, sd, httpHeaders)

	return &gapi.DataSource{
		ID:             id,
		Name:           d.Get("name").(string),
		Type:           d.Get("type").(string),
		URL:            d.Get("url").(string),
		Access:         d.Get("access_mode").(string),
		Database:       d.Get("database_name").(string),
		User:           d.Get("username").(string),
		IsDefault:      d.Get("is_default").(bool),
		BasicAuth:      d.Get("basic_auth_enabled").(bool),
		BasicAuthUser:  d.Get("basic_auth_username").(string),
		UID:            d.Get("uid").(string),
		JSONData:       jd,
		SecureJSONData: sd,
	}, err
}

func makeJSONData(d *schema.ResourceData) (map[string]interface{}, error) {
	jd := make(map[string]interface{})
	data := d.Get("json_data_encoded")
	if data != "" {
		if err := json.Unmarshal([]byte(data.(string)), &jd); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON data: %s %s", data, err)
		}
	}
	return jd, nil
}

func makeSecureJSONData(d *schema.ResourceData) (map[string]interface{}, error) {
	sjd := make(map[string]interface{})
	data := d.Get("secure_json_data_encoded")
	if data != "" {
		if err := json.Unmarshal([]byte(data.(string)), &sjd); err != nil {
			return nil, fmt.Errorf("failed to unmarshal secure JSON data: %s", err)
		}
	}
	return sjd, nil
}

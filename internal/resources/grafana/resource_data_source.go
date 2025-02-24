package grafana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

func resourceDataSource() *common.Resource {
	schema := &schema.Resource{

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
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

				resp, err := client.Datasources.GetDataSourceByUID(idStr)
				if err != nil {
					return nil, err
				}

				if resp.Payload.ReadOnly {
					return nil, fmt.Errorf("this Grafana data source is read-only. It cannot be imported as a resource. Use the `data_grafana_data_source` data source instead")
				}

				return schema.ImportStatePassthroughContext(ctx, d, meta)
			},
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
			"http_headers": datasourceHTTPHeadersAttribute(),
			"is_default": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to set the data source as default. This should only be `true` to a single data source.",
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					// You can't unset the default data source, because you need one, you have to set another as default instead.
					return oldValue == "true" && newValue == "false" || oldValue == newValue
				},
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
			"json_data_encoded":        datasourceJSONDataAttribute(),
			"secure_json_data_encoded": datasourceSecureJSONDataAttribute(),
			"private_data_source_connect_network_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "(Can only be used with data sources in Grafana Cloud) The ID of the Private Data source Connect network to use with this data source.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_data_source",
		orgResourceIDString("uid"),
		schema,
	).
		WithLister(listerFunctionOrgResource(listDatasources)).
		WithPreferredResourceNameField("name")
}

func datasourceHTTPHeadersAttribute() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeMap,
		Optional:    true,
		Sensitive:   true,
		Description: "Custom HTTP headers",
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
}

func datasourceJSONDataAttribute() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Serialized JSON string containing the json data. This attribute can be used to pass configuration options to the data source. To figure out what options a datasource has available, see its docs or inspect the network data when saving it from the Grafana UI. Note that keys in this map are usually camelCased.",
		ValidateFunc: func(i interface{}, s string) ([]string, []error) {
			if strings.Contains(i.(string), "httpHeaderName") {
				return nil, []error{
					errors.New("httpHeaderName{num} is a reserved key and cannot be used in JSON data. Use the http_headers attribute instead"),
				}
			}
			if strings.Contains(i.(string), "teamHttpHeaders") {
				return nil, []error{
					errors.New("teamHttpHeaders is a reserved key and cannot be used in JSON data. Use the data_source_config_lbac_rules resource instead"),
				}
			}
			return validation.StringIsJSON(i, s)
		},
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
		DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
			if oldValue == "{}" && newValue == "" {
				return true
			}

			newValueUnmarshalled := make(map[string]interface{})
			json.Unmarshal([]byte(newValue), &newValueUnmarshalled)
			pdcNetworkID := d.Get("private_data_source_connect_network_id")
			if pdcNetworkID != "" {
				newValueUnmarshalled["enableSecureSocksProxy"] = true
				newValueUnmarshalled["secureSocksProxyUsername"] = pdcNetworkID
			}
			newValue, _ = structure.FlattenJsonToString(newValueUnmarshalled)

			return common.SuppressEquivalentJSONDiffs(k, oldValue, newValue, d)
		},
	}
}

func datasourceSecureJSONDataAttribute() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "Serialized JSON string containing the secure json data. This attribute can be used to pass secure configuration options to the data source. To figure out what options a datasource has available, see its docs or inspect the network data when saving it from the Grafana UI. Note that keys in this map are usually camelCased.",
		ValidateFunc: func(i interface{}, s string) ([]string, []error) {
			if strings.Contains(i.(string), "httpHeaderValue") {
				return nil, []error{
					errors.New("httpHeaderValue{num} is a reserved key and cannot be used in JSON data. Use the http_headers attribute instead"),
				}
			}
			return validation.StringIsJSON(i, s)
		},
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
	}
}

func listDatasources(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	resp, err := client.Datasources.GetDataSources()
	if err != nil {
		return nil, err
	}

	for _, ds := range resp.Payload {
		if ds.ReadOnly {
			continue
		}
		ids = append(ids, MakeOrgResourceID(orgID, ds.UID))
	}

	return ids, nil
}

// CreateDataSource creates a Grafana datasource
func CreateDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	dataSource, err := stateToDatasource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.Datasources.AddDataSource(dataSource)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, resp.Payload.Datasource.UID))
	return ReadDataSource(ctx, d, meta)
}

// UpdateDataSource updates a Grafana datasource
func UpdateDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	dataSource, err := stateToDatasource(d)
	if err != nil {
		return diag.FromErr(err)
	}
	body := models.UpdateDataSourceCommand{
		Access:          dataSource.Access,
		BasicAuth:       dataSource.BasicAuth,
		BasicAuthUser:   dataSource.BasicAuthUser,
		Database:        dataSource.Database,
		IsDefault:       dataSource.IsDefault,
		JSONData:        dataSource.JSONData,
		Name:            dataSource.Name,
		SecureJSONData:  dataSource.SecureJSONData,
		Type:            dataSource.Type,
		UID:             dataSource.UID,
		URL:             dataSource.URL,
		User:            dataSource.User,
		WithCredentials: dataSource.WithCredentials,
	}
	_, err = client.Datasources.UpdateDataSourceByUID(idStr, &body)

	return diag.FromErr(err)
}

// ReadDataSource reads a Grafana datasource
func ReadDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Datasources.GetDataSourceByUID(idStr)
	if err, shouldReturn := common.CheckReadError("datasource", d, err); shouldReturn {
		return err
	}

	return datasourceToState(d, resp.Payload)
}

// DeleteDataSource deletes a Grafana datasource
func DeleteDataSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	_, err := client.Datasources.DeleteDataSourceByUID(idStr)
	diag, _ := common.CheckReadError("datasource", d, err)
	return diag
}

func datasourceToState(d *schema.ResourceData, dataSource *models.DataSource) diag.Diagnostics {
	d.SetId(MakeOrgResourceID(dataSource.OrgID, dataSource.UID))
	d.Set("access_mode", dataSource.Access)
	d.Set("database_name", dataSource.Database)
	d.Set("is_default", dataSource.IsDefault)
	d.Set("name", dataSource.Name)
	d.Set("type", dataSource.Type)
	d.Set("url", dataSource.URL)
	d.Set("username", dataSource.User)
	d.Set("uid", dataSource.UID)
	d.Set("org_id", strconv.FormatInt(dataSource.OrgID, 10))

	d.Set("basic_auth_enabled", dataSource.BasicAuth)
	d.Set("basic_auth_username", dataSource.BasicAuthUser)

	return datasourceConfigToState(d, dataSource)
}

func datasourceConfigToState(d *schema.ResourceData, dataSource *models.DataSource) diag.Diagnostics {
	gottenJSONData, gottenHeaders := removeHeadersFromJSONData(dataSource.JSONData.(map[string]interface{}))
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
	return nil
}

func stateToDatasource(d *schema.ResourceData) (*models.AddDataSourceCommand, error) {
	jd, sd, err := stateToDatasourceConfig(d)
	if err != nil {
		return nil, err
	}

	return &models.AddDataSourceCommand{
		Name:           d.Get("name").(string),
		Type:           d.Get("type").(string),
		URL:            d.Get("url").(string),
		Access:         models.DsAccess(d.Get("access_mode").(string)),
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

// stateToDatasourceConfig extracts the json data from the config
func stateToDatasourceConfig(d *schema.ResourceData) (map[string]interface{}, map[string]string, error) {
	httpHeaders := make(map[string]string)
	for key, value := range d.Get("http_headers").(map[string]interface{}) {
		httpHeaders[key] = fmt.Sprintf("%v", value)
	}

	jd, err := makeJSONData(d)
	if err != nil {
		return nil, nil, err
	}

	pdcNetworkID := d.Get("private_data_source_connect_network_id")
	if pdcNetworkID != nil {
		if id := pdcNetworkID.(string); id != "" {
			jd["enableSecureSocksProxy"] = true
			jd["secureSocksProxyUsername"] = pdcNetworkID
		}
	}

	sd, err := makeSecureJSONData(d)
	if err != nil {
		return nil, nil, err
	}

	jd, sd = jsonDataWithHeaders(jd, sd, httpHeaders)
	return jd, sd, nil
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

func makeSecureJSONData(d *schema.ResourceData) (map[string]string, error) {
	sjd := make(map[string]string)
	data := d.Get("secure_json_data_encoded")
	if data != "" {
		if err := json.Unmarshal([]byte(data.(string)), &sjd); err != nil {
			return nil, fmt.Errorf("failed to unmarshal secure JSON data: %s", err)
		}
	}
	return sjd, nil
}

func jsonDataWithHeaders(inputJSONData map[string]interface{}, inputSecureJSONData map[string]string, headers map[string]string) (map[string]interface{}, map[string]string) {
	jsonData := make(map[string]interface{})
	for name, value := range inputJSONData {
		jsonData[name] = value
	}

	secureJSONData := make(map[string]string)
	for name, value := range inputSecureJSONData {
		secureJSONData[name] = value
	}

	idx := 1
	for name, value := range headers {
		jsonData[fmt.Sprintf("httpHeaderName%d", idx)] = name
		secureJSONData[fmt.Sprintf("httpHeaderValue%d", idx)] = value
		idx++
	}

	return jsonData, secureJSONData
}

func removeHeadersFromJSONData(input map[string]interface{}) (map[string]interface{}, map[string]string) {
	jsonData := make(map[string]interface{})
	headers := make(map[string]string)

	for dataName, dataValue := range input {
		if strings.HasPrefix(dataName, "httpHeaderName") {
			headerName := dataValue.(string)
			headers[headerName] = "true" // We can't retrieve the headers, so we just set a dummy value
		} else {
			jsonData[dataName] = dataValue
		}
	}
	// for teamhttpheaders, we do not set it in the state and we do not want to return it in the diff
	delete(jsonData, "teamHttpHeaders")

	return jsonData, headers
}

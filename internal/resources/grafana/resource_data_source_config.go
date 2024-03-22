package grafana

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
)

func resourceDataSourceConfig() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/data_source/)

The required arguments for this resource vary depending on the type of data
source selected (via the 'type' argument).
`,

		CreateContext: UpdateDataSourceConfig,
		UpdateContext: UpdateDataSourceConfig,
		ReadContext:   ReadDataSourceConfig,
		DeleteContext: DeleteDataSourceConfig,
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
				Default:     nil,
				ForceNew:    true,
				Description: "Unique identifier. If unset, this will be automatically generated.",
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

// UpdateDataSource updates a Grafana datasource
func UpdateDataSourceConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dataSourceUID, ok := d.Get("uid").(string)
	if !ok {
		return diag.Errorf("UID is not a string")
	}
	client, _ := OAPIClientFromNewOrgResource(meta, d)

	return updateGrafanaDataSource(d, dataSourceUID, client)
}

// ReadDataSource reads a Grafana datasource configuration
func ReadDataSourceConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	var resp interface{ GetPayload() *models.DataSource }
	var err error
	// Support both numerical and UID IDs, so that we can import an existing datasource with either.
	// Following the read, it's normalized to a numerical ID.
	if _, parseErr := strconv.ParseInt(idStr, 10, 64); parseErr == nil {
		resp, err = client.Datasources.GetDataSourceByID(idStr)
	} else {
		resp, err = client.Datasources.GetDataSourceByUID(idStr)
	}

	if err, shouldReturn := common.CheckReadError("datasource", d, err); shouldReturn {
		return err
	}

	return readDatasourceConfig(d, resp.GetPayload())
}

// DeleteDataSource deletes a Grafana datasource
func DeleteDataSourceConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, d)

	d.Set("json_data_encoded", "")

	return updateGrafanaDataSource(d, d.Get("uid").(string), client)
}

func updateGrafanaDataSource(
	d *schema.ResourceData,
	dataSourceUID string,
	client *goapi.GrafanaHTTPAPI,
) diag.Diagnostics {
	// Get the existing datasource
	resp, err := client.Datasources.GetDataSourceByUID(dataSourceUID)
	fetchedDataSource := resp.GetPayload()
	d.SetId(MakeOrgResourceID(fetchedDataSource.OrgID, fetchedDataSource.ID))

	parsedUpdatedJSON, err := makeJSONData(d)
	if err != nil {
		return diag.FromErr(err)
	}
	fetchedDataSource.JSONData = parsedUpdatedJSON

	body := models.UpdateDataSourceCommand{
		Access:          fetchedDataSource.Access,
		BasicAuth:       fetchedDataSource.BasicAuth,
		BasicAuthUser:   fetchedDataSource.BasicAuthUser,
		Database:        fetchedDataSource.Database,
		IsDefault:       fetchedDataSource.IsDefault,
		JSONData:        fetchedDataSource.JSONData,
		Name:            fetchedDataSource.Name,
		Type:            fetchedDataSource.Type,
		UID:             fetchedDataSource.UID,
		URL:             fetchedDataSource.URL,
		User:            fetchedDataSource.User,
		WithCredentials: fetchedDataSource.WithCredentials,
	}
	_, err = client.Datasources.UpdateDataSourceByUID(dataSourceUID, &body)

	return diag.FromErr(err)
}

func readDatasourceConfig(d *schema.ResourceData, dataSource *models.DataSource) diag.Diagnostics {
	d.SetId(MakeOrgResourceID(dataSource.OrgID, dataSource.ID))
	d.Set("uid", dataSource.UID)
	d.Set("org_id", strconv.FormatInt(dataSource.OrgID, 10))

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

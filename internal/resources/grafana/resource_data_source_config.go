package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"

	_ "embed"
)

//go:embed resource_data_source_config.md
var resourceDataSourceConfigDescription string

func resourceDataSourceConfig() *common.Resource {
	schema := &schema.Resource{
		Description: resourceDataSourceConfigDescription,

		CreateContext: UpdateDataSourceConfig,
		UpdateContext: UpdateDataSourceConfig,
		ReadContext:   ReadDataSourceConfig,
		DeleteContext: DeleteDataSourceConfig,
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
			"http_headers":             datasourceHTTPHeadersAttribute(),
			"json_data_encoded":        datasourceJSONDataAttribute(),
			"secure_json_data_encoded": datasourceSecureJSONDataAttribute(),
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_data_source_config",
		orgResourceIDString("uid"),
		schema,
	)
}

func UpdateDataSourceConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	if diag := updateGrafanaDataSourceConfig(d, d.Get("uid").(string), client); diag.HasError() {
		return diag
	}
	return ReadDataSourceConfig(ctx, d, meta)
}

func ReadDataSourceConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Datasources.GetDataSourceByUID(idStr)
	if err, shouldReturn := common.CheckReadError("datasource", d, err); shouldReturn {
		return err
	}
	ds := resp.GetPayload()
	d.Set("uid", ds.UID)
	d.Set("org_id", strconv.FormatInt(ds.OrgID, 10))
	return datasourceConfigToState(d, ds)
}

func DeleteDataSourceConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	d.Set("json_data_encoded", "")
	return updateGrafanaDataSourceConfig(d, idStr, client)
}

func updateGrafanaDataSourceConfig(d *schema.ResourceData, dataSourceUID string, client *goapi.GrafanaHTTPAPI) diag.Diagnostics {
	resp, err := client.Datasources.GetDataSourceByUID(dataSourceUID)
	if err != nil {
		return diag.FromErr(err)
	}
	ds := resp.GetPayload()
	d.SetId(MakeOrgResourceID(ds.OrgID, ds.UID))

	jd, sd, err := stateToDatasourceConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	body := models.UpdateDataSourceCommand{
		Access:          ds.Access,
		BasicAuth:       ds.BasicAuth,
		BasicAuthUser:   ds.BasicAuthUser,
		Database:        ds.Database,
		IsDefault:       ds.IsDefault,
		Name:            ds.Name,
		Type:            ds.Type,
		UID:             ds.UID,
		URL:             ds.URL,
		User:            ds.User,
		WithCredentials: ds.WithCredentials,
		JSONData:        jd,
		SecureJSONData:  sd,
	}
	_, err = client.Datasources.UpdateDataSourceByUID(dataSourceUID, &body)
	return diag.FromErr(err)
}

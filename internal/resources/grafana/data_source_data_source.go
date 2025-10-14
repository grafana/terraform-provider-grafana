package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceDatasource() *common.DataSource {
	schema := &schema.Resource{
		Description: "Get details about a Grafana Datasource querying by either name, uid or ID",
		ReadContext: datasourceDatasourceRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceDataSource().Schema, map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"name", "uid"},
			},
			"uid": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"name", "uid"},
			},
			"secure_json_data_encoded": nil,
			"http_headers":             nil,
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_data_source", schema)
}

func datasourceDatasourceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, d)

	var resp interface{ GetPayload() *models.DataSource }
	var err error

	if name, ok := d.GetOk("name"); ok {
		resp, err = client.Datasources.GetDataSourceByName(name.(string))
	} else if uid, ok := d.GetOk("uid"); ok {
		resp, err = client.Datasources.GetDataSourceByUID(uid.(string))
	} else {
		return diag.Errorf("name or uid must be set")
	}

	if err != nil {
		return diag.FromErr(err)
	}

	if resp == nil {
		return diag.Errorf("unexpected state, API response is nil")
	}

	return datasourceToState(d, resp.GetPayload())
}

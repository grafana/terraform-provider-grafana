package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceDatasource() *schema.Resource {
	return &schema.Resource{
		Description: "Get details about a Grafana Datasource querying by either name, uid or ID",
		ReadContext: datasourceDatasourceRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceDataSource(), map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"id", "name", "uid"},
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"id", "name", "uid"},
			},
			"uid": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"id", "name", "uid"},
			},
			"secure_json_data_encoded": nil,
			"http_headers":             nil,
		}),
	}
}

func datasourceDatasourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, d)

	var resp interface{ GetPayload() *models.DataSource }
	var err error

	if name, ok := d.GetOk("name"); ok {
		resp, err = client.Datasources.GetDataSourceByName(name.(string))
	} else if id, ok := d.GetOk("id"); ok {
		_, idStr := SplitOrgResourceID(id.(string))
		resp, err = client.Datasources.GetDataSourceByID(idStr)
	} else if uid, ok := d.GetOk("uid"); ok {
		resp, err = client.Datasources.GetDataSourceByUID(uid.(string))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	return readDatasource(d, resp.GetPayload())
}

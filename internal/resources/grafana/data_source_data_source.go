package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceDatasource() *schema.Resource {
	return &schema.Resource{
		Description: "Get details about a Grafana Datasource querying by either name, uid or ID",
		ReadContext: datasourceDatasourceRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceDataSource().Schema, map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"id", "name", "uid"},
				Deprecated:   "Use `uid` instead of `id`",
				Description:  "Deprecated: Use `uid` instead of `id`",
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
		if _, parseErr := strconv.ParseInt(idStr, 10, 64); parseErr == nil {
			resp, err = client.Datasources.GetDataSourceByID(idStr) // TODO: Remove on next major release
		} else {
			resp, err = client.Datasources.GetDataSourceByUID(idStr)
		}
	} else if uid, ok := d.GetOk("uid"); ok {
		resp, err = client.Datasources.GetDataSourceByUID(uid.(string))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	return datasourceToState(d, resp.GetPayload())
}

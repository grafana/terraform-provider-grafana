package grafana

import (
	"context"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceDatasource() *schema.Resource {
	return &schema.Resource{
		Description: "Get details about a Grafana Datasource querying by either name, uid or ID",
		ReadContext: datasourceDatasourceRead,
		Schema: common.CloneResourceSchemaForDatasource(ResourceDataSource(), map[string]*schema.Schema{
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
	client, _ := ClientFromNewOrgResource(meta, d)

	var (
		dataSource *gapi.DataSource
		err        error
	)

	if name, ok := d.GetOk("name"); ok {
		id, getIDErr := client.DataSourceIDByName(name.(string))
		if getIDErr != nil {
			return diag.FromErr(getIDErr)
		}
		dataSource, err = client.DataSource(id)
	} else if id, ok := d.GetOk("id"); ok {
		_, idStr := SplitOrgResourceID(id.(string))
		idInt, parseErr := strconv.ParseInt(idStr, 10, 64)
		if parseErr != nil {
			return diag.FromErr(parseErr)
		}
		dataSource, err = client.DataSource(idInt)
	} else if uid, ok := d.GetOk("uid"); ok {
		dataSource, err = client.DataSourceByUID(uid.(string))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	return readDatasource(d, dataSource)
}

package grafana

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceDatasourceCache() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/query_and_resource_caching/)
		`,
		CreateContext: CreateDataSourceCaching,
		UpdateContext: UpdateDataSourceCaching,
		DeleteContext: DeleteDataSourceCaching,
		ReadContext:   ReadDataSourceCaching,
		SchemaVersion: 1,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Unique identifier. If unset, this will be automatically generated.",
			},
		},
	}

}

// CreateDataSource creates a Grafana datasource
func CreateDataSourceCaching(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

}

// UpdateDataSourceCaching updates a Grafana DataSourceCaching
func UpdateDataSourceCaching(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

}

// ReadDataSourceCaching reads a Grafana DataSourceCaching
func ReadDataSourceCaching(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

}

// DeleteDataSourceCaching deletes a Grafana DataSourceCaching
func DeleteDataSourceCaching(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

}

func makeDataSourceCaching(idStr string, d *schema.ResourceData) (*gapi.DataSourceCache, error) {

}

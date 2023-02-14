package grafana

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceLibraryPanels() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for retrieving all library panels.",
		ReadContext: dataSourceLibraryPanelsRead,
		Schema: map[string]*schema.Schema{
			"elements": {
				Type:        schema.TypeList,
				Description: "List of library elements.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: common.CloneResourceSchemaForDatasource(ResourceLibraryPanel(), map[string]*schema.Schema{
						"org_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The Organization ID. If not set, the Org ID defined in the provider block will be used.",
						},
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Name of the library panel.",
						},
						"uid": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The unique identifier (UID) of the library panel.",
						},
					}),
				},
			},
		},
	}
}

func dataSourceLibraryPanelsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	panels, err := client.LibraryPanels()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("grafana_library_panels")
	if err := d.Set("elements", flattenPanels(panels, meta)); err != nil {
		return diag.Errorf("error setting item: %v", err)
	}

	return nil
}

func flattenPanels(panels []gapi.LibraryPanel, meta interface{}) []interface{} {
	libraryPanels := make([]interface{}, len(panels))
	for i, p := range panels {
		libraryPanels[i], _ = flattenLibraryPanel(p, meta)
	}

	return libraryPanels
}

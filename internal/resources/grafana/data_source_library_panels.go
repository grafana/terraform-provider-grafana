package grafana

import (
	"context"

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

	var allDiags diag.Diagnostics
	elements := make([]*schema.ResourceData, len(panels))
	for i, p := range panels {
		resource := &schema.ResourceData{}
		resource.Set("uid", p.UID)

		diags := readLibraryPanel(ctx, resource, meta)
		if diags.HasError() {
			return diags
		}

		allDiags = append(allDiags, diags...)
		elements[i] = resource
	}

	d.Set("elements", elements)

	return allDiags
}

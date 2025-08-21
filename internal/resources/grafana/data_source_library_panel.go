package grafana

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceLibraryPanel() *common.DataSource {
	schema := &schema.Resource{
		Description: "Data source for retrieving a single library panel by name or uid.",
		ReadContext: dataSourceLibraryPanelRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceLibraryPanel().Schema, map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
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
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_library_panel", schema)
}

func dataSourceLibraryPanelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	uid := d.Get("uid").(string)

	if uid == "" {
		// get UID from name if specified
		name := d.Get("name").(string)

		if name == "" {
			return diag.Errorf("either name or uid must be specified")
		}

		resp, err := client.LibraryElements.GetLibraryElementByName(name)
		if err != nil {
			return diag.FromErr(err)
		}
		result := resp.GetPayload().Result
		if len(result) != 1 {
			return diag.Errorf("expected 1 library panel with name %q, got %d", name, len(result))
		}
		uid = result[0].UID
	}

	d.SetId(MakeOrgResourceID(orgID, uid))

	return readLibraryPanel(ctx, d, meta)
}

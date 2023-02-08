package grafana

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceLibraryPanel() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for retrieving a single library panel by name or uid.",
		ReadContext: dataSourceLibraryPanelRead,
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
	}
}

func dataSourceLibraryPanelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromOrgIDAttr(meta, d)
	uid := d.Get("uid").(string)

	// get UID from name if specified
	name := d.Get("name").(string)
	if name != "" {
		panel, err := client.LibraryPanelByName(name)
		if err != nil {
			return diag.FromErr(err)
		}
		uid = panel.UID
	}

	d.SetId(MakeOSSOrgID(orgID, uid))

	return readLibraryPanel(ctx, d, meta)
}

package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceLibraryPanel() *schema.Resource {
	panelSchema := datasourceSchemaFromResourceSchema(libraryPanel.Schema)
	panelSchema["uid"].Optional = true
	panelSchema["name"].Optional = true
	panelSchema["uid"].ExactlyOneOf = []string{"uid", "name"}
	panelSchema["name"].ExactlyOneOf = []string{"uid", "name"}

	return &schema.Resource{
		Description: "Data source for retrieving a single library panel by name or uid.",
		ReadContext: dataSourceLibraryPanelRead,
		Schema:      panelSchema,
	}
}

func dataSourceLibraryPanelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
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

	d.SetId(uid)
	ReadLibraryPanel(ctx, d, meta)

	return nil
}

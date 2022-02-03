package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceLibraryPanel() *schema.Resource {
	probeSchema := datasourceSchemaFromResourceSchema(libraryPanel.Schema)
	probeSchema["uid"].Optional = true
	delete(probeSchema, "name")
	// probeSchema["name"].Optional = true
	probeSchema["uid"].ExactlyOneOf = []string{"uid", "name"}
	// probeSchema["name"].ExactlyOneOf = []string{"uid", "name"}

	return &schema.Resource{
		Description: "Data source for retrieving a single library panel by name or uid.",
		ReadContext: dataSourceLibraryPanelRead,
		Schema:      probeSchema,
	}
	// 	return &schema.Resource{
	// 		Description: `
	// * [Official documentation](https://grafana.com/docs/grafana/latest/panels/panel-library/)
	// * [HTTP API](https://grafana.com/docs/grafana/latest/http_api/library_element/#library-element-api)
	// `,
	// 		ReadContext: dataSourceLibraryPanelRead,
	// 		Schema: map[string]*schema.Schema{
	// 			"uid": {
	// 				Type:         schema.TypeString,
	// 				Optional:     true,
	// 				Computed:     true,
	// 				ExactlyOneOf: []string{"uid", "name"},
	// 				Description: "The unique identifier (UID) of a library panel uniquely identifies library panels between multiple Grafana installs. " +
	// 					"It’s automatically generated unless you specify it during library panel creation." +
	// 					"The UID provides consistent URLs for accessing library panels and when syncing library panels between multiple Grafana installs.",
	// 			},
	// 			"name": {
	// 				Type:         schema.TypeString,
	// 				Optional:     true,
	// 				Computed:     true,
	// 				ExactlyOneOf: []string{"uid", "name"},
	// 				Description:  "Name of the library panel.",
	// 			},
	// 			"panel_id": {
	// 				Type:        schema.TypeInt,
	// 				Computed:    true,
	// 				Description: "The numeric ID of the library panel computed by Grafana.",
	// 			},
	// 			"org_id": {
	// 				Type:        schema.TypeInt,
	// 				Computed:    true,
	// 				Description: "The numeric ID of the library panel computed by Grafana.",
	// 			},
	// 			"folder_id": {
	// 				Type:        schema.TypeInt,
	// 				Computed:    true,
	// 				Description: "ID of the folder where the library panel is stored.",
	// 			},
	// 			"title": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "Title of the library panel.",
	// 			},
	// 			"description": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "Description of the library panel.",
	// 			},
	// 			"type": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "Type of the library panel (eg. text).",
	// 			},
	// 			"model_json": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "The JSON model for the library panel.",
	// 			},
	// 			"version": {
	// 				Type:        schema.TypeInt,
	// 				Computed:    true,
	// 				Description: "Version of the library panel.",
	// 			},
	// 			"folder_name": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "Name of the folder containing the library panel.",
	// 			},
	// 			"folder_uid": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "Unique ID (UID) of the folder containing the library panel.",
	// 			},
	// 			"created": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "Unique ID (UID) of the folder containing the library panel.",
	// 			},
	// 			"updated": {
	// 				Type:        schema.TypeString,
	// 				Computed:    true,
	// 				Description: "Unique ID (UID) of the folder containing the library panel.",
	// 			},
	// 			"dashboard_ids": {
	// 				Type:        schema.TypeList,
	// 				Computed:    true,
	// 				Description: "Numerical IDs of Grafana dashboards containing the library panel.",
	// 				Elem:        &schema.Schema{Type: schema.TypeInt},
	// 			},
	// 		},
	// 	}
}

func dataSourceLibraryPanelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// client := meta.(*client).gapi
	uid := d.Get("uid").(string)
	// var err error

	d.SetId(uid)
	ReadLibraryPanel(ctx, d, meta)

	// get UID from ID if specified
	// var panel *gapi.LibraryPanel
	// name := d.Get("name").(string)
	// if name != "" {
	// 	panel, err = client.LibraryPanelByName(name)
	// } else {
	// 	d.SetId(uid)
	// 	ReadLibraryPanel(ctx, d, meta)
	// }

	// if err != nil {
	// 	return diag.FromErr(err)
	// }

	// connections, err := client.LibraryPanelConnections(uid)
	// if err != nil {
	// 	return diag.FromErr(err)
	// }

	// dashboardIds := []int64{}
	// for _, connection := range *connections {
	// 	dashboardIds = append(dashboardIds, connection.DashboardID)
	// }
	// // return diag.Errorf("%#v", dashboardIds)
	// d.Set("dashboard_ids", dashboardIds)

	return nil
}

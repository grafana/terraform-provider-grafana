package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceLibraryPanel() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages Grafana library panels.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/manage-library-panels/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/library_element/)
`,

		CreateContext: createLibraryPanel,
		ReadContext:   readLibraryPanel,
		UpdateContext: updateLibraryPanel,
		DeleteContext: deleteLibraryPanel,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				Description: "The unique identifier (UID) of a library panel uniquely identifies library panels between multiple Grafana installs. " +
					"It’s automatically generated unless you specify it during library panel creation." +
					"The UID provides consistent URLs for accessing library panels and when syncing library panels between multiple Grafana installs.",
			},
			"panel_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numeric ID of the library panel computed by Grafana.",
			},
			"org_id": orgIDAttribute(),
			"folder_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "ID of the folder where the library panel is stored.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the library panel.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the library panel.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type of the library panel (eg. text).",
			},
			"model_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    normalizeLibraryPanelModelJSON,
				ValidateFunc: validateLibraryPanelModelJSON,
				Description:  "The JSON model for the library panel.",
			},
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Version of the library panel.",
			},
			"folder_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the folder containing the library panel.",
			},
			"folder_uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique ID (UID) of the folder containing the library panel.",
			},
			"created": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the library panel was created.",
			},
			"updated": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the library panel was last modified.",
			},
			"dashboard_ids": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Numerical IDs of Grafana dashboards containing the library panel.",
				Elem:        &schema.Schema{Type: schema.TypeInt},
			},
		},
	}
}

func createLibraryPanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := clientFromNewOrgResource(meta, d)

	panel := makeLibraryPanel(d)
	resp, err := client.NewLibraryPanel(panel)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(makeOrgResourceID(resp.OrgID, resp.UID))
	return readLibraryPanel(ctx, d, meta)
}

func readLibraryPanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := clientFromExistingOrgResource(meta, d.Id())

	panel, err := client.LibraryPanelByUID(uid)
	var diags diag.Diagnostics
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Library Panel %s is in state, but no longer exists in grafana", uid),
				Detail:   fmt.Sprintf("%s will be recreated when you apply", uid),
			})
			d.SetId("")
			return diags
		} else {
			return diag.FromErr(err)
		}
	}

	modelJSONBytes, err := json.Marshal(panel.Model)
	if err != nil {
		return diag.FromErr(err)
	}
	remotePanelJSON, err := unmarshalLibraryPanelModelJSON(string(modelJSONBytes))
	if err != nil {
		return diag.FromErr(err)
	}
	modelJSON := normalizeLibraryPanelModelJSON(remotePanelJSON)

	d.Set("uid", panel.UID)
	d.Set("panel_id", panel.ID)
	d.Set("org_id", strconv.FormatInt(panel.OrgID, 10))
	d.Set("folder_id", panel.Folder)
	d.Set("description", panel.Description)
	d.Set("type", panel.Type)
	d.Set("name", panel.Name)
	d.Set("model_json", modelJSON)
	d.Set("version", panel.Version)
	d.Set("folder_name", panel.Meta.FolderName)
	d.Set("folder_uid", panel.Meta.FolderUID)
	d.Set("created", panel.Meta.Created.String())
	d.Set("updated", panel.Meta.Updated.String())

	connections, err := client.LibraryPanelConnections(uid)
	if err != nil {
		return diag.FromErr(err)
	}

	dashboardIds := make([]int64, 0, len(*connections))
	for _, connection := range *connections {
		dashboardIds = append(dashboardIds, connection.DashboardID)
	}
	d.Set("dashboard_ids", dashboardIds)

	return diags
}

func updateLibraryPanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := clientFromExistingOrgResource(meta, d.Id())
	panel := makeLibraryPanel(d)

	resp, err := client.PatchLibraryPanel(uid, panel)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(makeOrgResourceID(resp.OrgID, resp.UID))
	return readLibraryPanel(ctx, d, meta)
}

func deleteLibraryPanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := clientFromExistingOrgResource(meta, d.Id())
	_, err := client.DeleteLibraryPanel(uid)
	return diag.FromErr(err)
}

func makeLibraryPanel(d *schema.ResourceData) gapi.LibraryPanel {
	modelJSON := d.Get("model_json").(string)
	panelJSON, err := unmarshalLibraryPanelModelJSON(modelJSON)

	panel := gapi.LibraryPanel{
		UID:    d.Get("uid").(string),
		Name:   d.Get("name").(string),
		Folder: int64(d.Get("folder_id").(int)),
		Model:  panelJSON,
	}
	if err != nil {
		return panel
	}
	return panel
}

// unmarshalLibraryPanelModelJSON is a convenience func for unmarshalling
// `model_json` field.
func unmarshalLibraryPanelModelJSON(modelJSON string) (map[string]interface{}, error) {
	unmarshalledJSON := map[string]interface{}{}
	err := json.Unmarshal([]byte(modelJSON), &unmarshalledJSON)
	if err != nil {
		return nil, err
	}
	return unmarshalledJSON, nil
}

// validateLibraryPanelModelJSON is the ValidateFunc for `model_json`. It
// ensures its value is valid JSON.
func validateLibraryPanelModelJSON(model interface{}, k string) ([]string, []error) {
	modelJSON := model.(string)
	if _, err := unmarshalLibraryPanelModelJSON(modelJSON); err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

// normalizeLibraryPanelModelJSON is the StateFunc for the `model_json` field.
func normalizeLibraryPanelModelJSON(config interface{}) string {
	var modelJSON map[string]interface{}
	switch c := config.(type) {
	case map[string]interface{}:
		modelJSON = c
	case string:
		var err error
		modelJSON, err = unmarshalLibraryPanelModelJSON(c)
		if err != nil {
			return c
		}
	}

	// replace nil with empty string in JSON
	// API will always return model JSON with description and type
	if _, ok := modelJSON["description"]; !ok {
		modelJSON["description"] = ""
	}
	if _, ok := modelJSON["type"]; !ok {
		modelJSON["type"] = ""
	}
	j, _ := json.Marshal(modelJSON)
	return string(j)
}

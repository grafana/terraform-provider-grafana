package grafana

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/library_elements"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const libraryPanelKind int64 = 1

func resourceLibraryPanel() *common.Resource {
	schema := &schema.Resource{

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
			"org_id": orgIDAttribute(),
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
			"folder_uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Unique ID (UID) of the folder containing the library panel.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, old = SplitOrgResourceID(old)
					_, new = SplitOrgResourceID(new)
					// In Grafana 11.6.8+, the API returns "general" for the default folder
					// Treat "general" and empty string as equivalent
					if (old == "general" && new == "") || (old == "" && new == "general") {
						return true
					}
					return old == new
				},
				ValidateFunc: folderUIDValidation,
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

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_library_panel",
		orgResourceIDString("uid"),
		schema,
	).WithLister(listerFunctionOrgResource(listLibraryPanels))
}

func listLibraryPanels(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	params := library_elements.NewGetLibraryElementsParams().WithKind(common.Ref(libraryPanelKind))
	resp, err := client.LibraryElements.GetLibraryElements(params)
	if err != nil {
		return nil, err
	}

	for _, panel := range resp.Payload.Result.Elements {
		ids = append(ids, MakeOrgResourceID(orgID, panel.UID))
	}

	return ids, nil
}

func createLibraryPanel(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, d)

	panel := makeLibraryPanel(d)
	resp, err := client.LibraryElements.CreateLibraryElement(&panel)
	if err != nil {
		return diag.FromErr(err)
	}
	createdPanel := resp.Payload.Result
	d.SetId(MakeOrgResourceID(createdPanel.OrgID, createdPanel.UID))
	return readLibraryPanel(ctx, d, meta)
}

func readLibraryPanel(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID, uid := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.LibraryElements.GetLibraryElementByUID(uid)
	if err, shouldReturn := common.CheckReadError("library panel", d, err); shouldReturn {
		return err
	}
	panel := resp.Payload.Result

	modelJSONBytes, err := json.Marshal(panel.Model)
	if err != nil {
		return diag.FromErr(err)
	}
	remotePanelJSON, err := unmarshalLibraryPanelModelJSON(string(modelJSONBytes))
	if err != nil {
		return diag.FromErr(err)
	}
	modelJSON := normalizeLibraryPanelModelJSON(remotePanelJSON)

	d.SetId(MakeOrgResourceID(orgID, uid))
	d.Set("uid", panel.UID)
	d.Set("panel_id", panel.ID)
	d.Set("org_id", strconv.FormatInt(panel.OrgID, 10))
	// In Grafana 11.6.8+, the API returns "general" for panels in the default folder
	// Normalize to empty string to match user config when folder_uid is not set
	folderUID := panel.Meta.FolderUID
	if folderUID == "general" && d.Get("folder_uid").(string) == "" {
		folderUID = ""
	}
	d.Set("folder_uid", folderUID)
	d.Set("description", panel.Description)
	d.Set("type", panel.Type)
	d.Set("name", panel.Name)
	d.Set("model_json", modelJSON)
	d.Set("version", panel.Version)
	d.Set("folder_name", panel.Meta.FolderName)
	d.Set("created", panel.Meta.Created.String())
	d.Set("updated", panel.Meta.Updated.String())

	connResp, err := client.LibraryElements.GetLibraryElementConnections(uid)
	if err != nil {
		return diag.FromErr(err)
	}
	connections := connResp.Payload.Result

	dashboardIds := make([]int64, 0, len(connections))
	for _, connection := range connections {
		dashboardIds = append(dashboardIds, connection.ConnectionID)
	}
	d.Set("dashboard_ids", dashboardIds)

	return nil
}

func updateLibraryPanel(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())

	modelJSON := d.Get("model_json").(string)
	panelJSON, _ := unmarshalLibraryPanelModelJSON(modelJSON)

	body := models.PatchLibraryElementCommand{
		Name:    d.Get("name").(string),
		Model:   panelJSON,
		Kind:    libraryPanelKind,
		Version: int64(d.Get("version").(int)),
	}
	_, body.FolderUID = SplitOrgResourceID(d.Get("folder_uid").(string))

	resp, err := client.LibraryElements.UpdateLibraryElement(uid, &body)
	if err != nil {
		return diag.FromErr(err)
	}
	updatedPanel := resp.Payload.Result
	d.SetId(MakeOrgResourceID(updatedPanel.OrgID, updatedPanel.UID))
	return readLibraryPanel(ctx, d, meta)
}

func deleteLibraryPanel(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	_, err := client.LibraryElements.DeleteLibraryElementByUID(uid)
	diag, _ := common.CheckReadError("library panel", d, err)
	return diag
}

func makeLibraryPanel(d *schema.ResourceData) models.CreateLibraryElementCommand {
	modelJSON := d.Get("model_json").(string)
	panelJSON, _ := unmarshalLibraryPanelModelJSON(modelJSON)

	panel := models.CreateLibraryElementCommand{
		UID:   d.Get("uid").(string),
		Name:  d.Get("name").(string),
		Model: panelJSON,
		Kind:  libraryPanelKind,
	}
	_, panel.FolderUID = SplitOrgResourceID(d.Get("folder_uid").(string))

	return panel
}

// unmarshalLibraryPanelModelJSON is a convenience func for unmarshalling
// `model_json` field.
func unmarshalLibraryPanelModelJSON(modelJSON string) (map[string]any, error) {
	unmarshalledJSON := map[string]any{}
	err := json.Unmarshal([]byte(modelJSON), &unmarshalledJSON)
	if err != nil {
		return nil, err
	}
	return unmarshalledJSON, nil
}

// validateLibraryPanelModelJSON is the ValidateFunc for `model_json`. It
// ensures its value is valid JSON.
func validateLibraryPanelModelJSON(model any, k string) ([]string, []error) {
	modelJSON := model.(string)
	if _, err := unmarshalLibraryPanelModelJSON(modelJSON); err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

// normalizeLibraryPanelModelJSON is the StateFunc for the `model_json` field.
func normalizeLibraryPanelModelJSON(config any) string {
	var modelJSON map[string]any
	switch c := config.(type) {
	case map[string]any:
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

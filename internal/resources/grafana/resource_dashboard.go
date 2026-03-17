package grafana

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	StoreDashboardSHA256 bool
)

// resourceDashboardModel is the Terraform Plugin Framework model for grafana_dashboard.
type resourceDashboardModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	UID         types.String `tfsdk:"uid"`
	DashboardID types.Int64  `tfsdk:"dashboard_id"`
	URL         types.String `tfsdk:"url"`
	Version     types.Int64  `tfsdk:"version"`
	Folder      types.String `tfsdk:"folder"`
	ConfigJSON  types.String `tfsdk:"config_json"`
	Overwrite   types.Bool   `tfsdk:"overwrite"`
	Message     types.String `tfsdk:"message"`
}

var (
	_ resource.Resource                = &dashboardResource{}
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
	_ resource.ResourceWithModifyPlan  = &dashboardResource{}

	resourceDashboardName = "grafana_dashboard"
	resourceDashboardID   = orgResourceIDString("uid")
)

type dashboardResource struct {
	basePluginFrameworkResource
	commonClient *common.Client
}

func makeResourceDashboard() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceDashboardName,
		resourceDashboardID,
		&dashboardResource{},
	).WithLister(listerFunctionOrgResource(listDashboards))
}

// folderDashboardPlanModifier suppresses diff when folder values are equivalent
// (e.g. "" vs "0", or same folder UID after stripping org prefix).
type folderDashboardPlanModifier struct{}

func (folderDashboardPlanModifier) Description(_ context.Context) string {
	return "Suppresses diff when folder is equivalent (e.g. empty vs 0, or same UID)."
}

func (m folderDashboardPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m folderDashboardPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}
	_, oldFolder := SplitOrgResourceID(req.StateValue.ValueString())
	_, newFolder := SplitOrgResourceID(req.PlanValue.ValueString())
	equivalent := (oldFolder == "0" && newFolder == "") || (oldFolder == "" && newFolder == "0") || oldFolder == newFolder
	if equivalent {
		resp.PlanValue = req.StateValue
	}
}

// jsonStringValidator validates that a string is valid JSON.
type jsonStringValidator struct{}

func (jsonStringValidator) Description(_ context.Context) string {
	return "value must be valid JSON"
}

func (v jsonStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if s == "" {
		return
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, v.Description(ctx), err.Error())
	}
}

func (r *dashboardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.basePluginFrameworkResource.Configure(ctx, req, resp)
	if req.ProviderData != nil {
		if client, ok := req.ProviderData.(*common.Client); ok {
			r.commonClient = client
		}
	}
}

func (r *dashboardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDashboardName
}

func (r *dashboardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = frameworkSchema.Schema{
		MarkdownDescription: `
Manages Grafana dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,
		Attributes: map[string]frameworkSchema.Attribute{
			"id": frameworkSchema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"uid": frameworkSchema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of a dashboard. This is used to construct its URL. It's automatically generated if not provided when creating a dashboard. The uid allows having consistent URLs for accessing dashboards and when syncing dashboards between multiple Grafana installs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_id": frameworkSchema.Int64Attribute{
				Computed:    true,
				Description: "The numeric ID of the dashboard computed by Grafana.",
			},
			"url": frameworkSchema.StringAttribute{
				Computed:    true,
				Description: "The full URL of the dashboard.",
			},
			"version": frameworkSchema.Int64Attribute{
				Computed:    true,
				Description: "Whenever you save a version of your dashboard, a copy of that version is saved so that previous versions of your dashboard are not lost.",
			},
			"folder": frameworkSchema.StringAttribute{
				Optional:    true,
				Description: "The id or UID of the folder to save the dashboard in.",
				PlanModifiers: []planmodifier.String{
					folderDashboardPlanModifier{},
				},
			},
			"config_json": frameworkSchema.StringAttribute{
				Required:    true,
				Description: "The complete dashboard model JSON.",
				Validators:  []validator.String{jsonStringValidator{}},
			},
			"overwrite": frameworkSchema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set to true if you want to overwrite existing dashboard with newer version, same dashboard title in folder or same dashboard uid.",
			},
			"message": frameworkSchema.StringAttribute{
				Optional:    true,
				Description: "Set a commit message for the version history.",
			},
		},
	}
}

func (r *dashboardResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Force replacement when config_json UID changes (matches SDK CustomizeDiff behavior)
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state resourceDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No state during create
	if state.ConfigJSON.ValueString() == "" && state.ID.ValueString() == "" {
		return
	}

	oldUID := extractUID(state.ConfigJSON.ValueString())
	newUID := extractUID(plan.ConfigJSON.ValueString())
	if oldUID != "" && newUID != "" && oldUID != newUID {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("config_json"))
	}
}

func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	dashboard, err := makeDashboardFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build dashboard", err.Error())
		return
	}

	if r.commonClient != nil {
		r.commonClient.LockDashboard()
		defer r.commonClient.UnlockDashboard()
	}

	apiResp, err := client.Dashboards.PostDashboard(&dashboard)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create dashboard", err.Error())
		return
	}

	// Read back to get computed values
	readData, diags := r.readDashboard(ctx, client, orgID, *apiResp.Payload.UID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, split, err := r.clientFromExistingOrgResource(resourceDashboardID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid resource ID", "Resource ID has no parts")
		return
	}
	uid := fmt.Sprintf("%v", split[0])

	readData, diags := r.readDashboard(ctx, client, orgID, uid, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state resourceDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, split, err := r.clientFromExistingOrgResource(resourceDashboardID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid resource ID", "Resource ID has no parts")
		return
	}

	dashboard, err := makeDashboardFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build dashboard", err.Error())
		return
	}
	dashboard.Dashboard.(map[string]any)["id"] = int(state.DashboardID.ValueInt64())
	dashboard.Overwrite = true

	if r.commonClient != nil {
		r.commonClient.LockDashboard()
		defer r.commonClient.UnlockDashboard()
	}

	apiResp, err := client.Dashboards.PostDashboard(&dashboard)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update dashboard", err.Error())
		return
	}

	readData, diags := r.readDashboard(ctx, client, orgID, *apiResp.Payload.UID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceDashboardID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid resource ID", "Resource ID has no parts")
		return
	}
	uid := fmt.Sprintf("%v", split[0])

	if r.commonClient != nil {
		r.commonClient.LockDashboard()
		defer r.commonClient.UnlockDashboard()
	}

	_, err = client.Dashboards.DeleteDashboardByUID(uid)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete dashboard", err.Error())
	}
}

func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID can be "uid" or "orgID:uid"
	client, orgID, split, err := r.clientFromExistingOrgResource(resourceDashboardID, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse import ID", err.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be 'uid' or 'orgID:uid'")
		return
	}
	uid := fmt.Sprintf("%v", split[0])

	prior := &resourceDashboardModel{ConfigJSON: types.StringValue("")}
	readData, diags := r.readDashboard(ctx, client, orgID, uid, prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Dashboard not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func listDashboards(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	return listDashboardOrFolder(client, orgID, "dash-db")
}

func listDashboardOrFolder(client *goapi.GrafanaHTTPAPI, orgID int64, searchType string) ([]string, error) {
	uids := []string{}
	resp, err := client.Search.Search(search.NewSearchParams().WithType(common.Ref(searchType)))
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Payload {
		uids = append(uids, MakeOrgResourceID(orgID, item.UID))
	}

	return uids, nil
}

func makeDashboardFromModel(data *resourceDashboardModel) (models.SaveDashboardCommand, error) {
	_, folderID := SplitOrgResourceID(data.Folder.ValueString())
	dashboard := models.SaveDashboardCommand{
		Overwrite: data.Overwrite.ValueBool(),
		Message:   data.Message.ValueString(),
		FolderUID: folderID,
	}

	configJSON := data.ConfigJSON.ValueString()
	dashboardJSON, err := UnmarshalDashboardConfigJSON(configJSON)
	if err != nil {
		return dashboard, err
	}
	delete(dashboardJSON, "id")
	dashboard.Dashboard = dashboardJSON
	return dashboard, nil
}

func (r *dashboardResource) readDashboard(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64, uid string, prior *resourceDashboardModel) (*resourceDashboardModel, fwdiag.Diagnostics) {
	var diags fwdiag.Diagnostics

	apiResp, err := client.Dashboards.GetDashboardByUID(uid)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Failed to read dashboard", err.Error())
		return nil, diags
	}

	dashboard := apiResp.Payload
	model := dashboard.Dashboard.(map[string]any)

	configJSONBytes, err := json.Marshal(dashboard.Dashboard)
	if err != nil {
		diags.AddError("Failed to marshal dashboard", err.Error())
		return nil, diags
	}
	remoteDashJSON, err := UnmarshalDashboardConfigJSON(string(configJSONBytes))
	if err != nil {
		diags.AddError("Failed to parse dashboard JSON", err.Error())
		return nil, diags
	}

	configJSON := prior.ConfigJSON.ValueString()
	if configJSON != "" && !common.SHA256Regexp.MatchString(configJSON) {
		configuredDashJSON, err := UnmarshalDashboardConfigJSON(configJSON)
		if err != nil {
			diags.AddError("Failed to parse configured dashboard JSON", err.Error())
			return nil, diags
		}
		if _, ok := configuredDashJSON["uid"].(string); !ok {
			delete(remoteDashJSON, "uid")
		}
	}
	configJSON = NormalizeDashboardConfigJSON(remoteDashJSON)

	urlStr := ""
	if r.commonClient != nil {
		urlStr = r.commonClient.GrafanaSubpath(dashboard.Meta.URL)
	}

	data := &resourceDashboardModel{
		ID:          types.StringValue(MakeOrgResourceID(orgID, uid)),
		OrgID:       types.StringValue(strconv.FormatInt(orgID, 10)),
		UID:         types.StringValue(model["uid"].(string)),
		DashboardID: types.Int64Value(int64(model["id"].(float64))),
		URL:         types.StringValue(urlStr),
		Version:     types.Int64Value(int64(model["version"].(float64))),
		Folder:      types.StringValue(dashboard.Meta.FolderUID),
		ConfigJSON:  types.StringValue(configJSON),
		Overwrite:   prior.Overwrite,
		Message:     prior.Message,
	}
	return data, diags
}

// UnmarshalDashboardConfigJSON is a convenience func for unmarshalling
// `config_json` field.
func UnmarshalDashboardConfigJSON(configJSON string) (map[string]any, error) {
	dashboardJSON := map[string]any{}
	err := json.Unmarshal([]byte(configJSON), &dashboardJSON)
	if err != nil {
		return nil, err
	}
	return dashboardJSON, nil
}

// NormalizeDashboardConfigJSON normalizes the dashboard JSON for state storage.
//
// It removes the following fields:
//
//   - `id`:      an auto-incrementing ID Grafana assigns to dashboards upon
//     creation. We cannot know this before creation and therefore it cannot
//     be managed in code.
//   - `version`: is incremented by Grafana each time a dashboard changes.
func NormalizeDashboardConfigJSON(config any) string {
	var dashboardJSON map[string]any
	switch c := config.(type) {
	case map[string]any:
		dashboardJSON = c
	case string:
		var err error
		dashboardJSON, err = UnmarshalDashboardConfigJSON(c)
		if err != nil {
			return c
		}
	}

	delete(dashboardJSON, "id")
	delete(dashboardJSON, "version")

	// similarly to uid removal above, remove any attributes panels[].libraryPanel.*
	// from the dashboard JSON other than "name" or "uid".
	// Grafana will populate all other libraryPanel attributes, so delete them to avoid diff.
	if panels, ok := dashboardJSON["panels"].([]any); ok {
		for _, panel := range panels {
			panelMap := panel.(map[string]any)
			delete(panelMap, "id")
			if libraryPanel, ok := panelMap["libraryPanel"].(map[string]any); ok {
				for k := range libraryPanel {
					if k != "name" && k != "uid" {
						delete(libraryPanel, k)
					}
				}
			}
		}
	}

	j, _ := json.Marshal(dashboardJSON)

	if StoreDashboardSHA256 {
		configHash := sha256.Sum256(j)
		return fmt.Sprintf("%x", configHash[:])
	} else {
		return string(j)
	}
}

func extractUID(jsonStr string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return ""
	}
	if uid, ok := parsed["uid"].(string); ok {
		return uid
	}
	return ""
}

package grafana

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	StoreDashboardSHA256 bool
)

// dashboardJSONType is a custom string type for config_json that provides
// semantic equality via NormalizeDashboardConfigJSON. This allows the provider
// to return a normalized JSON string from the API while the plan/state stores
// the user's original input, suppressing false diffs due to whitespace, field
// ordering, or stripped fields (id, version, panel ids).
type dashboardJSONType struct{ basetypes.StringType }

func (t dashboardJSONType) String() string { return "dashboardJSONType" }

func (t dashboardJSONType) ValueType(_ context.Context) attr.Value {
	return dashboardJSONValue{}
}

func (t dashboardJSONType) Equal(o attr.Type) bool {
	other, ok := o.(dashboardJSONType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t dashboardJSONType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return dashboardJSONValue{StringValue: in}, nil
}

func (t dashboardJSONType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrVal, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	sv, ok := attrVal.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrVal)
	}
	val, diags := t.ValueFromString(ctx, sv)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting to dashboardJSONValue: %v", diags)
	}
	return val, nil
}

// dashboardJSONValue wraps basetypes.StringValue and overrides semantic
// equality so that two config_json values are considered equal when they
// produce the same output from NormalizeDashboardConfigJSON (i.e. same
// compact JSON with id/version/panel-id fields removed).
type dashboardJSONValue struct{ basetypes.StringValue }

func (v dashboardJSONValue) Type(_ context.Context) attr.Type { return dashboardJSONType{} }

func (v dashboardJSONValue) Equal(o attr.Value) bool {
	other, ok := o.(dashboardJSONValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals returns true when both values represent the same
// dashboard after normalization. When this returns true the Plugin Framework
// preserves the prior state value (the user's raw input) rather than replacing
// it with the provider-computed normalized value.
func (v dashboardJSONValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newVal, ok := newValuable.(dashboardJSONValue)
	if !ok {
		return false, diags
	}
	return NormalizeDashboardConfigJSON(v.ValueString()) == NormalizeDashboardConfigJSON(newVal.ValueString()), diags
}

// folderValueType is a custom string type for the folder attribute that provides
// semantic equality by normalizing org-prefixed IDs (e.g. "1:uid" vs "uid").
// This lets users reference grafana_folder.xxx.id (org-prefixed) or
// grafana_folder.xxx.uid (plain UID) without causing perpetual diffs.
type folderValueType struct{ basetypes.StringType }

func (t folderValueType) String() string { return "folderValueType" }

func (t folderValueType) ValueType(_ context.Context) attr.Value { return folderValue{} }

func (t folderValueType) Equal(o attr.Type) bool {
	other, ok := o.(folderValueType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t folderValueType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return folderValue{StringValue: in}, nil
}

func (t folderValueType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrVal, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	sv, ok := attrVal.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrVal)
	}
	val, diags := t.ValueFromString(ctx, sv)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting to folderValue: %v", diags)
	}
	return val, nil
}

// folderValue wraps basetypes.StringValue and provides semantic equality that
// treats org-prefixed folder IDs (e.g. "1:my-folder") as equivalent to plain
// UIDs ("my-folder"), and treats "" and "0" as equivalent (both are the General
// folder). This prevents perpetual diffs when users switch between
// grafana_folder.xxx.id and grafana_folder.xxx.uid references.
type folderValue struct{ basetypes.StringValue }

func (v folderValue) Type(_ context.Context) attr.Type { return folderValueType{} }

func (v folderValue) Equal(o attr.Value) bool {
	other, ok := o.(folderValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v folderValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newVal, ok := newValuable.(folderValue)
	if !ok {
		return false, diags
	}
	_, oldUID := SplitOrgResourceID(v.ValueString())
	_, newUID := SplitOrgResourceID(newVal.ValueString())
	// Normalize "" and "0" as equivalent (both represent the General folder).
	if oldUID == "0" {
		oldUID = ""
	}
	if newUID == "0" {
		newUID = ""
	}
	return oldUID == newUID, diags
}

// resourceDashboardModel is the Terraform Plugin Framework model for grafana_dashboard.
type resourceDashboardModel struct {
	ID          types.String       `tfsdk:"id"`
	OrgID       types.String       `tfsdk:"org_id"`
	UID         types.String       `tfsdk:"uid"`
	DashboardID types.Int64        `tfsdk:"dashboard_id"`
	URL         types.String       `tfsdk:"url"`
	Version     types.Int64        `tfsdk:"version"`
	Folder      folderValue        `tfsdk:"folder"`
	ConfigJSON  dashboardJSONValue `tfsdk:"config_json"`
	Overwrite   types.Bool         `tfsdk:"overwrite"`
	Message     types.String       `tfsdk:"message"`
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

func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDashboardName
}

func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				CustomType:  folderValueType{},
				Description: "The id or UID of the folder to save the dashboard in.",
			},
			"config_json": frameworkSchema.StringAttribute{
				Optional:    true,
				Computed:    true,
				CustomType:  dashboardJSONType{},
				Description: "The complete dashboard model JSON.",
				Validators:  []validator.String{jsonStringValidator{}},
			},
			"overwrite": frameworkSchema.BoolAttribute{
				Optional:    true,
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
	// No state during create - return immediately to avoid req.State.Get which fails when
	// Framework tries to convert null state to resourceDashboardModel
	if req.State.Raw.IsNull() || !req.State.Raw.IsKnown() {
		return
	}
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state resourceDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldUID := extractUID(state.ConfigJSON.ValueString())
	newUID := extractUID(plan.ConfigJSON.ValueString())
	if oldUID != "" && newUID != "" && oldUID != newUID {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("config_json"))
	}
	// When uid is being added to config_json (transitioning from Grafana-generated to
	// explicit), the uid and id attributes must be marked unknown so the plan allows the
	// new value. Without this, UseStateForUnknown would lock in the old generated uid.
	if oldUID == "" && newUID != "" {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("uid"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
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

	prior := &resourceDashboardModel{ConfigJSON: dashboardJSONValue{StringValue: types.StringValue("")}}
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

func (r *dashboardResource) readDashboard(_ context.Context, client *goapi.GrafanaHTTPAPI, orgID int64, uid string, prior *resourceDashboardModel) (*resourceDashboardModel, diag.Diagnostics) {
	var diags diag.Diagnostics

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

	// Store the folder UID from the API. The folderValue custom type uses
	// StringSemanticEquals to suppress diffs between "orgID:uid" and "uid",
	// so the prior state value (e.g. "1:my-folder") is preserved when it refers
	// to the same folder as the API-returned plain UID ("my-folder").
	var folderVal folderValue
	switch {
	case dashboard.Meta.FolderUID != "":
		folderVal = folderValue{StringValue: types.StringValue(dashboard.Meta.FolderUID)}
	case prior.Folder.IsNull() || prior.Folder.IsUnknown():
		folderVal = folderValue{StringValue: types.StringNull()}
	default:
		// Preserve prior "" or "0" for the General folder.
		folderVal = prior.Folder
	}

	// Preserve overwrite from prior state (null if user did not set it).
	overwrite := prior.Overwrite

	data := &resourceDashboardModel{
		ID:          types.StringValue(MakeOrgResourceID(orgID, uid)),
		OrgID:       types.StringValue(strconv.FormatInt(orgID, 10)),
		UID:         types.StringValue(model["uid"].(string)),
		DashboardID: types.Int64Value(int64(model["id"].(float64))),
		URL:         types.StringValue(urlStr),
		Version:     types.Int64Value(int64(model["version"].(float64))),
		Folder:      folderVal,
		ConfigJSON:  dashboardJSONValue{StringValue: types.StringValue(configJSON)},
		Overwrite:   overwrite,
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

package grafana

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/mod/semver"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/dashboards"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var (
	StoreDashboardSHA256 bool

	_ resource.Resource                = &dashboardResource{}
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
	_ resource.ResourceWithModifyPlan  = &dashboardResource{}

	resourceDashboardName = "grafana_dashboard"
	resourceDashboardID   = orgResourceIDString("uid")
)

func makeResourceDashboard() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceDashboardName,
		resourceDashboardID,
		&dashboardResource{},
	).WithLister(listerFunctionOrgResource(listDashboards))
}

type dashboardResource struct {
	basePluginFrameworkResource
}

type dashboardModel struct {
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

func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDashboardName
}

func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Manages Grafana dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [HTTP API (legacy API, recommended for Grafana 12 or earlier)](https://grafana.com/docs/grafana/v11.6/developers/http_api/dashboard/)
* [HTTP API (new Kubernetes-style API, recommended for Grafana 13 and later)](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"uid": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of a dashboard. This is used to construct its URL. It's automatically generated if not provided when creating a dashboard. The uid allows having consistent URLs for accessing dashboards and when syncing dashboards between multiple Grafana installs. ",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The numeric ID of the dashboard computed by Grafana.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The full URL of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "Whenever you save a version of your dashboard, a copy of that version is saved so that previous versions of your dashboard are not lost.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"folder": schema.StringAttribute{
				Optional:    true,
				Description: "The id or UID of the folder to save the dashboard in.",
				PlanModifiers: []planmodifier.String{
					dashboardFolderPlanModifier{},
				},
			},
			"config_json": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					jsonStringValidator{},
				},
				Description: `The complete dashboard model JSON.

Starting with Grafana v13, use the resource corresponding to your dashboard's API version for Kubernetes-style dashboards.

If you decide to use this legacy resource with a Kubernetes-style dashboard definition:
- In Grafana v12, provide the "spec" field of the dashboard definition.
- In Grafana v13 and later, provide the full Kubernetes-style dashboard JSON (including "apiVersion", "kind", "metadata", and "spec").
`,
			},
			"overwrite": schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true if you want to overwrite existing dashboard with newer version, same dashboard title in folder or same dashboard uid.",
			},
			"message": schema.StringAttribute{
				Optional:    true,
				Description: "Set a commit message for the version history.",
			},
		},
	}
}

func (r *dashboardResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// On destroy, nothing to do.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan dashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize config_json in the plan.
	if !plan.ConfigJSON.IsNull() && !plan.ConfigJSON.IsUnknown() {
		normalized := NormalizeDashboardConfigJSON(plan.ConfigJSON.ValueString())
		plan.ConfigJSON = types.StringValue(normalized)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Require replace when the UID changes (equivalent to CustomizeDiff ForceNew).
	if req.State.Raw.IsNull() {
		return
	}
	var state dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ConfigJSON.IsNull() || plan.ConfigJSON.IsUnknown() ||
		state.ConfigJSON.IsNull() || state.ConfigJSON.IsUnknown() {
		return
	}

	oldUID := extractUID(state.ConfigJSON.ValueString())
	newUID := extractUID(plan.ConfigJSON.ValueString())
	// Only force replacement when both sides have an explicit UID and they differ.
	if oldUID != "" && newUID != "" && oldUID != newUID {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("config_json"))
	}
	// Transitioning from Grafana-assigned UID to an explicit one: no destroy needed,
	// but uid and id are not yet known (they will be set after apply).
	if oldUID == "" && newUID != "" {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("uid"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
	}
}

func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	dashboard, diags := makeDashboardCommand(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if dashboardJSON, ok := dashboard.Dashboard.(map[string]any); ok && isKubernetesStyleDashboard(dashboardJSON) {
		health, err := client.Health.GetHealth(nil)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get Grafana health", err.Error())
			return
		}

		v := health.Payload.Version
		if !strings.HasPrefix(v, "v") {
			v = "v" + v
		}

		if semver.Major(v) == "v12" {
			resp.Diagnostics.AddError("Unsupported Grafana version", "Grafana version 12 doesn't accept k8s-style json. You have to send only the spec")
			return
		}
	}

	var apiResp *dashboards.PostDashboardOK
	var apiErr error
	r.commonClient.WithDashboardLock(func() {
		apiResp, apiErr = client.Dashboards.PostDashboard(&dashboard)
	})
	if apiErr != nil {
		resp.Diagnostics.AddError("Failed to create dashboard", apiErr.Error())
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, *apiResp.Payload.UID))

	readData, diags := r.read(ctx, data.ID.ValueString(), data.ConfigJSON.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Dashboard not found after create", "")
		return
	}
	readData.Overwrite = data.Overwrite
	readData.Message = data.Message
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, data.ID.ValueString(), data.ConfigJSON.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	readData.Overwrite = data.Overwrite
	readData.Message = data.Message
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	dashboard, diags := makeDashboardCommand(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if dashboardJSON, ok := dashboard.Dashboard.(map[string]any); ok && !isKubernetesStyleDashboard(dashboardJSON) {
		dashboardJSON["id"] = state.DashboardID.ValueInt64()
	}
	dashboard.Overwrite = true

	var apiResp *dashboards.PostDashboardOK
	var apiErr error
	r.commonClient.WithDashboardLock(func() {
		apiResp, apiErr = client.Dashboards.PostDashboard(&dashboard)
	})
	if apiErr != nil {
		resp.Diagnostics.AddError("Failed to update dashboard", apiErr.Error())
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, *apiResp.Payload.UID))

	readData, diags := r.read(ctx, data.ID.ValueString(), data.ConfigJSON.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Dashboard not found after update", "")
		return
	}
	readData.Overwrite = data.Overwrite
	readData.Message = data.Message
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceDashboardID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	uid := split[0].(string)

	var deleteErr error
	r.commonClient.WithDashboardLock(func() {
		_, deleteErr = client.Dashboards.DeleteDashboardByUID(uid)
	})
	if deleteErr != nil && !common.IsNotFoundError(deleteErr) {
		resp.Diagnostics.AddError("Failed to delete dashboard", deleteErr.Error())
	}
}

func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID, "")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", fmt.Sprintf("Dashboard %q not found", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *dashboardResource) read(ctx context.Context, id string, currentConfigJSON string) (*dashboardModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, orgID, uid, err := r.clientFromExistingOrgResourceWithOrgID(resourceDashboardID, id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, diags
	}

	preferredAPIVersion := preferredDashboardAPIVersion(currentConfigJSON)

	apiResp, err := readDashboardByUID(ctx, client, uid, preferredAPIVersion)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		diags.AddError("Failed to read dashboard", err.Error())
		return nil, diags
	}

	dashboard := apiResp.Payload
	model := dashboard.Dashboard.(map[string]any)

	configJSONBytes, err := json.Marshal(dashboard.Dashboard)
	if err != nil {
		diags.AddError("Failed to marshal dashboard JSON", err.Error())
		return nil, diags
	}
	remoteDashJSON, err := UnmarshalDashboardConfigJSON(string(configJSONBytes))
	if err != nil {
		diags.AddError("Failed to unmarshal dashboard JSON", err.Error())
		return nil, diags
	}

	configJSON, err := normalizeDashboardConfigJSONForState(currentConfigJSON, remoteDashJSON)
	if err != nil {
		diags.AddError("Failed to normalize dashboard config JSON", err.Error())
		return nil, diags
	}

	data := &dashboardModel{
		ID:          types.StringValue(MakeOrgResourceID(orgID, uid)),
		OrgID:       types.StringValue(fmt.Sprintf("%d", orgID)),
		UID:         types.StringValue(model["uid"].(string)),
		DashboardID: types.Int64Value(int64(model["id"].(float64))),
		Version:     types.Int64Value(int64(model["version"].(float64))),
		URL:         types.StringValue(r.commonClient.GrafanaSubpath(dashboard.Meta.URL)),
		Folder:      dashboardFolderValue(dashboard.Meta.FolderUID),
		ConfigJSON:  types.StringValue(configJSON),
	}

	return data, diags
}

// clientFromExistingOrgResourceWithOrgID is like clientFromExistingOrgResource but also returns orgID.
func (r *dashboardResource) clientFromExistingOrgResourceWithOrgID(resourceID *common.ResourceID, id string) (*goapi.GrafanaHTTPAPI, int64, string, error) {
	client, orgID, split, err := r.clientFromExistingOrgResource(resourceID, id)
	if err != nil {
		return nil, 0, "", err
	}
	return client, orgID, split[0].(string), nil
}

func makeDashboardCommand(data dashboardModel) (models.SaveDashboardCommand, diag.Diagnostics) {
	var diags diag.Diagnostics
	_, folderUID := SplitOrgResourceID(data.Folder.ValueString())

	dashboard := models.SaveDashboardCommand{
		Overwrite: data.Overwrite.ValueBool(),
		Message:   data.Message.ValueString(),
		FolderUID: folderUID,
	}

	configJSON := data.ConfigJSON.ValueString()
	dashboardJSON, err := UnmarshalDashboardConfigJSON(configJSON)
	if err != nil {
		diags.AddError("Invalid config_json", err.Error())
		return dashboard, diags
	}
	delete(dashboardJSON, "id")
	dashboard.Dashboard = dashboardJSON
	return dashboard, diags
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

func isKubernetesStyleDashboard(dashboardJSON map[string]any) bool {
	_, hasAPIVersion := dashboardJSON["apiVersion"].(string)
	_, hasKind := dashboardJSON["kind"].(string)
	_, hasSpec := dashboardJSON["spec"].(map[string]any)
	return hasAPIVersion && hasKind && hasSpec
}

func preferredDashboardAPIVersion(configJSON string) string {
	if configJSON == "" || common.SHA256Regexp.MatchString(configJSON) {
		return ""
	}

	dashboardJSON, err := UnmarshalDashboardConfigJSON(configJSON)
	if err != nil || !isKubernetesStyleDashboard(dashboardJSON) {
		return ""
	}

	apiVersion, _ := dashboardJSON["apiVersion"].(string)
	return extractDashboardAPIVersion(apiVersion)
}

func extractDashboardAPIVersion(apiVersion string) string {
	if apiVersion == "" {
		return ""
	}

	if _, version, ok := strings.Cut(apiVersion, "/"); ok && version != "" {
		return version
	}

	if strings.HasPrefix(apiVersion, "v") {
		return apiVersion
	}

	return ""
}

func readDashboardByUID(ctx context.Context, client *goapi.GrafanaHTTPAPI, uid, preferredAPIVersion string) (*dashboards.GetDashboardByUIDOK, error) {
	return client.Dashboards.GetDashboardByUID(uid, func(op *runtime.ClientOperation) {
		op.Context = ctx
		if preferredAPIVersion != "" {
			op.Params = newReadDashboardByUIDParams(ctx, uid, preferredAPIVersion)
		}
	})
}

type readDashboardByUIDParams struct {
	*dashboards.GetDashboardByUIDParams
	apiVersion string
}

func newReadDashboardByUIDParams(ctx context.Context, uid, apiVersion string) *readDashboardByUIDParams {
	return &readDashboardByUIDParams{
		GetDashboardByUIDParams: dashboards.NewGetDashboardByUIDParams().WithContext(ctx).WithUID(uid),
		apiVersion:              apiVersion,
	}
}

func (p *readDashboardByUIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
	if err := p.GetDashboardByUIDParams.WriteToRequest(r, reg); err != nil {
		return err
	}
	if p.apiVersion != "" {
		if err := r.SetQueryParam("apiVersion", p.apiVersion); err != nil {
			return err
		}
	}
	return nil
}

func normalizeDashboardConfigJSONForState(configJSON string, remoteDashJSON map[string]any) (string, error) {
	// Skip if configJSON string is a sha256 hash.
	if configJSON != "" && !common.SHA256Regexp.MatchString(configJSON) {
		configuredDashJSON, err := UnmarshalDashboardConfigJSON(configJSON)
		if err != nil {
			return "", err
		}
		if isKubernetesStyleDashboard(configuredDashJSON) {
			return normalizeKubernetesDashboardConfigJSONForState(configuredDashJSON, remoteDashJSON)
		}
		if _, ok := configuredDashJSON["uid"].(string); !ok {
			delete(remoteDashJSON, "uid")
		}
	}
	return NormalizeDashboardConfigJSON(remoteDashJSON), nil
}

func normalizeKubernetesDashboardConfigJSONForState(configuredDashJSON map[string]any, remoteDashJSON map[string]any) (string, error) {
	configuredSpec, ok := configuredDashJSON["spec"].(map[string]any)
	if !ok {
		return NormalizeDashboardConfigJSON(configuredDashJSON), nil
	}

	localSpecJSON, _, err := normalizeDashboardBodyJSON(configuredSpec)
	if err != nil {
		return "", err
	}
	remoteSpecJSON, remoteSpecMap, err := normalizeDashboardBodyJSON(remoteDashJSON)
	if err != nil {
		return "", err
	}
	if localSpecJSON == remoteSpecJSON {
		return NormalizeDashboardConfigJSON(configuredDashJSON), nil
	}

	stateDashJSON, err := cloneDashboardJSON(configuredDashJSON)
	if err != nil {
		return "", err
	}
	stateDashJSON["spec"] = remoteSpecMap
	return NormalizeDashboardConfigJSON(stateDashJSON), nil
}

func normalizeDashboardBodyJSON(dashboardJSON map[string]any) (string, map[string]any, error) {
	normalizedDashJSON, err := cloneDashboardJSON(dashboardJSON)
	if err != nil {
		return "", nil, err
	}
	delete(normalizedDashJSON, "uid")

	normalizedJSON := NormalizeDashboardConfigJSON(normalizedDashJSON)
	normalizedMap, err := UnmarshalDashboardConfigJSON(normalizedJSON)
	if err != nil {
		return "", nil, err
	}
	return normalizedJSON, normalizedMap, nil
}

func cloneDashboardJSON(dashboardJSON map[string]any) (map[string]any, error) {
	clonedJSONBytes, err := json.Marshal(dashboardJSON)
	if err != nil {
		return nil, err
	}
	clonedDashboardJSON, err := UnmarshalDashboardConfigJSON(string(clonedJSONBytes))
	if err != nil {
		return nil, err
	}
	return clonedDashboardJSON, nil
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

// NormalizeDashboardConfigJSON is the normalization function for the `config_json` field.
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

// dashboardFolderValue converts the API-returned FolderUID to the appropriate types.String.
// When FolderUID is empty (dashboard is in General folder), returns null so that
// an unset `folder` attribute in config stays null in state (no perpetual diff).
func dashboardFolderValue(folderUID string) types.String {
	if folderUID == "" {
		return types.StringNull()
	}
	return types.StringValue(folderUID)
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

// dashboardFolderPlanModifier suppresses diff when folder only differs by org-ID prefix
// or between "0" and empty string — matching the SDKv2 DiffSuppressFunc behaviour.
type dashboardFolderPlanModifier struct{}

func (m dashboardFolderPlanModifier) Description(_ context.Context) string {
	return "Suppresses diff when folder only differs by org-ID prefix or between '0' and empty string."
}

func (m dashboardFolderPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m dashboardFolderPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	_, oldFolder := SplitOrgResourceID(req.StateValue.ValueString())
	_, newFolder := SplitOrgResourceID(req.PlanValue.ValueString())

	if (oldFolder == "0" && newFolder == "") || (oldFolder == "" && newFolder == "0") || oldFolder == newFolder {
		resp.PlanValue = req.StateValue
	}
}

// jsonStringValidator validates that a string attribute contains valid JSON.
type jsonStringValidator struct{}

func (v jsonStringValidator) Description(_ context.Context) string {
	return "value must be valid JSON"
}

func (v jsonStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var js map[string]any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &js); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", err.Error())
	}
}

package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	openapiruntime "github.com/go-openapi/runtime"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	resourceOrganizationPreferencesName = "grafana_organization_preferences"
)

var resourceOrganizationPreferencesID = common.NewResourceID(common.IntIDField("orgID"))

// Check interface
var _ resource.ResourceWithImportState = (*organizationPreferencesResource)(nil)

func makeResourceOrganizationPreferences() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceOrganizationPreferencesName,
		resourceOrganizationPreferencesID,
		&organizationPreferencesResource{},
	).WithLister(listerFunction(listOrganizationPreferences))
}

type organizationPreferencesModel struct {
	ID               types.String `tfsdk:"id"`
	OrgID            types.String `tfsdk:"org_id"`
	Theme            types.String `tfsdk:"theme"`
	HomeDashboardUID types.String `tfsdk:"home_dashboard_uid"`
	Timezone         types.String `tfsdk:"timezone"`
	WeekStart        types.String `tfsdk:"week_start"`
}

type organizationPreferencesResource struct {
	basePluginFrameworkResource
}

// setStringFromAPI returns null when currentOrPlanned is null and apiVal is empty, so Terraform
// state stays consistent (no "inconsistent result after apply" or drift on refresh).
func setStringFromAPI(currentOrPlanned types.String, apiVal string) types.String {
	if currentOrPlanned.IsNull() && apiVal == "" {
		return types.StringNull()
	}
	return types.StringValue(apiVal)
}

// weekStartValidator allows null/unknown (optional attribute unset) and otherwise validates OneOf.
type weekStartValidator struct{}

func (weekStartValidator) Description(ctx context.Context) string {
	return "Value must be one of: sunday, monday, saturday, or empty string."
}

func (weekStartValidator) MarkdownDescription(ctx context.Context) string {
	return weekStartValidator{}.Description(ctx)
}

func (v weekStartValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	stringvalidator.OneOf("sunday", "monday", "saturday", "").ValidateString(ctx, req, resp)
}

// Grafana may return 401 on PUT /org/preferences under load or right after a new org/dashboard.
// OSS acc job 68902735509: GET /org/preferences succeeded (no "not yet accessible" diagnostic)
// but PUT still failed with updateOrgPreferencesUnauthorized after retries — so GET is not a
// reliable gate for PUT. When home_dashboard_uid is set we apply a two-phase PUT: general prefs
// first, then home dashboard UID (avoids validation/races on the dashboard reference).
const (
	orgPrefsRetryAttempts = 15
	orgPrefsRetryDelay    = 3 * time.Second
	// CI (e.g. run 23653044808): TestAccResourceOrganizationPreferences fails with PUT /org/preferences 401
	// on newly created orgs while TestAccResourceOrganizationPreferences_OrgScoped (default org) passes.
	orgPrefsNewOrgSettleDelay = 2 * time.Second
	// Cursor debug NDJSON ingest (written to session log file by the ingest server).
	orgPrefsCursorDebugIngest = "http://127.0.0.1:7392/ingest/c3867395-5cb0-4f1c-823e-e0960dbfac06"
)

// #region agent log

// orgPrefsDebugLogPathWalk walks dir upward until go.mod exists, then returns <module>/.cursor/debug-42b169.ndjson.
// Use .ndjson (not .log): this repo gitignores *.log, which hid the file from tooling and sync.
func orgPrefsDebugLogPathWalk(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, ".cursor", "debug-42b169.ndjson")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// orgPrefsDebugLogPathFromSourceFile walks upward from a stack frame in this package to go.mod.
// With -trimpath, Caller paths are module-relative (not on disk) — those frames are skipped.
func orgPrefsDebugLogPathFromSourceFile() string {
	for skip := 1; skip < 24; skip++ {
		_, file, _, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		if !filepath.IsAbs(file) {
			continue
		}
		if p := orgPrefsDebugLogPathWalk(filepath.Dir(file)); p != "" {
			return p
		}
	}
	return ""
}

// orgPrefsDebugLogPathFromExecutable finds the module root via the provider binary path (external plugin runs).
func orgPrefsDebugLogPathFromExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return orgPrefsDebugLogPathWalk(filepath.Dir(exe))
}

func orgPrefsDebugLogPathModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return orgPrefsDebugLogPathWalk(dir)
}

func tfAccLike() bool {
	v := strings.TrimSpace(os.Getenv("TF_ACC"))
	return v == "1" || strings.EqualFold(v, "true")
}

// addOrgPrefsDebugDotCursorAndMirror adds the .cursor NDJSON path and a duplicate at the module root
// (org-prefs-debug-42b169.ndjson). Some environments index or sync the repo root but not .cursor/.
func addOrgPrefsDebugDotCursorAndMirror(add func(string), dotCursorPath string) {
	if dotCursorPath == "" {
		return
	}
	add(dotCursorPath)
	modRoot := filepath.Dir(filepath.Dir(dotCursorPath))
	if modRoot == "" || modRoot == "." {
		return
	}
	add(filepath.Join(modRoot, "org-prefs-debug-42b169.ndjson"))
}

func debugOrgPrefsLogPaths() []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if e := strings.TrimSpace(os.Getenv("GRAFANA_ORG_PREFS_DEBUG_LOG")); e != "" {
		// Default CI/local path is under .cursor/; mirror to repo root NDJSON for tooling that skips .cursor/.
		if strings.Contains(e, ".cursor") && strings.HasSuffix(e, "debug-42b169.ndjson") {
			addOrgPrefsDebugDotCursorAndMirror(add, e)
		} else {
			add(e)
		}
	}
	if ws := strings.TrimSpace(os.Getenv("GITHUB_WORKSPACE")); ws != "" {
		addOrgPrefsDebugDotCursorAndMirror(add, filepath.Join(ws, ".cursor", "debug-42b169.ndjson"))
	}
	addOrgPrefsDebugDotCursorAndMirror(add, orgPrefsDebugLogPathFromSourceFile())
	addOrgPrefsDebugDotCursorAndMirror(add, orgPrefsDebugLogPathModuleRoot())
	addOrgPrefsDebugDotCursorAndMirror(add, orgPrefsDebugLogPathFromExecutable())
	add(filepath.Join(os.TempDir(), "grafana-org-prefs-debug-42b169.ndjson"))
	return out
}

func debugOrgPrefsNDJSON(ctx context.Context, hypothesisID, location, message string, data map[string]any) {
	if ctx == nil {
		ctx = context.Background()
	}
	payload := map[string]any{
		"sessionId":    "42b169",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	dataJSON, _ := json.Marshal(data)
	tflog.Info(ctx, "org_prefs_agent_debug", map[string]interface{}{
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data_json":    string(dataJSON),
	})
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if tfAccLike() {
		fmt.Fprintf(os.Stderr, "AGENT_DEBUG_ORG_PREFS_NDJSON %s\n", string(b))
	}
	line := append(append([]byte(nil), b...), '\n')
	for _, p := range debugOrgPrefsLogPaths() {
		if p == "" {
			continue
		}
		dir := filepath.Dir(p)
		if dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		f, oerr := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if oerr != nil {
			continue
		}
		_, _ = f.Write(line)
		_ = f.Close()
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, orgPrefsCursorDebugIngest, bytes.NewReader(b))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Debug-Session-Id", "42b169")
	client := &http.Client{Timeout: 900 * time.Millisecond}
	go func(r *http.Request) {
		resp, doErr := client.Do(r)
		if doErr != nil {
			return
		}
		_ = resp.Body.Close()
	}(req)
}

// #endregion

func isRetryableOrgPrefsAuthError(err error) bool {
	if err == nil {
		return false
	}
	var status openapiruntime.ClientResponseStatus
	if errors.As(err, &status) && (status.IsCode(401) || status.IsCode(403)) {
		return true
	}
	// Fallback if error type doesn't implement ClientResponseStatus (e.g. wrapped)
	errStr := err.Error()
	return strings.Contains(errStr, "401") || strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "Unauthorized") || strings.Contains(errStr, "Forbidden")
}

// updateOrgPreferencesWithRetryWithDelay calls UpdateOrgPreferences with optional initial delay and retries on 401/403.
func updateOrgPreferencesWithRetryWithDelay(ctx context.Context, client *goapi.GrafanaHTTPAPI, body *models.UpdatePrefsCmd, initialDelay time.Duration) error {
	if initialDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(initialDelay):
		}
	}
	var lastErr error
	for attempt := 0; attempt < orgPrefsRetryAttempts; attempt++ {
		_, lastErr = client.OrgPreferences.UpdateOrgPreferences(body)
		if lastErr == nil {
			// #region agent log
			debugOrgPrefsNDJSON(ctx, "B", "resource_organization_preferences.go:UpdateOrgPrefsRetry", "update succeeded", map[string]any{
				"attempt": attempt,
			})
			// #endregion
			return nil
		}
		errMsg := lastErr.Error()
		if len(errMsg) > 240 {
			errMsg = errMsg[:240]
		}
		// #region agent log
		debugOrgPrefsNDJSON(ctx, "B,C", "resource_organization_preferences.go:UpdateOrgPrefsRetry", "update attempt failed", map[string]any{
			"attempt":       attempt,
			"retryableAuth": isRetryableOrgPrefsAuthError(lastErr),
			"errType":       fmt.Sprintf("%T", lastErr),
			"errMsg":        errMsg,
			"willRetryMore": isRetryableOrgPrefsAuthError(lastErr) && attempt < orgPrefsRetryAttempts-1,
		})
		// #endregion
		if isRetryableOrgPrefsAuthError(lastErr) && attempt < orgPrefsRetryAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(orgPrefsRetryDelay):
			}
			continue
		}
		return lastErr
	}
	return lastErr
}

// updateOrgPreferencesPhased runs one or two PUTs. With a non-empty homeDashboardUID, we first PUT
// theme/timezone/week_start without a dashboard, then PUT the full payload (OSS acc 68902735509:
// GET ok but PUT 401 when home dashboard was set in the same apply as a new org).
func updateOrgPreferencesPhased(ctx context.Context, client *goapi.GrafanaHTTPAPI, theme, homeDashboardUID, timezone, weekStart string) error {
	if homeDashboardUID == "" {
		return updateOrgPreferencesWithRetryWithDelay(ctx, client, &models.UpdatePrefsCmd{
			Theme:            theme,
			HomeDashboardUID: "",
			Timezone:         timezone,
			WeekStart:        weekStart,
		}, 0)
	}
	// #region agent log
	debugOrgPrefsNDJSON(ctx, "E", "resource_organization_preferences.go:phased", "phase1 prefs without home_dashboard_uid", map[string]any{
		"homeDashboardUID": homeDashboardUID,
	})
	// #endregion
	if err := updateOrgPreferencesWithRetryWithDelay(ctx, client, &models.UpdatePrefsCmd{
		Theme:            theme,
		HomeDashboardUID: "",
		Timezone:         timezone,
		WeekStart:        weekStart,
	}, 0); err != nil {
		return err
	}
	// #region agent log
	debugOrgPrefsNDJSON(ctx, "E", "resource_organization_preferences.go:phased", "phase2 prefs with home_dashboard_uid", map[string]any{
		"homeDashboardUID": homeDashboardUID,
	})
	// #endregion
	return updateOrgPreferencesWithRetryWithDelay(ctx, client, &models.UpdatePrefsCmd{
		Theme:            theme,
		HomeDashboardUID: homeDashboardUID,
		Timezone:         timezone,
		WeekStart:        weekStart,
	}, 0)
}

func (r *organizationPreferencesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceOrganizationPreferencesName
}

func (r *organizationPreferencesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/organization-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/preferences/#get-current-org-prefs)
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
			"theme": schema.StringAttribute{
				Optional:    true,
				Description: "The Organization theme. Any string value is supported, including custom themes. Common values are `light`, `dark`, `system`, or an empty string for the default.",
			},
			"home_dashboard_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The Organization home dashboard UID. This is only available in Grafana 9.0+.",
			},
			"timezone": schema.StringAttribute{
				Optional:    true,
				Description: "The Organization timezone. Any string value is supported, including IANA timezone names. Common values are `utc`, `browser`, or an empty string for the default.",
			},
			"week_start": schema.StringAttribute{
				Optional:    true,
				Description: "The Organization week start day. Available values are `sunday`, `monday`, `saturday`, or an empty string for the default. Defaults to ``.",
				Validators: []validator.String{
					weekStartValidator{},
				},
			},
		},
	}
}

func (r *organizationPreferencesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationPreferencesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	// #region agent log
	debugOrgPrefsNDJSON(ctx, "A", "resource_organization_preferences.go:Create", "client for org prefs create", map[string]any{
		"planOrgIDStr":  data.OrgID.ValueString(),
		"resolvedOrgID": orgID,
	})
	// #endregion

	if orgID != 1 {
		// #region agent log
		debugOrgPrefsNDJSON(ctx, "F", "resource_organization_preferences.go:Create", "settle delay before first PUT (non-default org)", map[string]any{
			"resolvedOrgID": orgID,
			"delayMs":       orgPrefsNewOrgSettleDelay.Milliseconds(),
		})
		// #endregion
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Failed to update organization preferences", ctx.Err().Error())
			return
		case <-time.After(orgPrefsNewOrgSettleDelay):
		}
	}

	theme := data.Theme.ValueString()
	homeDashboardUID := data.HomeDashboardUID.ValueString()
	timezone := data.Timezone.ValueString()
	weekStart := data.WeekStart.ValueString()

	err = updateOrgPreferencesPhased(ctx, client, theme, homeDashboardUID, timezone, weekStart)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update organization preferences", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	// Read back from API to populate any server-side defaults
	readResp, err := client.OrgPreferences.GetOrgPreferences()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read organization preferences after create", err.Error())
		return
	}
	prefs := readResp.Payload
	data.Theme = setStringFromAPI(data.Theme, prefs.Theme)
	data.HomeDashboardUID = setStringFromAPI(data.HomeDashboardUID, prefs.HomeDashboardUID)
	data.Timezone = setStringFromAPI(data.Timezone, prefs.Timezone)
	data.WeekStart = setStringFromAPI(data.WeekStart, prefs.WeekStart)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationPreferencesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationPreferencesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, _, err := r.clientFromExistingOrgResource(resourceOrganizationPreferencesID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	apiResp, err := client.OrgPreferences.GetOrgPreferences()
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read organization preferences", err.Error())
		return
	}

	prefs := apiResp.Payload
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.Theme = setStringFromAPI(data.Theme, prefs.Theme)
	data.HomeDashboardUID = setStringFromAPI(data.HomeDashboardUID, prefs.HomeDashboardUID)
	data.Timezone = setStringFromAPI(data.Timezone, prefs.Timezone)
	data.WeekStart = setStringFromAPI(data.WeekStart, prefs.WeekStart)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationPreferencesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationPreferencesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	// #region agent log
	debugOrgPrefsNDJSON(ctx, "A", "resource_organization_preferences.go:Update", "client for org prefs update", map[string]any{
		"planOrgIDStr":  data.OrgID.ValueString(),
		"resolvedOrgID": orgID,
	})
	// #endregion

	theme := data.Theme.ValueString()
	homeDashboardUID := data.HomeDashboardUID.ValueString()
	timezone := data.Timezone.ValueString()
	weekStart := data.WeekStart.ValueString()

	err = updateOrgPreferencesPhased(ctx, client, theme, homeDashboardUID, timezone, weekStart)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update organization preferences", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	// Read back from API so state matches what Read would return (avoids "inconsistent result after apply")
	readResp, err := client.OrgPreferences.GetOrgPreferences()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read organization preferences after update", err.Error())
		return
	}
	prefs := readResp.Payload
	data.Theme = setStringFromAPI(data.Theme, prefs.Theme)
	data.HomeDashboardUID = setStringFromAPI(data.HomeDashboardUID, prefs.HomeDashboardUID)
	data.Timezone = setStringFromAPI(data.Timezone, prefs.Timezone)
	data.WeekStart = setStringFromAPI(data.WeekStart, prefs.WeekStart)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationPreferencesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationPreferencesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, _, err := r.clientFromExistingOrgResource(resourceOrganizationPreferencesID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	err = updateOrgPreferencesWithRetryWithDelay(ctx, client, &models.UpdatePrefsCmd{}, 0)
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset organization preferences", err.Error())
		return
	}
}

func (r *organizationPreferencesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID is the org ID (e.g. "1" or "2")
	client, orgID, _, err := r.clientFromExistingOrgResource(resourceOrganizationPreferencesID, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	apiResp, err := client.OrgPreferences.GetOrgPreferences()
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.Diagnostics.AddError("Organization preferences not found", "The organization may not exist or preferences may not be accessible.")
			return
		}
		resp.Diagnostics.AddError("Failed to read organization preferences", err.Error())
		return
	}

	prefs := apiResp.Payload
	// Import state from API as-is so ImportStateVerify and refresh see the same values.
	state := organizationPreferencesModel{
		ID:               types.StringValue(req.ID),
		OrgID:            types.StringValue(strconv.FormatInt(orgID, 10)),
		Theme:            types.StringValue(prefs.Theme),
		HomeDashboardUID: types.StringValue(prefs.HomeDashboardUID),
		Timezone:         types.StringValue(prefs.Timezone),
		WeekStart:        types.StringValue(prefs.WeekStart),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func listOrganizationPreferences(ctx context.Context, client *goapi.GrafanaHTTPAPI, data *ListerData) ([]string, error) {
	orgIDs, err := listOrganizations(ctx, client, data)
	if err != nil {
		return nil, err
	}
	// Default org. We can set preferences for it even if it can't be managed otherwise.
	orgIDs = append(orgIDs, "1")
	return orgIDs, nil
}

package grafana

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
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

// Grafana may return 401 briefly after creating a new org until the creating user's membership is propagated.
// We use an initial delay on Create and retry on 401.
const orgPrefsRetryAttempts = 25
const orgPrefsRetryDelay = 6 * time.Second

func isRetryableOrgPrefs401(err error) bool {
	if err == nil {
		return false
	}
	var status runtime.ClientResponseStatus
	if errors.As(err, &status) && status.IsCode(401) {
		return true
	}
	// Fallback if error type doesn't implement ClientResponseStatus (e.g. wrapped)
	errStr := err.Error()
	return strings.Contains(errStr, "401") || strings.Contains(errStr, "Unauthorized")
}

// updateOrgPreferencesWithRetryWithDelay calls UpdateOrgPreferences with optional initial delay and retries on 401.
// initialDelay is used on Create to give new orgs time for membership to propagate before the first attempt.
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
			return nil
		}
		if isRetryableOrgPrefs401(lastErr) && attempt < orgPrefsRetryAttempts-1 {
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

	theme := data.Theme.ValueString()
	homeDashboardUID := data.HomeDashboardUID.ValueString()
	timezone := data.Timezone.ValueString()
	weekStart := data.WeekStart.ValueString()

	// Initial delay for new orgs so Grafana can propagate membership before first API call.
	err = updateOrgPreferencesWithRetryWithDelay(ctx, client, &models.UpdatePrefsCmd{
		Theme:            theme,
		HomeDashboardUID: homeDashboardUID,
		Timezone:         timezone,
		WeekStart:        weekStart,
	}, 15*time.Second)
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

	theme := data.Theme.ValueString()
	homeDashboardUID := data.HomeDashboardUID.ValueString()
	timezone := data.Timezone.ValueString()
	weekStart := data.WeekStart.ValueString()

	err = updateOrgPreferencesWithRetryWithDelay(ctx, client, &models.UpdatePrefsCmd{
		Theme:            theme,
		HomeDashboardUID: homeDashboardUID,
		Timezone:         timezone,
		WeekStart:        weekStart,
	}, 0)
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

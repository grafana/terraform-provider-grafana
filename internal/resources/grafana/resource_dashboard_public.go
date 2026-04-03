package grafana

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/client/dashboard_public"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &publicDashboardResource{}
	_ resource.ResourceWithConfigure   = &publicDashboardResource{}
	_ resource.ResourceWithImportState = &publicDashboardResource{}

	resourcePublicDashboardName = "grafana_dashboard_public"
	resourcePublicDashboardID   = common.NewResourceID(
		common.OptionalIntIDField("orgID"),
		common.StringIDField("dashboardUID"),
		common.StringIDField("publicDashboardUID"),
	)
)

func resourcePublicDashboard() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourcePublicDashboardName,
		resourcePublicDashboardID,
		&publicDashboardResource{},
	)
}

type resourcePublicDashboardModel struct {
	ID                   types.String `tfsdk:"id"`
	OrgID                types.String `tfsdk:"org_id"`
	UID                  types.String `tfsdk:"uid"`
	DashboardUID         types.String `tfsdk:"dashboard_uid"`
	AccessToken          types.String `tfsdk:"access_token"`
	TimeSelectionEnabled types.Bool   `tfsdk:"time_selection_enabled"`
	IsEnabled            types.Bool   `tfsdk:"is_enabled"`
	AnnotationsEnabled   types.Bool   `tfsdk:"annotations_enabled"`
	Share                types.String `tfsdk:"share"`
}

type publicDashboardResource struct {
	basePluginFrameworkResource
}

func (r *publicDashboardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourcePublicDashboardName
}

func (r *publicDashboardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages Grafana public dashboards.

**Note:** This resource is available only with Grafana 10.2+.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/share-dashboards-panels/shared-dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/next/developers/http_api/dashboard_public/)
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
				Optional: true,
				Computed: true,
				Description: "The unique identifier of a public dashboard. " +
					"It's automatically generated if not provided when creating a public dashboard. ",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_uid": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the original dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_token": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "A public unique identifier of a public dashboard. This is used to construct its URL. " +
					"It's automatically generated if not provided when creating a public dashboard. ",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"time_selection_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set to `true` to enable the time picker in the public dashboard. The default value is `false`.",
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set to `true` to enable the public dashboard. The default value is `false`.",
			},
			"annotations_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set to `true` to show annotations. The default value is `false`.",
			},
			"share": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set the share mode. The default value is `public`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *publicDashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data, diags := r.read(ctx, req.ID)
	resp.Diagnostics = diags
	if resp.Diagnostics.HasError() {
		return
	}
	if data == nil {
		resp.Diagnostics.AddError("Resource not found", fmt.Sprintf("public dashboard %q not found", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *publicDashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourcePublicDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	payload := publicDashboardFromModel(&data)
	createResp, err := client.DashboardPublic.CreatePublicDashboard(data.DashboardUID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create public dashboard", err.Error())
		return
	}
	pd := createResp.Payload
	data.ID = types.StringValue(resourcePublicDashboardID.Make(orgID, pd.DashboardUID, pd.UID))

	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *publicDashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourcePublicDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, data.ID.ValueString())
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

func (r *publicDashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourcePublicDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, split, err := r.clientFromExistingOrgResource(resourcePublicDashboardID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	dashboardUID := fmt.Sprintf("%v", split[0])
	publicDashboardUID := fmt.Sprintf("%v", split[1])

	params := dashboard_public.NewUpdatePublicDashboardParams().
		WithDashboardUID(dashboardUID).
		WithUID(publicDashboardUID).
		WithBody(publicDashboardFromModel(&data))
	updateResp, err := client.DashboardPublic.UpdatePublicDashboard(params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update public dashboard", err.Error())
		return
	}
	pd := updateResp.Payload
	data.ID = types.StringValue(fmt.Sprintf("%d:%s:%s", orgID, pd.DashboardUID, pd.UID))

	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *publicDashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourcePublicDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourcePublicDashboardID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	dashboardUID := fmt.Sprintf("%v", split[0])
	publicDashboardUID := fmt.Sprintf("%v", split[1])

	_, err = client.DashboardPublic.DeletePublicDashboard(publicDashboardUID, dashboardUID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete public dashboard", err.Error())
	}
}

func (r *publicDashboardResource) read(_ context.Context, id string) (*resourcePublicDashboardModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, orgID, split, err := r.clientFromExistingOrgResource(resourcePublicDashboardID, id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, diags
	}
	dashboardUID := fmt.Sprintf("%v", split[0])

	resp, err := client.DashboardPublic.GetPublicDashboard(dashboardUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Failed to read public dashboard", err.Error())
		return nil, diags
	}
	pd := resp.Payload

	data := &resourcePublicDashboardModel{
		ID:                   types.StringValue(fmt.Sprintf("%d:%s:%s", orgID, pd.DashboardUID, pd.UID)),
		OrgID:                types.StringValue(strconv.FormatInt(orgID, 10)),
		UID:                  types.StringValue(pd.UID),
		DashboardUID:         types.StringValue(pd.DashboardUID),
		AccessToken:          types.StringValue(pd.AccessToken),
		TimeSelectionEnabled: types.BoolValue(pd.TimeSelectionEnabled),
		IsEnabled:            types.BoolValue(pd.IsEnabled),
		AnnotationsEnabled:   types.BoolValue(pd.AnnotationsEnabled),
		Share:                types.StringValue(string(pd.Share)),
	}
	return data, diags
}

func publicDashboardFromModel(data *resourcePublicDashboardModel) *models.PublicDashboardDTO {
	return &models.PublicDashboardDTO{
		UID:                  data.UID.ValueString(),
		AccessToken:          data.AccessToken.ValueString(),
		TimeSelectionEnabled: data.TimeSelectionEnabled.ValueBool(),
		IsEnabled:            data.IsEnabled.ValueBool(),
		AnnotationsEnabled:   data.AnnotationsEnabled.ValueBool(),
		Share:                models.ShareType(data.Share.ValueString()),
	}
}

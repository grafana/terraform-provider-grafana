package frontendo11y

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
)

var (
	resourceFrontendO11yAppName        = "grafana_frontend_o11y_app"
	resourceFrontendO11yAppTerraformID = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("name"))

	// Check interface
	_ resource.ResourceWithImportState = (*resourceFrontendO11yApp)(nil)
)

type resourceFrontendO11yApp struct {
	client     *frontendo11yapi.Client
	gcomClient *gcom.APIClient
}

func makeResourceFrontendO11yApp() *common.Resource {
	return common.NewResource(
		common.CategoryFrontendO11y,
		resourceFrontendO11yAppName,
		resourceFrontendO11yAppTerraformID,
		&resourceFrontendO11yApp{},
	)
}

func (r *resourceFrontendO11yApp) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || (r.client != nil && r.gcomClient != nil) {
		return
	}

	fc, gc, err := withClientForResource(req, resp)
	if err != nil {
		return
	}

	r.client = fc
	r.gcomClient = gc
}

func (r *resourceFrontendO11yApp) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceFrontendO11yAppName
}

func (r *resourceFrontendO11yApp) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The Terraform Resource ID. This is auto-generated from Frontend Observability API.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					// See https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#usestateforunknown
					// for details on how this works.
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"stack_id": schema.Int64Attribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of Frontend Observability App. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"collector_endpoint": schema.StringAttribute{
				Description: "The collector URL Grafana Cloud Frontend Observability. Use this endpoint to send your Telemetry.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allowed_origins": schema.ListAttribute{
				Description: "A list of allowed origins for CORS.",
				ElementType: types.StringType,
				Required:    true,
			},
			"extra_log_attributes": schema.MapAttribute{
				Description: "The extra attributes to append in each signal.",
				ElementType: types.StringType,
				Required:    true,
			},
			"settings": schema.MapAttribute{
				MarkdownDescription: "The key-value settings of the Frontend Observability app. Available Settings: `{combineLabData=(0|1),geolocation.level=(0|1),geolocation.level=0-4,geolocation.country_denylist=<comma-separated-list-of-country-codes>}`",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf([]string{
							"combineLabData",
							"geolocation.enabled",
							"geolocation.level",
							"geolocation.country_denylist",
						}...),
					),
				},
				ElementType: types.StringType,
				Required:    true,
			},
		},
	}
}

func (r *resourceFrontendO11yApp) getStackCluster(ctx context.Context, stackID string) (string, error) {
	stack, res, err := r.gcomClient.InstancesAPI.GetInstance(ctx, stackID).Execute()
	if err != nil {
		return "", err
	}

	if res.StatusCode >= 500 {
		return "", errors.New("server error")
	}

	if res.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("stack %q not found", stackID)
	}
	return stack.ClusterSlug, nil
}

func (r *resourceFrontendO11yApp) getStack(ctx context.Context, stackID string) (*gcom.FormattedApiInstance, error) {
	stack, res, err := r.gcomClient.InstancesAPI.GetInstance(ctx, stackID).Execute()
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 500 {
		return nil, errors.New("server error")
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("stack %q not found", stackID)
	}
	return stack, nil
}

func (r *resourceFrontendO11yApp) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var dataTF FrontendO11yAppTFModel
	diags := req.Plan.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, diags := dataTF.toClientModel(ctx)
	resp.Diagnostics.Append(diags...)

	stackCluster, err := r.getStackCluster(ctx, dataTF.StackID.String())
	if err != nil {
		resp.Diagnostics.AddError("failed to get Grafana Cloud Stack information", err.Error())
		return
	}

	appClientModel, err := r.client.CreateApp(ctx, apiURLForCluster(stackCluster, r.client.Host()), dataTF.StackID.ValueInt64(), app)
	if err != nil {
		resp.Diagnostics.AddError("failed to get Grafana Cloud Stack information", err.Error())
		return
	}

	appTFState, diags := convertClientModelToTFModel(dataTF.StackID.ValueInt64(), appClientModel)
	resp.Diagnostics.Append(diags...)
	resp.State.Set(ctx, appTFState)
}

func (r *resourceFrontendO11yApp) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	reqParts := strings.Split(req.ID, ":")
	if len(reqParts) != 2 {
		resp.Diagnostics.AddError("incorrect ID format", "Resource ID should be in the format of 'stackID:appID'")
		return
	}
	stackSlug := reqParts[0]
	appID := reqParts[1]
	i64AppID, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid app ID", err.Error())
		return
	}

	stack, err := r.getStack(ctx, stackSlug)
	if err != nil {
		resp.Diagnostics.AddError("failed to get Grafana Cloud Stack information", err.Error())
		return
	}
	appClientModel, err := r.client.GetApp(
		ctx,
		apiURLForCluster(stack.ClusterSlug, r.client.Host()),
		int64(stack.Id),
		i64AppID,
	)

	if err != nil {
		resp.Diagnostics.AddError("failed to get frontend o11y app", err.Error())
		return
	}

	clientTFData, diags := convertClientModelToTFModel(int64(stack.Id), appClientModel)
	resp.Diagnostics.Append(diags...)
	resp.State.Set(ctx, clientTFData)
}

func (r *resourceFrontendO11yApp) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var dataTF FrontendO11yAppTFModel
	diags := req.State.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stackCluster, err := r.getStackCluster(ctx, dataTF.StackID.String())
	if err != nil {
		resp.Diagnostics.AddError("failed to get Grafana Cloud Stack information", err.Error())
		return
	}
	appClientModel, err := r.client.GetApps(
		ctx,
		apiURLForCluster(stackCluster, r.client.Host()),
		dataTF.StackID.ValueInt64(),
	)

	if err != nil {
		resp.Diagnostics.AddError("failed to get frontend o11y app", err.Error())
		return
	}

	for _, app := range appClientModel {
		if app.Name == dataTF.Name.ValueString() {
			clientTFData, diags := convertClientModelToTFModel(dataTF.StackID.ValueInt64(), app)
			resp.Diagnostics.Append(diags...)
			resp.State.Set(ctx, clientTFData)
			return
		}
	}
}

func (r *resourceFrontendO11yApp) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataTF FrontendO11yAppTFModel
	diags := req.Plan.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, diags := dataTF.toClientModel(ctx)
	resp.Diagnostics.Append(diags...)

	stackCluster, err := r.getStackCluster(ctx, dataTF.StackID.String())
	if err != nil {
		resp.Diagnostics.AddError("failed to get Grafana Cloud Stack information", err.Error())
		return
	}
	appClientModel, err := r.client.UpdateApp(ctx, apiURLForCluster(stackCluster, r.client.Host()), dataTF.StackID.ValueInt64(), app.ID, app)

	if err != nil {
		resp.Diagnostics.AddError("failed to update frontend o11y app", err.Error())
		return
	}

	appTFState, diags := convertClientModelToTFModel(dataTF.StackID.ValueInt64(), appClientModel)
	appTFState.CollectorEndpoint = dataTF.CollectorEndpoint
	resp.Diagnostics.Append(diags...)
	resp.State.Set(ctx, appTFState)
}

func (r *resourceFrontendO11yApp) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var dataTF FrontendO11yAppTFModel
	diags := req.State.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stackCluster, err := r.getStackCluster(ctx, dataTF.StackID.String())
	if err != nil {
		resp.Diagnostics.AddError("failed to get Grafana Cloud Stack information", err.Error())
		return
	}

	err = r.client.DeleteApp(
		ctx,
		apiURLForCluster(stackCluster, r.client.Host()),
		dataTF.StackID.ValueInt64(),
		dataTF.ID.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete frontend o11y app", err.Error())
		return
	}

	resp.State.Set(ctx, nil)
}

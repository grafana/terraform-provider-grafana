package connections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/connectionsapi"
)

var (
	resourceMetricsEndpointScrapeJobTerraformName = "grafana_connections_metrics_endpoint_scrape_job"
	resourceMetricsEndpointScrapeJobTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("name"))
)

type resourceMetricsEndpointScrapeJob struct {
	client *connectionsapi.Client
}

func makeResourceMetricsEndpointScrapeJob() *common.Resource {
	return common.NewResource(
		common.CategoryConnections,
		resourceMetricsEndpointScrapeJobTerraformName,
		resourceMetricsEndpointScrapeJobTerraformID,
		&resourceMetricsEndpointScrapeJob{},
	)
}

func (r *resourceMetricsEndpointScrapeJob) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}

	r.client = client
}

func (r *resourceMetricsEndpointScrapeJob) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceMetricsEndpointScrapeJobTerraformName
}

func (r *resourceMetricsEndpointScrapeJob) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ name }}\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#usestateforunknown
					// for details on how this works.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"stack_id": schema.StringAttribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the metrics endpoint scrape job. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the metrics endpoint scrape job is enabled or not.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"authentication_method": schema.StringAttribute{
				Description: "Method to pass authentication credentials: basic or bearer.",
				Validators: []validator.String{
					stringvalidator.OneOf("basic", "bearer"),
					authBasicValidator{},
					authBearerValidator{},
				},
				Required: true,
			},
			"authentication_bearer_token": schema.StringAttribute{
				Description: "Bearer token used for authentication, use if scrape job is using bearer authentication method",
				Sensitive:   true,
				Optional:    true,
			},
			"authentication_basic_username": schema.StringAttribute{
				Description: "Username for basic authentication, use if scrape job is using basic authentication method",
				Optional:    true,
			},
			"authentication_basic_password": schema.StringAttribute{
				Description: "Password for basic authentication, use if scrape job is using basic authentication method",
				Sensitive:   true,
				Optional:    true,
			},
			"url": schema.StringAttribute{
				Description: "The url to scrape metrics from; a valid HTTPs URL is required.",
				Validators:  []validator.String{HTTPSURLValidator{}},
				Required:    true,
			},
			"scrape_interval_seconds": schema.Int64Attribute{
				Description: "Frequency for scraping the metrics endpoint: 30, 60, or 120 seconds.",
				Computed:    true,
				Validators:  []validator.Int64{int64validator.OneOf(30, 60, 120)},
				Default:     int64default.StaticInt64(60),
				Optional:    true,
			},
		},
	}
}

func (r *resourceMetricsEndpointScrapeJob) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("authentication_bearer_token"),
			path.MatchRoot("authentication_basic_username"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("authentication_bearer_token"),
			path.MatchRoot("authentication_basic_password"),
		),
	}
}

func (r *resourceMetricsEndpointScrapeJob) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var dataTF metricsEndpointScrapeJobTFModel
	diags := req.Plan.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobClientModel, err := r.client.CreateMetricsEndpointScrapeJob(ctx, dataTF.StackID.ValueString(), dataTF.Name.ValueString(),
		convertJobTFModelToClientModel(dataTF))
	if err != nil {
		resp.Diagnostics.AddError("failed to create metrics endpoint scrape job", err.Error())
		return
	}

	resp.State.Set(ctx, convertClientModelToTFModel(dataTF.StackID.ValueString(), dataTF.Name.ValueString(), jobClientModel))
}

func (r *resourceMetricsEndpointScrapeJob) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var dataTF metricsEndpointScrapeJobTFModel
	diags := req.State.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobClientModel, err := r.client.GetMetricsEndpointScrapeJob(
		ctx,
		dataTF.StackID.ValueString(),
		dataTF.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("failed to get metrics endpoint scrape job", err.Error())
		return
	}

	jobTF := convertClientModelToTFModel(dataTF.StackID.ValueString(), dataTF.Name.ValueString(), jobClientModel)

	// Set only non-sensitive attributes
	resp.State.SetAttribute(ctx, path.Root("stack_id"), jobTF.StackID)
	resp.State.SetAttribute(ctx, path.Root("name"), jobTF.Name)
	resp.State.SetAttribute(ctx, path.Root("enabled"), jobTF.Enabled)
	resp.State.SetAttribute(ctx, path.Root("authentication_method"), jobTF.AuthenticationMethod)
	resp.State.SetAttribute(ctx, path.Root("url"), jobTF.URL)
	resp.State.SetAttribute(ctx, path.Root("scrape_interval_seconds"), jobTF.ScrapeIntervalSeconds)
}

func (r *resourceMetricsEndpointScrapeJob) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataTF metricsEndpointScrapeJobTFModel
	diags := req.Plan.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobClientModel, err := r.client.UpdateMetricsEndpointScrapeJob(ctx, dataTF.StackID.ValueString(), dataTF.Name.ValueString(),
		convertJobTFModelToClientModel(dataTF))
	if err != nil {
		resp.Diagnostics.AddError("failed to update metrics endpoint scrape job", err.Error())
		return
	}

	resp.State.Set(ctx, convertClientModelToTFModel(dataTF.StackID.ValueString(), dataTF.Name.ValueString(), jobClientModel))
}

func (r *resourceMetricsEndpointScrapeJob) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var dataTF metricsEndpointScrapeJobTFModel
	diags := req.State.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMetricsEndpointScrapeJob(
		ctx,
		dataTF.StackID.ValueString(),
		dataTF.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete metrics endpoint scrape job", err.Error())
		return
	}

	resp.State.Set(ctx, nil)
}

type authBasicValidator struct{}

func (v authBasicValidator) Description(ctx context.Context) string {
	return "Validates that both username and password are provided for authentication basic"
}

func (v authBasicValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that both username and password are provided for authentication basic"
}

func (v authBasicValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "basic" {
		return
	}

	var data metricsEndpointScrapeJobTFModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.AuthenticationBasicUsername.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Required Field",
			"authentication_basic_username is required when authentication_method is basic",
		)
	}
	if data.AuthenticationBasicPassword.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Required Field",
			"authentication_basic_password is required when authentication_method is basic",
		)
	}
}

type authBearerValidator struct{}

func (v authBearerValidator) Description(ctx context.Context) string {
	return "Validates that bearer token is provided for bearer authentication"
}

func (v authBearerValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that bearer token is provided for bearer authentication"
}

func (v authBearerValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "bearer" {
		return
	}

	var data metricsEndpointScrapeJobTFModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.AuthenticationBearerToken.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Required Field",
			"authentication_bearer_token is required when authentication_method is bearer",
		)
	}
}

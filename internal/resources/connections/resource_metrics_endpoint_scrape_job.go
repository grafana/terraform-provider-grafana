package connections

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

var Resources = makeResourceMetricsEndpointScrapeJob()

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

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*connectionsapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.ConnectionsAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Metrics Endpoint API.",
			"Please ensure that connections_url and connections_access_token are set in the provider configuration.",
		)

		return nil, fmt.Errorf("ConnectionsAPI is nil")
	}

	return client.ConnectionsAPI, nil
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
				Description: "The name of the Metrics Endpoint Scrape Job. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the Metrics Endpoint Scrape Job is enabled or not.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"authentication_method": schema.StringAttribute{
				Description: "Method to pass authentication credentials: basic or bearer.",
				Validators: []validator.String{
					stringvalidator.OneOf("basic", "bearer"),
				},
				Required: true,
			},
			"authentication_bearer_token": schema.StringAttribute{
				Description: "Token for authentication bearer.",
				Sensitive:   true,
				Optional:    true,
			},
			"authentication_basic_username": schema.StringAttribute{
				Description: "Username for basic authentication.",
				Optional:    true,
			},
			"authentication_basic_password": schema.StringAttribute{
				Description: "Password for basic authentication.",
				Sensitive:   true,
				Optional:    true,
			},
			"url": schema.StringAttribute{
				Description: "The url to scrape metrics; a valid HTTPs URL is required.",
				//Validators: []validator.String{}, // TODO: Find a validator for urls
				Required: true,
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

func (r *resourceMetricsEndpointScrapeJob) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO implement me
	panic("implement me")
}

func (r *resourceMetricsEndpointScrapeJob) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO implement me
	panic("implement me")
}

func (r *resourceMetricsEndpointScrapeJob) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO implement me
	panic("implement me")
}

func (r *resourceMetricsEndpointScrapeJob) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO implement me
	panic("implement me")
}

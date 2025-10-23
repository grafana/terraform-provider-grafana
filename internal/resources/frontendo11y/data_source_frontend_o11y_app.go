package frontendo11y

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
)

type datasourceFrontendO11yApp struct {
	client     *frontendo11yapi.Client
	gcomClient *gcom.APIClient
}

func makeFrontendO11yAppDataSource() *common.DataSource {
	return common.NewDataSource(
		common.CategoryFrontendO11y,
		resourceFrontendO11yAppName,
		&datasourceFrontendO11yApp{},
	)
}

func (r *datasourceFrontendO11yApp) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	c, gc, err := withClientForDataSource(req, resp)
	if err != nil {
		return
	}

	r.client = c
	r.gcomClient = gc
}

func (r *datasourceFrontendO11yApp) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = resourceFrontendO11yAppName
}

func (r *datasourceFrontendO11yApp) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The Terraform Resource ID. This auto-generated from Frontend Observability API.",
				Computed:    true,
			},
			"stack_id": schema.Int64Attribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the Frontend Observability App. Part of the Terraform Resource ID.",
				Required:    true,
			},
			"collector_endpoint": schema.StringAttribute{
				Description: "The collector URL Grafana Cloud Frontend Observability. Use this endpoint to send your Telemetry.",
				Computed:    true,
			},
			"allowed_origins": schema.ListAttribute{
				Description: "A list of allowed origins for CORS.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"extra_log_attributes": schema.MapAttribute{
				Description: "The extra attributes to append in each signal.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"settings": schema.MapAttribute{
				Description: "The settings of the Frontend Observability App.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// getStackRegion gets the region slug from the stack id
func (r *datasourceFrontendO11yApp) getStackRegion(ctx context.Context, stackID string) (string, error) {
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
	return stack.RegionSlug, nil
}

func (r *datasourceFrontendO11yApp) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var dataTF FrontendO11yAppTFModel
	diags := req.Config.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stackRegionSlug, err := r.getStackRegion(ctx, dataTF.StackID.String())
	if err != nil {
		resp.Diagnostics.AddError("failed to get Grafana Cloud Stack information", err.Error())
		return
	}
	faroEndpointURL := getFrontendO11yAPIURLForRegion(stackRegionSlug)
	appsClientModel, err := r.client.GetApps(ctx, faroEndpointURL, dataTF.StackID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("failed to get frontend o11y apps", err.Error())
		return
	}

	for _, app := range appsClientModel {
		if app.Name == dataTF.Name.ValueString() {
			tfState, diags := convertClientModelToTFModel(dataTF.StackID.ValueInt64(), app)
			resp.Diagnostics.Append(diags...)
			resp.State.Set(ctx, tfState)
			return
		}
	}

	resp.Diagnostics.AddError(fmt.Sprintf("failed to get app %q: not found", dataTF.Name.ValueString()), "please verify the app name and stack ID are correct.")
}

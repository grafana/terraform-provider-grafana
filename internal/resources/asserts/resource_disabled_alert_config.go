package asserts

import (
	"context"
	"fmt"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkretry "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

var (
	_ resource.Resource                = (*disabledAlertConfigResource)(nil)
	_ resource.ResourceWithConfigure   = (*disabledAlertConfigResource)(nil)
	_ resource.ResourceWithImportState = (*disabledAlertConfigResource)(nil)
)

func makeResourceDisabledAlertConfig() *common.Resource {
	return common.NewResource(
		common.CategoryAsserts,
		"grafana_asserts_suppressed_assertions_config",
		common.NewResourceID(common.StringIDField("name")),
		&disabledAlertConfigResource{},
	).WithLister(assertsListerFunction(listDisabledAlertConfigs))
}

type disabledAlertConfigModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	MatchLabels types.Map    `tfsdk:"match_labels"`
}

type disabledAlertConfigResource struct {
	client  *assertsapi.APIClient
	stackID int64
}

func (r *disabledAlertConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	if client.AssertsAPIClient == nil {
		resp.Diagnostics.AddError(
			"Asserts API client is not configured",
			"Please ensure that the Asserts API client is configured.",
		)
		return
	}
	r.client = client.AssertsAPIClient
	r.stackID = client.GrafanaStackID
}

func (r *disabledAlertConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_asserts_suppressed_assertions_config"
}

func (r *disabledAlertConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Knowledge Graph Disabled Alert Configurations through Grafana API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the disabled alert configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"match_labels": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Labels to match for this disabled alert configuration.",
			},
		},
	}
}

func (r *disabledAlertConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data disabledAlertConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.put(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to create disabled alert configuration", err.Error())
		return
	}

	readData, diags := r.read(ctx, data.Name.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *disabledAlertConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data disabledAlertConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, data.Name.ValueString())
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

func (r *disabledAlertConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data disabledAlertConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.put(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to update disabled alert configuration", err.Error())
		return
	}

	readData, diags := r.read(ctx, data.Name.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *disabledAlertConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data disabledAlertConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.AlertConfigurationAPI.DeleteDisabledAlertConfig(ctx, data.Name.ValueString()).
		XScopeOrgID(fmt.Sprintf("%d", r.stackID)).
		Execute()
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete disabled alert configuration", err.Error())
	}
}

func (r *disabledAlertConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", fmt.Sprintf("disabled alert configuration %q not found", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

// put creates or updates a disabled alert config via the Asserts API (upsert).
func (r *disabledAlertConfigResource) put(ctx context.Context, data *disabledAlertConfigModel) error {
	name := data.Name.ValueString()
	dto := assertsapi.DisabledAlertConfigDto{
		Name:      &name,
		ManagedBy: getManagedByTerraform(),
	}

	if !data.MatchLabels.IsNull() && len(data.MatchLabels.Elements()) > 0 {
		var matchLabels map[string]string
		if diags := data.MatchLabels.ElementsAs(context.Background(), &matchLabels, false); diags.HasError() {
			return fmt.Errorf("failed to read match_labels")
		}
		dto.MatchLabels = matchLabels
	}

	_, err := r.client.AlertConfigurationAPI.PutDisabledAlertConfig(ctx).
		DisabledAlertConfigDto(dto).
		XScopeOrgID(fmt.Sprintf("%d", r.stackID)).
		Execute()
	return err
}

// read fetches a disabled alert config by name with retry for eventual consistency.
func (r *disabledAlertConfigResource) read(ctx context.Context, name string) (*disabledAlertConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var foundConfig *assertsapi.DisabledAlertConfigDto

	err := withRetryRead(ctx, func(retryCount, maxRetries int) *sdkretry.RetryError {
		configs, _, err := r.client.AlertConfigurationAPI.GetAllDisabledAlertConfigs(ctx).
			XScopeOrgID(fmt.Sprintf("%d", r.stackID)).
			Execute()
		if err != nil {
			return createAPIError("get disabled alert configurations", retryCount, maxRetries, err)
		}
		for _, config := range configs.DisabledAlertConfigs {
			if config.Name != nil && *config.Name == name {
				foundConfig = &config
				return nil
			}
		}
		if retryCount >= maxRetries {
			return createNonRetryableError("disabled alert configuration", name, retryCount)
		}
		return createRetryableError("disabled alert configuration", name, retryCount, maxRetries)
	})
	if err != nil {
		diags.AddError("Failed to read disabled alert configuration", err.Error())
		return nil, diags
	}
	if foundConfig == nil {
		return nil, diags
	}

	var matchLabels types.Map
	if len(foundConfig.MatchLabels) > 0 {
		var mapDiags diag.Diagnostics
		matchLabels, mapDiags = types.MapValueFrom(ctx, types.StringType, foundConfig.MatchLabels)
		diags.Append(mapDiags...)
		if diags.HasError() {
			return nil, diags
		}
	} else {
		matchLabels = types.MapNull(types.StringType)
	}

	return &disabledAlertConfigModel{
		ID:          types.StringValue(name),
		Name:        types.StringValue(name),
		MatchLabels: matchLabels,
	}, diags
}

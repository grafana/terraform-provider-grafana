package grafana

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	// Check interface
	_ resource.ResourceWithImportState = (*resourceDataSourceConfigCorrelations)(nil)
)

var (
	resourceDataSourceConfigCorrelationsName = "grafana_data_source_config_correlations"
	resourceDataSourceConfigCorrelationsID   = common.NewResourceID(
		common.StringIDField("datasource_uid"),
	)
)

func makeResourceDataSourceConfigCorrelations() *common.Resource {
	resourceStruct := &resourceDataSourceConfigCorrelations{}
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceDataSourceConfigCorrelationsName,
		resourceDataSourceConfigCorrelationsID,
		resourceStruct,
	)
}

/*type resourceDataSourceConfigCorrelationsModel struct {
	ID            types.String `tfsdk:"id"`
	DatasourceUID types.String `tfsdk:"datasource_uid"`
	Rules         types.String `tfsdk:"rules"` //TODO
}*/

type resourceDataSourceConfigCorrelations struct {
	client *common.Client
}

func (r *resourceDataSourceConfigCorrelations) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDataSourceConfigCorrelationsName
}

func (r *resourceDataSourceConfigCorrelations) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages Correlations for a data source.

!> Warning: The resource is experimental and will be subject to change.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/correlations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/correlations/)

TODO



This resource requires Grafana >=11.5.0.
`,
		Attributes: map[string]schema.Attribute{
			"uid": schema.StringAttribute{
				Computed: true,
			},
			"source_uid": schema.StringAttribute{
				Required:    true,
				Description: "UID of the data source the correlation originates from.",
			},
			"org_id": schema.Int64Attribute{
				Required:    true,
				Description: "OrgID of the data source the correlation originates from.",
			},
			"target_uid": schema.StringAttribute{
				Required:    true,
				Description: "UID of the data source the correlation points to.",
			},
			"label": schema.StringAttribute{
				Required:    true,
				Description: "Label identifying the correlation.",
			},
			"description": schema.StringAttribute{
				Required:    false,
				Description: "Description of the correlation.",
			},
			"provisioned": schema.BoolAttribute{
				Required:    false,
				Computed:    true,
				Description: "True if the correlation was created during provisioning",
				Default:     booldefault.StaticBool(true),
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The type of correlation.",
				Validators: []validator.String{
					stringvalidator.OneOf("query", "external"),
				},
			},
			"config": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"field": schema.StringAttribute{
						Required:    true,
						Description: "Field used to attach the correlation link.",
					},
					"type": schema.StringAttribute{
						Required:           false,
						DeprecationMessage: "This is deprecated: use the type property outside of config",
					},
					"target": schema.StringAttribute{ //TODO: Arbitrary json?
						Required:    true,
						Description: "Target datasource query.",
					},
				},
			},
		},
	}
}

/**
// swagger:model
type CorrelationConfig struct {


	// Source data transformations
	// required:false
	// example: [{"type":"logfmt"}]
	Transformations Transformations `json:"transformations,omitempty"`
}
*/

func (r *resourceDataSourceConfigCorrelations) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Check if the provider data is nil or if the client is already set
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Check if the client is correctly configured
	if client.GrafanaAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana API.",
			"Please ensure that URL and auth are set in the provider configuration.",
		)
		return
	}

	r.client = client
}

func (r *resourceDataSourceConfigCorrelations) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.CorrelationConfig
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	/*
		rulesMap := make(map[string][]string)
		if err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap); err != nil {
			resp.Diagnostics.AddError(
				"Invalid rules JSON",
				fmt.Sprintf("Failed to parse rules for datasource %q: %v. Please ensure the rules are valid JSON.", data.DatasourceUID.ValueString(), err),
			)
			return
		}

		if err := r.updateRules(ctx, &data, rulesMap); err != nil {
			resp.Diagnostics.AddError("Failed to create Correlations", err.Error())
			return
		}

		data.ID = types.StringValue(data.DatasourceUID.ValueString())
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)*/
	return
}

func (r *resourceDataSourceConfigCorrelations) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	/*var data resourceDataSourceConfigCorrelationsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	datasourceUID := data.DatasourceUID.ValueString()
	client := r.client.GrafanaAPI

	getResp, err := client.Correlations.GetCorrelationsBySourceUID(datasourceUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Correlations",
			fmt.Sprintf("Could not read Correlations for datasource %q: %v", datasourceUID, err),
		)
		return
	}

	rulesMap := make(map[string][]string)
	for _, rule := range getResp.Payload.Rules {
		rulesMap[rule.TeamUID] = rule.Rules
	}

	rulesJSON, err := json.Marshal(rulesMap)
	if err != nil {
		// Marshal error should never happen for a valid map
		resp.Diagnostics.AddError(
			"Failed to encode rules",
			fmt.Sprintf("Could not encode Correlations for datasource %q: %v. This is an internal error, please report it.", datasourceUID, err),
		)
		return
	}

	data = resourceDataSourceConfigCorrelationsModel{
		ID:            types.StringValue(datasourceUID),
		DatasourceUID: types.StringValue(datasourceUID),
		Rules:         types.StringValue(string(rulesJSON)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...) */
	return
}

func (r *resourceDataSourceConfigCorrelations) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	/*var data resourceDataSourceConfigCorrelationsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rulesMap := make(map[string][]string)
	if err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap); err != nil {
		resp.Diagnostics.AddError(
			"Invalid rules JSON",
			fmt.Sprintf("Failed to parse updated rules for datasource %q: %v. Please ensure the rules are valid JSON.", data.DatasourceUID.ValueString(), err),
		)
		return
	}

	if err := r.updateRules(ctx, &data, rulesMap); err != nil {
		resp.Diagnostics.AddError("Failed to update Correlations", err.Error())
		return
	}

	data.ID = types.StringValue(data.DatasourceUID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...) */
	return
}

// TODO DELETE
func (r *resourceDataSourceConfigCorrelations) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	return
}

func (r *resourceDataSourceConfigCorrelations) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	/*datasourceUID := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), datasourceUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("datasource_uid"), datasourceUID)...)
	*/
}

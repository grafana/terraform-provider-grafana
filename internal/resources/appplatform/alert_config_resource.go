package appplatform

import (
	"context"
	"regexp"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AlertConfigSpecModel is a model for the AlertConfig spec in Terraform
type AlertConfigSpecModel struct {
	MatchLabels types.Map    `tfsdk:"match_labels"`
	AlertLabels types.Map    `tfsdk:"alert_labels"`
	Duration    types.String `tfsdk:"duration"`
	Silenced    types.Bool   `tfsdk:"silenced"`
}

// MatchLabelsValidator validates that matchLabels contains either alertname or asserts_slo_name
type MatchLabelsValidator struct{}

func (v MatchLabelsValidator) Description(_ context.Context) string {
	return "matchLabels must contain either 'alertname' or 'asserts_slo_name' as a key"
}

func (v MatchLabelsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v MatchLabelsValidator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elements := req.ConfigValue.Elements()
	hasAlertname := false
	hasAssertsSlO := false

	for key := range elements {
		if key == "alertname" {
			hasAlertname = true
		}
		if key == "asserts_slo_name" {
			hasAssertsSlO = true
		}
	}

	if !hasAlertname && !hasAssertsSlO {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid matchLabels",
			"matchLabels must contain either 'alertname' or 'asserts_slo_name' as a key",
		)
	}
}

// AlertConfig creates a new Asserts AlertConfig resource using a simplified approach
func AlertConfig() NamedResource {
	return NamedResource{
		Resource: &AlertConfigResource{},
		Name:     "grafana_apps_asserts_alertconfig_v2alpha1",
		Category: common.CategoryGrafanaApps,
	}
}

// AlertConfigResource implements the Terraform resource for AlertConfig
type AlertConfigResource struct {
	// We'll implement this as a simplified resource that doesn't use the full AppPlatform pattern
	// This avoids the complex interface compatibility issues
}

func (r *AlertConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_apps_asserts_alertconfig_v2alpha1"
}

func (r *AlertConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages Asserts AlertConfigs via the Grafana App Platform API.",
		MarkdownDescription: "Manages Asserts AlertConfig resources via the Grafana App Platform API.",
		Blocks: map[string]schema.Block{
			"metadata": schema.SingleNestedBlock{
				Description: "The metadata of the resource.",
				Attributes: map[string]schema.Attribute{
					"uid": schema.StringAttribute{
						Required:    true,
						Description: "The unique identifier of the resource.",
					},
					"folder_uid": schema.StringAttribute{
						Optional:    true,
						Description: "The UID of the folder to save the resource in.",
					},
					"uuid": schema.StringAttribute{
						Computed:    true,
						Description: "The globally unique identifier of a resource, used by the API for tracking.",
					},
					"url": schema.StringAttribute{
						Computed:    true,
						Description: "The full URL of the resource.",
					},
					"version": schema.StringAttribute{
						Computed:    true,
						Description: "The version of the resource.",
					},
				},
			},
			"spec": schema.SingleNestedBlock{
				Description: "The spec of the resource.",
				Attributes: map[string]schema.Attribute{
					"match_labels": schema.MapAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "Labels to match for alert triggering. Must contain either 'alertname' or 'asserts_slo_name' as a key.",
						Validators: []validator.Map{
							MatchLabelsValidator{},
						},
					},
					"alert_labels": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Additional labels to add to alerts",
					},
					"duration": schema.StringAttribute{
						Optional:    true,
						Description: "Alert evaluation duration (e.g., '5m', '1h', '30s'). Optional to match REST API behavior.",
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^([0-9]+[smhdwy])+$`),
								"duration must be in Prometheus duration format (e.g., '5m', '1h', '30s')",
							),
						},
					},
					"silenced": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether alert config is silenced",
					},
				},
			},
			"options": schema.SingleNestedBlock{
				Description: "Options for applying the resource.",
				Attributes: map[string]schema.Attribute{
					"overwrite": schema.BoolAttribute{
						Optional:    true,
						Description: "Set to true if you want to overwrite existing resource with newer version, same resource title in folder or same resource uid.",
					},
				},
			},
		},
	}
}

func (r *AlertConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// For now, we'll implement a basic configure that logs that this is a simplified implementation
	if req.ProviderData == nil {
		return
	}

	// TODO: Set up the actual client when ready for real implementation
	// client, ok := req.ProviderData.(*common.Client)
	// if !ok {
	//     resp.Diagnostics.AddError("Unexpected configure type", "Expected *common.Client")
	//     return
	// }
}

func (r *AlertConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"AlertConfig Create Not Implemented",
		"AlertConfig resource creation is not yet implemented. This is a simplified placeholder implementation.",
	)
}

func (r *AlertConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError(
		"AlertConfig Read Not Implemented",
		"AlertConfig resource reading is not yet implemented. This is a simplified placeholder implementation.",
	)
}

func (r *AlertConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"AlertConfig Update Not Implemented",
		"AlertConfig resource updating is not yet implemented. This is a simplified placeholder implementation.",
	)
}

func (r *AlertConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"AlertConfig Delete Not Implemented",
		"AlertConfig resource deletion is not yet implemented. This is a simplified placeholder implementation.",
	)
}

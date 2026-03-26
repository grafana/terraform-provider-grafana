package appplatform

import (
	"context"
	"encoding/json"

	v2 "github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v2"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DashboardV2StableSpecModel is a model for the dashboard v2 spec.
type DashboardV2StableSpecModel struct {
	JSON  jsontypes.Normalized `tfsdk:"json"`
	Title types.String         `tfsdk:"title"`
	Tags  types.List           `tfsdk:"tags"`
}

// DashboardV2Stable creates a new Grafana Dashboard v2 resource.
func DashboardV2Stable() NamedResource {
	return NewNamedResource[*v2.Dashboard, *v2.DashboardList](
		common.CategoryGrafanaApps,
		ResourceConfig[*v2.Dashboard]{
			Kind: v2.DashboardKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana dashboards using the v2 schema (Dynamic Dashboards).",
				MarkdownDescription: `
Manages Grafana dashboards using the v2 (Dynamic Dashboards) schema.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/#new-dashboard-apis)
	`,
				SpecAttributes: map[string]schema.Attribute{
					"json": schema.StringAttribute{
						Required:    true,
						Description: "The JSON representation of the dashboard v2 spec.",
						CustomType:  jsontypes.NormalizedType{},
					},
					"title": schema.StringAttribute{
						Optional:    true,
						Description: "The title of the dashboard. If not set, the title will be derived from the JSON spec.",
					},
					"tags": schema.ListAttribute{
						Optional:    true,
						Description: "The tags of the dashboard. If not set, the tags will be derived from the JSON spec.",
						ElementType: types.StringType,
					},
				},
				OptionsAttributes: map[string]schema.Attribute{
					"allow_ui_updates": schema.BoolAttribute{
						Optional:    true,
						Description: "Set to true to allow editing the resource from the Grafana UI. By default, resources managed by Terraform cannot be edited in the UI. Enabling this option will cause divergence between the Terraform configuration and the resource in Grafana.",
					},
				},
			},
			SpecParser: func(ctx context.Context, spec types.Object, dst *v2.Dashboard) diag.Diagnostics {
				var data DashboardV2StableSpecModel
				if diag := spec.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}

				var res v2.DashboardSpec
				if diag := data.JSON.Unmarshal(&res); diag.HasError() {
					return diag
				}

				if !data.Title.IsNull() && !data.Title.IsUnknown() {
					res.Title = data.Title.ValueString()
				}

				if tags, ok := getTagsFromV2StableModel(data); ok {
					res.Tags = tags
				}

				if err := dst.SetSpec(res); err != nil {
					return diag.Diagnostics{
						diag.NewErrorDiagnostic("failed to set spec", err.Error()),
					}
				}

				return diag.Diagnostics{}
			},
			SpecSaver: func(ctx context.Context, obj *v2.Dashboard, dst *ResourceModel) diag.Diagnostics {
				jsonBytes, err := json.Marshal(obj.Spec)
				if err != nil {
					return diag.Diagnostics{
						diag.NewErrorDiagnostic("failed to marshal dashboard v2 spec", err.Error()),
					}
				}

				var data DashboardV2StableSpecModel
				if diag := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}
				data.JSON = jsontypes.NewNormalizedValue(string(jsonBytes))

				// SpecSaver is only called during import — always populate title and tags
				// from the spec so that imported state reflects the actual resource.
				if obj.Spec.Title != "" {
					data.Title = types.StringValue(obj.Spec.Title)
				} else {
					data.Title = types.StringNull()
				}

				if len(obj.Spec.Tags) > 0 {
					tags, diags := types.ListValueFrom(ctx, types.StringType, obj.Spec.Tags)
					if diags.HasError() {
						return diags
					}
					data.Tags = tags
				} else {
					data.Tags = types.ListNull(types.StringType)
				}

				spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
					"json":  jsontypes.NormalizedType{},
					"title": types.StringType,
					"tags":  types.ListType{ElemType: types.StringType},
				}, &data)
				if diags.HasError() {
					return diags
				}
				dst.Spec = spec

				return diag.Diagnostics{}
			},
		})
}

func getTagsFromV2StableModel(data DashboardV2StableSpecModel) ([]string, bool) {
	if data.Tags.IsNull() || data.Tags.IsUnknown() {
		return nil, false
	}

	tags := make([]string, 0, len(data.Tags.Elements()))
	for _, tag := range data.Tags.Elements() {
		if tag, ok := tag.(types.String); ok {
			tags = append(tags, tag.ValueString())
		}
	}

	return tags, true
}

package appplatform

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v1alpha1"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DashboardSpecModel is a model for the dashboard spec.
type DashboardSpecModel struct {
	JSON  jsontypes.Normalized `tfsdk:"json"`
	Title types.String         `tfsdk:"title"`
	Tags  types.List           `tfsdk:"tags"`
}

// Dashboard creates a new Grafana Dashboard resource.
func Dashboard() resource.Resource {
	return NewResource[*v1alpha1.Dashboard, *v1alpha1.DashboardList](ResourceConfig[*v1alpha1.Dashboard]{
		Kind: v1alpha1.DashboardKind(),
		Schema: ResourceSpecSchema{
			Description: "Manages Grafana dashboards.",
			MarkdownDescription: `
Manages Grafana dashboards via the new Grafana App Platform API. This resource is currently **EXPERIMENTAL** and may be subject to change. It requires a development build of Grafana with specific feature flags enabled.
	`,
			DeprecationMessage: "This resource is currently EXPERIMENTAL and may be subject to change.",
			SpecAttributes: map[string]schema.Attribute{
				"json": schema.StringAttribute{
					Required:    true,
					Description: "The JSON representation of the dashboard spec.",
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
		},
		SpecParser: func(ctx context.Context, spec types.Object, dst *v1alpha1.Dashboard) diag.Diagnostics {
			var data DashboardSpecModel
			if diag := spec.As(ctx, &data, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			}); diag.HasError() {
				return diag
			}

			var res v1alpha1.DashboardSpec
			if diag := data.JSON.Unmarshal(&res); diag.HasError() {
				return diag
			}

			if !data.Title.IsNull() && !data.Title.IsUnknown() {
				res.Object["title"] = data.Title.ValueString()
			}

			if tags, ok := getTagsFromModel(data); ok {
				res.Object["tags"] = tags
			}

			// HACK: for v0 we need to clean a few fields from the spec,
			// which are not supposed to be set by the user.
			delete(res.Object, "version")

			if err := dst.SetSpec(res); err != nil {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("failed to set spec", err.Error()),
				}
			}

			return diag.Diagnostics{}
		},
		SpecSaver: func(ctx context.Context, obj *v1alpha1.Dashboard, dst *ResourceModel) diag.Diagnostics {
			// HACK: for v0 we need to clean a few fields from the spec,
			// which are not supposed to be set by the user.
			delete(obj.Spec.Object, "version")

			json, err := json.Marshal(obj.Spec.Object)
			if err != nil {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("failed to marshal dashboard spec", err.Error()),
				}
			}

			var data DashboardSpecModel
			if diag := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			}); diag.HasError() {
				return diag
			}
			data.JSON = jsontypes.NewNormalizedValue(string(json))

			// Only copy title from JSON if it is not set in Terraform.
			if !data.Title.IsNull() && !data.Title.IsUnknown() {
				tval := obj.Spec.Object["title"]
				title, ok := tval.(string)
				if !ok {
					return diag.Diagnostics{
						diag.NewErrorDiagnostic("failed to get title", "title is not a string"),
					}
				}
				data.Title = types.StringValue(title)
			} else {
				data.Title = types.StringNull()
			}

			// Only copy tags from JSON if they are not set in Terraform.
			if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
				tags, diags := types.ListValueFrom(ctx, types.StringType, getTagsFromResource(obj))
				if diags.HasError() {
					return diags
				}
				data.Tags = tags
			} else {
				data.Tags = types.ListNull(types.StringType)
			}

			spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"json":  types.StringType,
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

func getTagsFromResource(src *v1alpha1.Dashboard) []string {
	tags, ok := src.Spec.Object["tags"]
	if !ok {
		return nil
	}

	taglist, ok := tags.([]any)
	if !ok {
		return nil
	}

	if taglist == nil {
		return nil
	}

	res := make([]string, 0, len(taglist))
	for _, tag := range taglist {
		if tag, ok := tag.(string); ok {
			res = append(res, tag)
		}
	}

	return res
}

func getTagsFromModel(data DashboardSpecModel) ([]string, bool) {
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

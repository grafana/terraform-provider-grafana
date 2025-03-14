package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/playlist/pkg/apis/playlist/v0alpha1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// TODO: validate type.
// const (
// 	PlaylistItemTypeDashboardByTag PlaylistItemType = "dashboard_by_tag"
// 	PlaylistItemTypeDashboardByUid PlaylistItemType = "dashboard_by_uid"
// 	PlaylistItemTypeDashboardById  PlaylistItemType = "dashboard_by_id"
// )

// PlaylistSpecModel is a model for the dashboard spec.
type PlaylistSpecModel struct {
	Title    types.String `tfsdk:"title"`
	Interval types.String `tfsdk:"interval"`
	Items    types.List   `tfsdk:"items"`
}

type PlaylistItemModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// PlaylistItemType is the type of the playlist item.
var PlaylistItemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
	},
}

// Playlist creates a new Grafana Playlist resource.
func Playlist() resource.Resource {
	return NewResource(ResourceConfig[*v0alpha1.Playlist, *v0alpha1.PlaylistList, v0alpha1.PlaylistSpec]{
		Kind: v0alpha1.PlaylistKind(),
		Schema: ResourceSpecSchema{
			Description: "Manages Grafana playlists.",
			MarkdownDescription: `
Manages Grafana playlists.
	`,
			SpecAttributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{
					Required:    true,
					Description: "The title of the playlist. If not set, the title will be derived from the JSON spec.",
				},
				"interval": schema.StringAttribute{
					Optional:    true,
					Description: "The interval of the playlist.",
				},
				"items": schema.ListAttribute{
					Required:    true,
					Description: "The items of the playlist.",
					ElementType: PlaylistItemType,
				},
			},
		},
		SpecParser: func(ctx context.Context, src types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			var data PlaylistSpecModel
			if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			}); diag.HasError() {
				return diag
			}

			var res v0alpha1.PlaylistSpec
			if !data.Title.IsNull() && !data.Title.IsUnknown() {
				res.Title = data.Title.ValueString()
			}

			if !data.Interval.IsNull() && !data.Interval.IsUnknown() {
				res.Interval = data.Interval.ValueString()
			}

			if !data.Items.IsNull() && !data.Items.IsUnknown() {
				res.Items = make([]v0alpha1.PlaylistItem, 0, len(data.Items.Elements()))

				for _, item := range data.Items.Elements() {
					obj, ok := item.(types.Object)
					if !ok {
						return diag.Diagnostics{
							diag.NewErrorDiagnostic("failed to parse playlist item", "item is not a PlaylistItemModel"),
						}
					}

					var im PlaylistItemModel
					if diag := obj.As(ctx, &im, basetypes.ObjectAsOptions{
						UnhandledNullAsEmpty:    true,
						UnhandledUnknownAsEmpty: true,
					}); diag.HasError() {
						return diag
					}

					res.Items = append(res.Items, v0alpha1.PlaylistItem{
						// TODO: validate type.
						Type:  v0alpha1.PlaylistItemType(im.Type.ValueString()),
						Value: im.Value.ValueString(),
					})
				}
			}

			if err := dst.SetSpec(res); err != nil {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("failed to set spec", err.Error()),
				}
			}

			return diag.Diagnostics{}
		},
		SpecSaver: func(ctx context.Context, src *v0alpha1.Playlist, dst *ResourceModel) diag.Diagnostics {
			var data PlaylistSpecModel
			if diag := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			}); diag.HasError() {
				return diag
			}

			data.Title = types.StringValue(src.Spec.Title)
			data.Interval = types.StringValue(src.Spec.Interval)

			ims := make([]PlaylistItemModel, 0, len(src.Spec.Items))
			for _, item := range src.Spec.Items {
				ims = append(ims, PlaylistItemModel{
					Type:  types.StringValue(string(item.Type)),
					Value: types.StringValue(item.Value),
				})
			}

			its, diags := types.ListValueFrom(ctx, PlaylistItemType, ims)
			if diags.HasError() {
				return diags
			}
			data.Items = its

			spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"title":    types.StringType,
				"interval": types.StringType,
				"items":    types.ListType{ElemType: PlaylistItemType},
			}, &data)
			if diags.HasError() {
				return diags
			}
			dst.Spec = spec

			return diag.Diagnostics{}
		},
	})
}

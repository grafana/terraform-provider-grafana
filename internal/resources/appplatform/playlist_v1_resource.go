package appplatform

import (
	"context"

	playlistv1 "github.com/grafana/grafana/apps/playlist/pkg/apis/playlist/v1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// PlaylistV1 creates a new Grafana Playlist v1 resource.
func PlaylistV1() NamedResource {
	return NewNamedResource[*playlistv1.Playlist, *playlistv1.PlaylistList](
		common.CategoryGrafanaApps,
		ResourceConfig[*playlistv1.Playlist]{
			Kind: playlistv1.PlaylistKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana playlists.",
				MarkdownDescription: `
Manages Grafana playlists using the new Grafana APIs.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/create-manage-playlists/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/apis/)
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "The title of the playlist.",
					},
					"interval": schema.StringAttribute{
						Optional:    true,
						Description: "The interval of the playlist.",
					},
					"items": schema.ListAttribute{
						Required:    true,
						Description: "The items of the playlist.",
						ElementType: PlaylistItemType,
						Validators: []validator.List{
							PlaylistItemValidator{},
						},
					},
				},
			},
			SpecParser: func(ctx context.Context, src types.Object, dst *playlistv1.Playlist) diag.Diagnostics {
				var data PlaylistSpecModel
				if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}

				res := playlistv1.PlaylistSpec{
					Title: data.Title.ValueString(),
				}

				if !data.Interval.IsNull() && !data.Interval.IsUnknown() {
					res.Interval = data.Interval.ValueString()
				}

				items, diags := parsePlaylistItemsV1(ctx, data.Items)
				if diags.HasError() {
					return diags
				}
				res.Items = items

				if err := dst.SetSpec(res); err != nil {
					return diag.Diagnostics{
						diag.NewErrorDiagnostic("failed to set spec", err.Error()),
					}
				}

				return diag.Diagnostics{}
			},
			SpecSaver: func(ctx context.Context, src *playlistv1.Playlist, dst *ResourceModel) diag.Diagnostics {
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

func parsePlaylistItemsV1(ctx context.Context, src types.List) ([]playlistv1.PlaylistItem, diag.Diagnostics) {
	if src.IsNull() || src.IsUnknown() {
		return []playlistv1.PlaylistItem{}, diag.Diagnostics{}
	}

	res := make([]playlistv1.PlaylistItem, 0, len(src.Elements()))
	for _, item := range src.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			return nil, diag.Diagnostics{
				diag.NewErrorDiagnostic("failed to parse playlist item", "item is not a PlaylistItemModel"),
			}
		}

		var im PlaylistItemModel
		if diag := obj.As(ctx, &im, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diag.HasError() {
			return nil, diag
		}

		res = append(res, playlistv1.PlaylistItem{
			Type:  playlistv1.PlaylistPlaylistItemType(im.Type.ValueString()),
			Value: im.Value.ValueString(),
		})
	}

	return res, nil
}

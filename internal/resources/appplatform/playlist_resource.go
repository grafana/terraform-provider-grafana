package appplatform

import (
	"context"
	"fmt"
	"slices"

	"github.com/grafana/grafana/apps/playlist/pkg/apis/playlist/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// PlaylistSpecModel is a model for the playlist spec.
type PlaylistSpecModel struct {
	Title    types.String `tfsdk:"title"`
	Interval types.String `tfsdk:"interval"`
	Items    types.List   `tfsdk:"items"`
}

// PlaylistItemModel is a model for the playlist item.
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
func Playlist() NamedResource {
	return NewNamedResource[*v0alpha1.Playlist, *v0alpha1.PlaylistList](
		common.CategoryGrafanaApps,
		ResourceConfig[*v0alpha1.Playlist]{
			Kind: v0alpha1.PlaylistKind(),
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
			SpecParser: func(ctx context.Context, src types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
				var data PlaylistSpecModel
				if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}

				res := v0alpha1.PlaylistSpec{
					Title: data.Title.ValueString(),
				}

				if !data.Interval.IsNull() && !data.Interval.IsUnknown() {
					res.Interval = data.Interval.ValueString()
				}

				items, diags := parsePlaylistItems(ctx, data.Items)
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

// KnownPlaylistItemTypeValues is a list of known playlist item types.
var KnownPlaylistItemTypeValues = []string{
	string(v0alpha1.PlaylistItemTypeDashboardByTag),
	string(v0alpha1.PlaylistItemTypeDashboardByUid),
	string(v0alpha1.PlaylistItemTypeDashboardById),
}

// PlaylistItemValidator is a validator for the playlist item.
// It ensures that the playlist item type is one of the known values.
type PlaylistItemValidator struct{}

// Description returns the description of the validator.
func (v PlaylistItemValidator) Description(_ context.Context) string {
	return fmt.Sprintf("playlist item must be one of %v", KnownPlaylistItemTypeValues)
}

// MarkdownDescription returns the markdown description of the validator.
func (v PlaylistItemValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateList validates the playlist item.
// It ensures that the playlist item type is one of the known values.
func (v PlaylistItemValidator) ValidateList(
	ctx context.Context, req validator.ListRequest, resp *validator.ListResponse,
) {
	items, diags := parsePlaylistItems(ctx, req.ConfigValue)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	for _, item := range items {
		if !slices.Contains(KnownPlaylistItemTypeValues, string(item.Type)) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				v.Description(ctx),
				fmt.Sprintf("invalid playlist item type: %s, must be one of %v", item.Type, KnownPlaylistItemTypeValues),
			)
		}
	}
}

func parsePlaylistItems(ctx context.Context, src types.List) ([]v0alpha1.PlaylistItem, diag.Diagnostics) {
	if src.IsNull() || src.IsUnknown() {
		return []v0alpha1.PlaylistItem{}, diag.Diagnostics{}
	}

	res := make([]v0alpha1.PlaylistItem, 0, len(src.Elements()))
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

		res = append(res, v0alpha1.PlaylistItem{
			Type:  v0alpha1.PlaylistItemType(im.Type.ValueString()),
			Value: im.Value.ValueString(),
		})
	}

	return res, nil
}

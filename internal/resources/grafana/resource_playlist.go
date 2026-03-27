package grafana

import (
	"context"
	"errors"
	"sort"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/playlists"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &playlistResource{}
	_ resource.ResourceWithConfigure   = &playlistResource{}
	_ resource.ResourceWithImportState = &playlistResource{}

	resourcePlaylistName = "grafana_playlist"
	resourcePlaylistID   = orgResourceIDString("uid")
)

// playlistItemAttrTypes is the attribute type map for a single item in the item set.
var playlistItemAttrTypes = map[string]attr.Type{
	"id":    types.StringType,
	"order": types.Int64Type,
	"type":  types.StringType,
	"value": types.StringType,
}

func resourcePlaylist() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourcePlaylistName,
		resourcePlaylistID,
		&playlistResource{},
	).
		WithLister(listerFunctionOrgResource(listPlaylists)).
		WithPreferredResourceNameField("name")
}

func listPlaylists(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	resp, err := client.Playlists.SearchPlaylists(playlists.NewSearchPlaylistsParams())
	if err != nil {
		return nil, err
	}

	for _, playlist := range resp.Payload {
		ids = append(ids, MakeOrgResourceID(orgID, playlist.UID))
	}

	return ids, nil
}

type playlistItemModel struct {
	ID    types.String `tfsdk:"id"`
	Order types.Int64  `tfsdk:"order"`
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

type playlistResourceModel struct {
	ID       types.String `tfsdk:"id"`
	OrgID    types.String `tfsdk:"org_id"`
	Name     types.String `tfsdk:"name"`
	Interval types.String `tfsdk:"interval"`
	Item     types.Set    `tfsdk:"item"`
}

type playlistResource struct {
	basePluginFrameworkResource
}

func (r *playlistResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourcePlaylistName
}

func (r *playlistResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages Grafana playlists.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/create-manage-playlists/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/playlist/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Organization ID. If not set, the Org ID defined in the provider block will be used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					&orgIDAttributePlanModifier{},
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the playlist.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"interval": schema.StringAttribute{
				Required: true,
			},
		},
		Blocks: map[string]schema.Block{
			"item": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"order": schema.Int64Attribute{
							Required: true,
						},
						"type": schema.StringAttribute{
							Optional: true,
						},
						"value": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func (r *playlistResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playlistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	items, diags := expandPlaylistItemsFromModel(ctx, plan.Item)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := models.CreatePlaylistCommand{
		Name:     plan.Name.ValueString(),
		Interval: plan.Interval.ValueString(),
		Items:    items,
	}

	createResp, err := client.Playlists.CreatePlaylist(&cmd)
	if err != nil {
		resp.Diagnostics.AddError("Error creating playlist", err.Error())
		return
	}

	id := createResp.Payload.UID
	if id == "" && createResp.Payload.ID != 0 {
		id = strconv.FormatInt(createResp.Payload.ID, 10)
	}
	if id == "" {
		resp.Diagnostics.AddError("Error creating playlist", "API response did not include a playlist UID or numeric ID")
		return
	}
	plan.ID = types.StringValue(MakeOrgResourceID(orgID, id))

	readData, diags := r.read(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *playlistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playlistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, state.ID.ValueString())
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

// clientOrgAndPlaylistUID parses the Terraform resource id and returns a Grafana client
// scoped to the resource org and the playlist UID.
func (r *playlistResource) clientOrgAndPlaylistUID(id string) (*goapi.GrafanaHTTPAPI, int64, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	client, orgID, split, parseErr := r.clientFromExistingOrgResource(resourcePlaylistID, id)
	if parseErr != nil {
		diags.AddError("Failed to parse resource ID", parseErr.Error())
		return nil, 0, "", diags
	}
	if len(split) == 0 {
		diags.AddError("Invalid resource ID", "Resource ID has no parts")
		return nil, 0, "", diags
	}
	uid, ok := split[0].(string)
	if !ok || uid == "" {
		diags.AddError("Invalid resource ID", "Playlist UID is missing or invalid")
		return nil, 0, "", diags
	}
	return client, orgID, uid, diags
}

func (r *playlistResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan playlistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, uid, diags := r.clientOrgAndPlaylistUID(plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, diags := expandPlaylistItemsFromModel(ctx, plan.Item)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := models.UpdatePlaylistCommand{
		Name:     plan.Name.ValueString(),
		Interval: plan.Interval.ValueString(),
		Items:    items,
	}

	_, err := client.Playlists.UpdatePlaylist(uid, &cmd)
	if err != nil {
		resp.Diagnostics.AddError("Error updating playlist", err.Error())
		return
	}

	readData, diags := r.read(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *playlistResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playlistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, uid, diags := r.clientOrgAndPlaylistUID(state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := client.Playlists.DeletePlaylist(uid)
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting playlist", err.Error())
		return
	}
}

func (r *playlistResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Playlist not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *playlistResource) read(ctx context.Context, id string) (*playlistResourceModel, diag.Diagnostics) {
	client, orgID, uid, diags := r.clientOrgAndPlaylistUID(id)
	if diags.HasError() {
		return nil, diags
	}

	resp, err := client.Playlists.GetPlaylist(uid)
	// In Grafana 9.0+, if the playlist doesn't exist, the API returns an empty playlist but not a notfound error
	if resp != nil && resp.GetPayload().ID == 0 && resp.GetPayload().UID == "" {
		err = errors.New(common.NotFoundError)
	}
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Error reading playlist", err.Error())
		return nil, diags
	}

	playlist := resp.Payload
	itemsResp, err := client.Playlists.GetPlaylistItems(uid)
	if err != nil {
		diags.AddError("Error getting playlist items", err.Error())
		return nil, diags
	}

	itemSet, itemDiags := flattenPlaylistItemsToSet(ctx, itemsResp.Payload)
	diags.Append(itemDiags...)
	if diags.HasError() {
		return nil, diags
	}

	return &playlistResourceModel{
		ID:       types.StringValue(MakeOrgResourceID(orgID, uid)),
		OrgID:    types.StringValue(strconv.FormatInt(orgID, 10)),
		Name:     types.StringValue(playlist.Name),
		Interval: types.StringValue(playlist.Interval),
		Item:     itemSet,
	}, diags
}

func expandPlaylistItemsFromModel(ctx context.Context, set types.Set) ([]*models.PlaylistItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}

	var elems []playlistItemModel
	diags.Append(set.ElementsAs(ctx, &elems, false)...)
	if diags.HasError() {
		return nil, diags
	}

	items := make([]*models.PlaylistItem, 0, len(elems))
	for _, e := range elems {
		p := &models.PlaylistItem{
			Order: e.Order.ValueInt64(),
		}
		if !e.Type.IsNull() {
			p.Type = e.Type.ValueString()
		}
		if !e.Value.IsNull() {
			p.Value = e.Value.ValueString()
		}
		items = append(items, p)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Order < items[j].Order
	})
	return items, diags
}

func flattenPlaylistItemsToSet(ctx context.Context, items []*models.PlaylistItem) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(items) == 0 {
		set, setDiags := types.SetValue(types.ObjectType{AttrTypes: playlistItemAttrTypes}, nil)
		diags.Append(setDiags...)
		return set, diags
	}

	elems := make([]attr.Value, 0, len(items))
	for i, item := range items {
		order := item.Order
		if order == 0 {
			order = int64(i + 1)
		}
		idAttr := types.StringNull()
		if item.ID != 0 {
			idAttr = types.StringValue(strconv.FormatInt(item.ID, 10))
		}
		obj, objDiags := types.ObjectValue(playlistItemAttrTypes, map[string]attr.Value{
			"id":    idAttr,
			"order": types.Int64Value(order),
			"type":  types.StringValue(item.Type),
			"value": types.StringValue(item.Value),
		})
		diags.Append(objDiags...)
		if diags.HasError() {
			return types.SetNull(types.ObjectType{AttrTypes: playlistItemAttrTypes}), diags
		}
		elems = append(elems, obj)
	}

	set, setDiags := types.SetValue(types.ObjectType{AttrTypes: playlistItemAttrTypes}, elems)
	diags.Append(setDiags...)
	return set, diags
}

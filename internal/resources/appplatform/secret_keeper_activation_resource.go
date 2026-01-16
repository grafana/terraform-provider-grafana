package appplatform

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/grafana/grafana/apps/secret/pkg/apis/secret/v1beta1"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const SystemKeeperName = "system"

type keeperActivationResource struct {
	client *sdkresource.NamespacedClient[*v1beta1.Keeper, *v1beta1.KeeperList]
}

type keeperActivationModel struct {
	ID       types.String `tfsdk:"id"`
	Metadata types.Object `tfsdk:"metadata"`
}

func KeeperActivation() NamedResource {
	return NamedResource{
		Resource: &keeperActivationResource{},
		Name:     "grafana_apps_secret_keeper_activation_v1beta1",
		Category: common.CategoryGrafanaEnterprise,
	}
}

func (r *keeperActivationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_apps_secret_keeper_activation_v1beta1"
}

func (r *keeperActivationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Sets the active keeper for a namespace. Only one keeper can be active at a time.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the resource derived from the keeper name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": secretMetadataBlock(DNS1123SubdomainValidator{}),
		},
	}
}

func (r *keeperActivationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v1beta1.KeeperKind())
	if err != nil {
		resp.Diagnostics.AddError("Error creating Grafana App Platform API client", err.Error())
		return
	}

	ns, errMsg := namespaceForClient(client.GrafanaOrgID, client.GrafanaStackID)
	if errMsg != "" {
		resp.Diagnostics.AddError("Error creating Grafana App Platform API client", errMsg)
		return
	}

	r.client = sdkresource.NewNamespaced(sdkresource.NewTypedClient[*v1beta1.Keeper, *v1beta1.KeeperList](rcli, v1beta1.KeeperKind()), ns)
}

func (r *keeperActivationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data keeperActivationModel
	if diag := req.Config.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	uid, diag := metadataUID(ctx, data.Metadata)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if err := r.activateKeeper(ctx, uid); err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionCreate, uid, "grafana_apps_secret_keeper_activation_v1beta1", err)...)
		return
	}

	data.ID = types.StringValue(uid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keeperActivationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data keeperActivationModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	uid, diag := metadataUID(ctx, data.Metadata)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if _, err := r.client.Get(ctx, uid); err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionRead, uid, "grafana_apps_secret_keeper_activation_v1beta1", err)...)
		return
	}

	data.ID = types.StringValue(uid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keeperActivationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data keeperActivationModel
	if diag := req.Config.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	uid, diag := metadataUID(ctx, data.Metadata)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if err := r.activateKeeper(ctx, uid); err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionUpdate, uid, "grafana_apps_secret_keeper_activation_v1beta1", err)...)
		return
	}

	data.ID = types.StringValue(uid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keeperActivationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.activateKeeper(ctx, SystemKeeperName); err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionDelete, SystemKeeperName, "grafana_apps_secret_keeper_activation_v1beta1", err)...)
		return
	}
}

func (r *keeperActivationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := keeperActivationModel{
		Metadata: emptyMetadataObject(),
	}

	meta, diag := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"uuid":        types.StringType,
		"uid":         types.StringType,
		"folder_uid":  types.StringType,
		"version":     types.StringType,
		"url":         types.StringType,
		"annotations": types.MapType{ElemType: types.StringType},
	}, ResourceMetadataModel{
		UID:         types.StringValue(req.ID),
		Annotations: types.MapNull(types.StringType),
	})
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	data.Metadata = meta
	data.ID = types.StringValue(req.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keeperActivationResource) activateKeeper(ctx context.Context, name string) error {
	body := io.NopCloser(strings.NewReader("{}"))
	_, err := r.client.SubresourceRequest(ctx, name, sdkresource.CustomRouteRequestOptions{
		Path: "activate",
		Verb: "POST",
		Body: body,
	})
	return err
}

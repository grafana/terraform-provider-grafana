package appplatform

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/secret/pkg/apis/secret/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const SystemKeeperName = "system"

type keeperActivationResource struct {
	// typedClient is the namespace-agnostic client; namespaced clients are derived from it
	// per operation so activation can target a specific org via metadata.org_id.
	typedClient *sdkresource.TypedClient[*v1beta1.Keeper, *v1beta1.KeeperList]
	// defaultClient targets the provider-level namespace (from the provider's org_id/stack_id).
	defaultClient *sdkresource.NamespacedClient[*v1beta1.Keeper, *v1beta1.KeeperList]
	// providerStackID is the provider-level Grafana Cloud stack ID (0 for self-hosted). When
	// set, per-resource org_id overrides are rejected because they only apply to self-hosted orgs.
	providerStackID int64
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

	if client.GrafanaAppPlatformAPI == nil {
		resp.Diagnostics.AddError(
			"Grafana App Platform API client not configured",
			"The grafana provider must be configured with 'url' and 'auth' to use App Platform resources. "+
				"Please check your provider configuration.",
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

	r.typedClient = sdkresource.NewTypedClient[*v1beta1.Keeper, *v1beta1.KeeperList](rcli, v1beta1.KeeperKind())
	r.defaultClient = sdkresource.NewNamespaced(r.typedClient, ns)
	r.providerStackID = client.GrafanaStackID
}

// clientForOrg resolves the namespaced client for an explicit per-resource org ID override.
// orgID <= 0 selects the provider-level default client. A non-zero override is only valid
// for self-hosted Grafana; on Grafana Cloud it is rejected rather than silently ignored.
func (r *keeperActivationResource) clientForOrg(orgID int64) (*sdkresource.NamespacedClient[*v1beta1.Keeper, *v1beta1.KeeperList], diag.Diagnostics) {
	var diags diag.Diagnostics
	if orgID <= 0 {
		return r.defaultClient, diags
	}
	if r.providerStackID > 0 {
		diags.AddAttributeError(
			path.Root("metadata").AtName("org_id"),
			"Invalid metadata.org_id",
			"metadata.org_id targets a self-hosted Grafana organization, but the provider is configured for a "+
				"Grafana Cloud stack (stack_id). Remove metadata.org_id, or configure the provider for a self-hosted instance.",
		)
		return nil, diags
	}
	return sdkresource.NewNamespaced(r.typedClient, claims.OrgNamespaceFormatter(orgID)), diags
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

	cli, diags := r.clientForOrg(orgIDFromMetadata(data.Metadata))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.activateKeeper(ctx, cli, uid); err != nil {
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

	cli, diags := r.clientForOrg(orgIDFromMetadata(data.Metadata))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := cli.Get(ctx, uid); err != nil {
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

	cli, diags := r.clientForOrg(orgIDFromMetadata(data.Metadata))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.activateKeeper(ctx, cli, uid); err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionUpdate, uid, "grafana_apps_secret_keeper_activation_v1beta1", err)...)
		return
	}

	data.ID = types.StringValue(uid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keeperActivationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data keeperActivationModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	cli, diags := r.clientForOrg(orgIDFromMetadata(data.Metadata))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.activateKeeper(ctx, cli, SystemKeeperName); err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionDelete, SystemKeeperName, "grafana_apps_secret_keeper_activation_v1beta1", err)...)
		return
	}
}

func (r *keeperActivationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import IDs may be either "<uid>" (provider-default org) or "<orgID>:<uid>" to import an
	// activation that lives in a specific self-hosted organization.
	orgID, name := splitImportID(req.ID)

	orgIDValue := types.Int64Null()
	if orgID > 0 {
		orgIDValue = types.Int64Value(orgID)
	}

	data := keeperActivationModel{
		Metadata: emptyMetadataObject(),
	}

	meta, diag := types.ObjectValueFrom(ctx, secretMetadataAttrTypes, secretMetadataModel{
		UID:         types.StringValue(name),
		OrgID:       orgIDValue,
		Annotations: types.MapNull(types.StringType),
	})
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	data.Metadata = meta
	data.ID = types.StringValue(name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keeperActivationResource) activateKeeper(ctx context.Context, cli *sdkresource.NamespacedClient[*v1beta1.Keeper, *v1beta1.KeeperList], name string) error {
	body := io.NopCloser(strings.NewReader("{}"))
	_, err := cli.SubresourceRequest(ctx, name, sdkresource.CustomRouteRequestOptions{
		Path: "activate",
		Verb: "POST",
		Body: body,
	})
	return err
}

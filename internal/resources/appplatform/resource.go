package appplatform

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// ResourceModel is a Terraform model for a Grafana resource.
type ResourceModel struct {
	Metadata types.Object `tfsdk:"metadata"`
	Spec     types.Object `tfsdk:"spec"`
	Options  types.Object `tfsdk:"options"`
}

// ResourceMetadataModel is a Terraform model for the metadata of a Grafana resource.
type ResourceMetadataModel struct {
	UUID      types.String `tfsdk:"uuid"`
	UID       types.String `tfsdk:"uid"`
	FolderUID types.String `tfsdk:"folder_uid"`
	Version   types.String `tfsdk:"version"`
	URL       types.String `tfsdk:"url"`
}

// ResourceOptionsModel is a Terraform model for the options of a Grafana resource.
type ResourceOptionsModel struct {
	Overwrite types.Bool `tfsdk:"overwrite"`
}

// ResourceConfig is a configuration for a Grafana resource.
type ResourceConfig[T sdkresource.Object, L sdkresource.ListObject, S any] struct {
	Schema     ResourceSpecSchema
	Kind       sdkresource.Kind
	SpecParser SpecParser[T]
	SpecSaver  SpecSaver[T]
}

// ResourceSpecSchema is the Terraform schema for a Grafana resource spec.
type ResourceSpecSchema struct {
	Description         string
	MarkdownDescription string
	SpecAttributes      map[string]schema.Attribute
}

// Resource is a generic Terraform resource for a Grafana resource.
type Resource[T sdkresource.Object, L sdkresource.ListObject, S any] struct {
	config   ResourceConfig[T, L, S]
	client   *sdkresource.NamespacedClient[T, L]
	clientID string
}

// NewResource creates a new Terraform resource for a Grafana resource.
func NewResource[T sdkresource.Object, L sdkresource.ListObject, S any](cfg ResourceConfig[T, L, S]) resource.Resource {
	return &Resource[T, L, S]{
		config: cfg,
	}
}

// Metadata returns the metadata for the Resource.
func (r *Resource[T, L, S]) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	// TODO: extract & validate in the constructor instead,
	// because we need to make sure that the group has the proper format.
	g := strings.Split(r.config.Kind.Group(), ".")[0]

	resp.TypeName = fmt.Sprintf("grafana_%s_%s_%s", g, strings.ToLower(r.config.Kind.Kind()), r.config.Kind.Version())
}

// Schema returns the schema for the DashboardResource.
func (r *Resource[T, L, S]) Schema(ctx context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	sch := r.config.Schema
	res.Schema = schema.Schema{
		Description:         sch.Description,
		MarkdownDescription: sch.MarkdownDescription,
		Blocks: map[string]schema.Block{
			"metadata": schema.SingleNestedBlock{
				Description: "The metadata of the resource.",
				Attributes: map[string]schema.Attribute{
					// Specified by user
					"uid": schema.StringAttribute{
						Required:    true,
						Description: "The unique identifier of the resource.",
					},
					"folder_uid": schema.StringAttribute{
						Optional:    true,
						Description: "The UID of the folder to save the resource in.",
					},
					//
					// TODO: add labels & annotations
					//
					// Computed by API
					"uuid": schema.StringAttribute{
						Computed:    true,
						Description: "The globally unique identifier of a resource, used by the API for tracking.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"url": schema.StringAttribute{
						Computed:    true,
						Description: "The full URL of the resource.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"version": schema.StringAttribute{
						Computed:    true,
						Description: "The version of the resource.",
					},
				},
			},
			"spec": schema.SingleNestedBlock{
				Description: "The spec of the resource.",
				Attributes:  sch.SpecAttributes,
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

// Configure initializes the Resource.
func (r *Resource[T, L, S]) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times
	// (sometimes when ProviderData is not yet available),
	// we only want to configure once.
	if req.ProviderData == nil {
		return
	}

	// Skip if already configured.
	if r.client != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf(
				"Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData,
			),
		)

		return
	}

	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(r.config.Kind)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Grafana App Platform API client",
			err.Error(),
		)

		return
	}

	var ns string
	switch {
	case client.GrafanaOrgID > 0:
		ns = claims.OrgNamespaceFormatter(client.GrafanaOrgID)
	case client.GrafanaStackID > 0:
		ns = claims.CloudNamespaceFormatter(client.GrafanaStackID)
	default:
		resp.Diagnostics.AddError(
			"Error creating Grafana App Platform API client",
			"Expected either Grafana org ID (for local Grafana) or Grafana stack ID (for Grafana Cloud) to be set",
		)

		return
	}

	r.client = sdkresource.NewNamespaced(sdkresource.NewTypedClient[T, L](rcli, r.config.Kind), ns)
	r.clientID = client.GrafanaAppPlatformAPIClientID
}

// Read reads the Grafana resource.
func (r *Resource[T, L, S]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	obj, ok := r.config.Kind.Schema.ZeroValue().(T)
	if !ok {
		var t T
		resp.Diagnostics.AddError(
			"failed to instantiate resource",
			fmt.Sprintf("invalid type, expected: %T, got: %T", t, r.config.Kind.Schema.ZeroValue()),
		)

		return
	}

	if diag := ParseResourceFromModel(ctx, data, obj, r.config.SpecParser); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// TODO: we currently don't have a use for this, but we might need it in the future,
	// if we end up adding [sdkresource.GetOptions].
	var opts ResourceOptions
	if diag := ParseResourceOptionsFromModel(ctx, data, &opts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	res, err := r.client.Get(ctx, obj.GetName())
	if err != nil {
		resp.Diagnostics.AddError("failed to get resource", err.Error())
		return
	}

	if diag := SaveResourceToModel(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new Grafana resource.
func (r *Resource[T, L, S]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel
	if diag := req.Plan.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	obj, ok := r.config.Kind.Schema.ZeroValue().(T)
	if !ok {
		var t T
		resp.Diagnostics.AddError(
			"failed to instantiate resource",
			fmt.Sprintf("invalid type, expected: %T, got: %T", t, r.config.Kind.Schema.ZeroValue()),
		)

		return
	}

	if err := setManagerProperties(obj, r.clientID); err != nil {
		resp.Diagnostics.AddError("failed to set manager properties", err.Error())
		return
	}

	if diag := ParseResourceFromModel(ctx, data, obj, r.config.SpecParser); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// TODO: we currently don't have a use for this, but we might need it in the future,
	// once we add support for dry-run in [sdkresource.CreateOptions].
	var opts ResourceOptions
	if diag := ParseResourceOptionsFromModel(ctx, data, &opts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	res, err := r.client.Create(ctx, obj, sdkresource.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("failed to create resource", err.Error())
		return
	}

	if diag := SaveResourceToModel(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the Grafana resource.
func (r *Resource[T, L, S]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceModel
	if diag := req.Plan.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	obj, ok := r.config.Kind.Schema.ZeroValue().(T)
	if !ok {
		var t T
		resp.Diagnostics.AddError(
			"failed to instantiate resource",
			fmt.Sprintf("invalid type, expected: %T, got: %T", t, r.config.Kind.Schema.ZeroValue()),
		)

		return
	}

	if diag := ParseResourceFromModel(ctx, data, obj, r.config.SpecParser); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	var opts ResourceOptions
	if diag := ParseResourceOptionsFromModel(ctx, data, &opts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if err := setManagerProperties(obj, r.clientID); err != nil {
		resp.Diagnostics.AddError("failed to set manager properties", err.Error())
		return
	}

	reqopts := sdkresource.UpdateOptions{
		ResourceVersion: obj.GetResourceVersion(),
	}

	if opts.Overwrite {
		reqopts.ResourceVersion = ""
	}

	res, err := r.client.Update(ctx, obj, reqopts)
	if err != nil {
		resp.Diagnostics.AddError("failed to update resource", err.Error())
		return
	}

	if diag := SaveResourceToModel(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the Grafana resource.
func (r *Resource[T, L, S]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	obj, ok := r.config.Kind.Schema.ZeroValue().(T)
	if !ok {
		var t T
		resp.Diagnostics.AddError(
			"failed to instantiate resource",
			fmt.Sprintf("invalid type, expected: %T, got: %T", t, r.config.Kind.Schema.ZeroValue()),
		)

		return
	}

	if diag := ParseResourceFromModel(ctx, data, obj, r.config.SpecParser); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// TODO: we currently don't have a use for this, but we might need it in the future,
	// once we figure out what to pass to [sdkresource.DeleteOptions].
	var opts ResourceOptions
	if diag := ParseResourceOptionsFromModel(ctx, data, &opts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if err := r.client.Delete(ctx, obj.GetName(), sdkresource.DeleteOptions{}); err != nil {
		resp.Diagnostics.AddError("failed to delete resource", err.Error())
		return
	}
}

// ImportState imports the state of the Grafana resource.
func (r *Resource[T, L, S]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	res, err := r.client.Get(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("failed to get resource", err.Error())
		return
	}

	var data ResourceModel
	if diag := SaveResourceToModel(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if diag := r.config.SpecSaver(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	opts, diag := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"overwrite": types.BoolType,
	}, ResourceOptionsModel{
		Overwrite: types.BoolValue(true),
	})
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	data.Options = opts

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// SpecParser is a function that parses a resource spec from a Terraform model.
type SpecParser[T sdkresource.Object] func(ctx context.Context, src types.Object, dst T) diag.Diagnostics

// ParseResourceFromModel parses a resource model into a resource.
func ParseResourceFromModel[T sdkresource.Object](
	ctx context.Context, src ResourceModel, dst T, specParser SpecParser[T],
) diag.Diagnostics {
	var (
		diag = make(diag.Diagnostics, 0)
	)

	if diag := SetMetadataFromModel(ctx, src.Metadata, dst); diag.HasError() {
		return diag
	}

	if diag := specParser(ctx, src.Spec, dst); diag.HasError() {
		return diag
	}

	return diag
}

// SpecSaver is a function that saves a resource spec to a Terraform model.
type SpecSaver[T sdkresource.Object] func(ctx context.Context, src T, dst *ResourceModel) diag.Diagnostics

// SaveResourceToModel saves a resource to a Terraform model.
func SaveResourceToModel[T sdkresource.Object](
	ctx context.Context, src T, dst *ResourceModel,
) diag.Diagnostics {
	diag := make(diag.Diagnostics, 0)

	var meta ResourceMetadataModel
	if diag := dst.Metadata.As(ctx, &meta, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	if diag := GetModelFromMetadata(ctx, src, &meta); diag.HasError() {
		return diag
	} else {
		dst.Metadata, diag = types.ObjectValueFrom(
			ctx,
			// TODO: re-use these from the schema.
			map[string]attr.Type{
				"uuid":       types.StringType,
				"uid":        types.StringType,
				"folder_uid": types.StringType,
				"version":    types.StringType,
				"url":        types.StringType,
			},
			meta,
		)

		if diag.HasError() {
			return diag
		}
	}

	return diag
}

// GetModelFromMetadata gets the metadata of a resource from the Terraform model.
func GetModelFromMetadata(
	ctx context.Context, src sdkresource.Object, dst *ResourceMetadataModel,
) diag.Diagnostics {
	diag := make(diag.Diagnostics, 0)

	meta, err := utils.MetaAccessor(src)
	if err != nil {
		diag.AddError("failed to get metadata accessor", err.Error())
		return diag
	}

	if !dst.FolderUID.IsNull() && !dst.FolderUID.IsUnknown() {
		dst.FolderUID = types.StringValue(meta.GetFolder())
	}

	dst.UUID = types.StringValue(string(src.GetUID()))
	dst.UID = types.StringValue(src.GetName())
	dst.Version = types.StringValue(src.GetResourceVersion())
	dst.URL = types.StringValue(meta.GetSelfLink())

	return diag
}

// SetMetadataFromModel sets the metadata of a resource from the Terraform config.
func SetMetadataFromModel(
	ctx context.Context, src types.Object, dst sdkresource.Object,
) diag.Diagnostics {
	diag := make(diag.Diagnostics, 0)
	if src.IsNull() || src.IsUnknown() {
		return diag
	}

	var mod ResourceMetadataModel
	if diag := src.As(ctx, &mod, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		diag.AddError("failed to get metadata accessor", err.Error())
		return diag
	}

	meta.SetUID(k8stypes.UID(mod.UUID.ValueString()))
	meta.SetName(mod.UID.ValueString())
	meta.SetFolder(mod.FolderUID.ValueString())
	meta.SetResourceVersion(mod.Version.ValueString())

	return diag
}

// ResourceOptions is a struct for the options of a Grafana resource.
type ResourceOptions struct {
	Overwrite bool
	Validate  bool
	LintRules []string
}

// ParseResourceOptionsFromModel parses the options of a resource from the Terraform model.
func ParseResourceOptionsFromModel(
	ctx context.Context, src ResourceModel, dst *ResourceOptions,
) diag.Diagnostics {
	diag := make(diag.Diagnostics, 0)
	if src.Options.IsNull() || src.Options.IsUnknown() {
		return diag
	}

	var mod ResourceOptionsModel
	if diag := src.Options.As(ctx, &mod, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	dst.Overwrite = mod.Overwrite.ValueBool()

	return diag
}

// setManagerProperties ensures that the manager properties of a resource are set to the correct values.
// If they already are set correctly, it will do nothing.
func setManagerProperties(obj sdkresource.Object, clientID string) error {
	meta, err := utils.MetaAccessor(obj)
	if err != nil {
		// This should never happen,
		// but we'll add this error for extra safety.
		return fmt.Errorf(
			"failed to configure resource metadata: %w",
			err.Error(),
		)
	}

	var changed bool
	ex, ok := meta.GetManagerProperties()
	if ok {
		if ex.Kind != utils.ManagerKindTerraform {
			ex.Kind = utils.ManagerKindTerraform
			changed = true
		}

		if ex.Identity != clientID {
			ex.Identity = clientID
			changed = true
		}
	}

	if changed {
		meta.SetManagerProperties(utils.ManagerProperties{
			Kind:     utils.ManagerKindTerraform,
			Identity: clientID,
		})
	}

	return nil
}

package appplatform

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	apicommon "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const (
	errNamespaceMissingIDs = "Expected either Grafana org ID (for local Grafana) or Grafana stack ID (for Grafana Cloud) to be set"
)

// ResourceModel is a Terraform model for a Grafana resource.
type ResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Metadata types.Object `tfsdk:"metadata"`
	Spec     types.Object `tfsdk:"spec"`
	Options  types.Object `tfsdk:"options"`
}

// ResourceMetadataModel is a Terraform model for the metadata of a Grafana resource.
type ResourceMetadataModel struct {
	UUID        types.String `tfsdk:"uuid"`
	UID         types.String `tfsdk:"uid"`
	FolderUID   types.String `tfsdk:"folder_uid"`
	Version     types.String `tfsdk:"version"`
	URL         types.String `tfsdk:"url"`
	Annotations types.Map    `tfsdk:"annotations"`
}

// ResourceOptionsModel is a Terraform model for the options of a Grafana resource.
type ResourceOptionsModel struct {
	Overwrite types.Bool `tfsdk:"overwrite"`
}

// ResourceConfig is a configuration for a Grafana resource.
type ResourceConfig[T sdkresource.Object] struct {
	Schema       ResourceSpecSchema
	Kind         sdkresource.Kind
	SpecParser   SpecParser[T]
	SpecSaver    SpecSaver[T]
	SecureParser SecureParser[T]
}

// ResourceSpecSchema is the Terraform schema for a Grafana resource spec.
type ResourceSpecSchema struct {
	Description           string
	MarkdownDescription   string
	DeprecationMessage    string
	SpecAttributes        map[string]schema.Attribute
	SpecBlocks            map[string]schema.Block
	SecureValueAttributes map[string]SecureValueAttribute
}

// SecureValueAttribute defines an input in the secure block.
// Values are rendered as write-only Terraform objects with exactly one of:
// - `create` (maps to InlineSecureValue.Create)
// - `name` (maps to InlineSecureValue.Name)
// APIName controls the destination key used in the API object's `.Secure` field.
// When omitted, Terraform attribute name is used as-is.
type SecureValueAttribute struct {
	Description         string
	MarkdownDescription string
	DeprecationMessage  string
	Required            bool
	Optional            bool
	APIName             string
}

// Resource is a generic Terraform resource for a Grafana resource.
type Resource[T sdkresource.Object, L sdkresource.ListObject] struct {
	config       ResourceConfig[T]
	client       *sdkresource.NamespacedClient[T, L]
	clientID     string
	resourceName string
}

// NamedResource is a Resource with a name and category.
type NamedResource struct {
	Resource resource.Resource
	Name     string
	Category common.ResourceCategory
}

// NewNamedResource creates a new Terraform resource for a Grafana resource.
// The named resource contains the name of the resource and its category.
func NewNamedResource[T sdkresource.Object, L sdkresource.ListObject](
	category common.ResourceCategory, cfg ResourceConfig[T],
) NamedResource {
	return NamedResource{
		Resource: NewResource[T, L](cfg),
		Name:     formatResourceType(cfg.Kind),
		Category: category,
	}
}

// NewResource creates a new Terraform resource for a Grafana resource.
func NewResource[T sdkresource.Object, L sdkresource.ListObject](cfg ResourceConfig[T]) resource.Resource {
	return &Resource[T, L]{
		config:       cfg,
		resourceName: formatResourceType(cfg.Kind),
	}
}

// Metadata returns the metadata for the Resource.
func (r *Resource[T, L]) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.resourceName
}

// Schema returns the schema for the Resource.
func (r *Resource[T, L]) Schema(ctx context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	sch := r.config.Schema

	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "The ID of the resource derived from UUID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
	blocks := map[string]schema.Block{
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
				// TODO: add labels
				//

				"annotations": schema.MapAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "Annotations of the resource.",
				},

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
			Blocks:      sch.SpecBlocks,
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
	}

	if len(sch.SecureValueAttributes) > 0 {
		if r.config.SecureParser == nil {
			res.Diagnostics.AddError(
				"Invalid resource secure configuration",
				"SecureValueAttributes is configured, but SecureParser is nil.",
			)
		}

		secureAttrs, secureDiags := buildSecureValueSchemaAttributes(sch.SecureValueAttributes)
		res.Diagnostics.Append(secureDiags...)

		blocks["secure"] = schema.SingleNestedBlock{
			Description: "Sensitive credentials. Values are write-only and never stored in Terraform state.",
			Attributes:  secureAttrs,
		}
		attrs["secure_version"] = schema.Int64Attribute{
			Optional:    true,
			Description: "Increment this value to trigger re-application of all secure values.",
		}
	} else if r.config.SecureParser != nil {
		res.Diagnostics.AddError(
			"Invalid resource secure configuration",
			"SecureParser is configured, but SecureValueAttributes is empty.",
		)
	}

	res.Schema = schema.Schema{
		Description:         sch.Description,
		MarkdownDescription: sch.MarkdownDescription,
		DeprecationMessage:  sch.DeprecationMessage,
		Attributes:          attrs,
		Blocks:              blocks,
	}
}

// Configure initializes the Resource.
func (r *Resource[T, L]) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	ns, errMsg := namespaceForClient(client.GrafanaOrgID, client.GrafanaStackID)
	if errMsg != "" {
		resp.Diagnostics.AddError("Error creating Grafana App Platform API client", errMsg)
		return
	}

	r.client = sdkresource.NewNamespaced(sdkresource.NewTypedClient[T, L](rcli, r.config.Kind), ns)
	r.clientID = client.GrafanaAppPlatformAPIClientID
}

func namespaceForClient(orgID, stackID int64) (string, string) {
	switch {
	// GrafanaOrgID is 1 by default, so we check first if the stack ID is set
	// and only then fall back to org ID, otherwise GrafanaOrgID would always take precedence
	// unless it is explicitly set to 0.
	case stackID > 0:
		return claims.CloudNamespaceFormatter(stackID), ""
	case orgID > 0:
		return claims.OrgNamespaceFormatter(orgID), ""
	default:
		return "", errNamespaceMissingIDs
	}
}

// Read reads the Grafana resource.
func (r *Resource[T, L]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data, diags := getResourceModelFromData(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secureVersion := types.Int64Null()
	if r.hasSecureSchema() {
		secureVersion, diags = getSecureVersionFromData(ctx, req.State)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	r.readModel(ctx, data, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(r.setState(ctx, &resp.State, updated, secureVersion)...)
	})
}

func (r *Resource[T, L]) readModel(ctx context.Context, data ResourceModel, resp *resource.ReadResponse, setState func(updated ResourceModel)) {
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
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionRead, obj.GetName(), r.resourceName, err)...)
		return
	}

	if diag := SaveResourceToModel(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	setState(data)
}

// Create creates a new Grafana resource.
func (r *Resource[T, L]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data, diags := getResourceModelFromData(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secureVersion := types.Int64Null()
	if r.hasSecureSchema() {
		secureVersion, diags = getSecureVersionFromData(ctx, req.Plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	r.createModel(ctx, req.Config, data, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(r.setState(ctx, &resp.State, updated, secureVersion)...)
	})
}

func (r *Resource[T, L]) createModel(
	ctx context.Context,
	cfg tfsdk.Config,
	data ResourceModel,
	resp *resource.CreateResponse,
	setState func(updated ResourceModel),
) {
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

	if diag := r.applySecureValues(ctx, cfg, obj, false); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	res, err := r.client.Create(ctx, obj, sdkresource.CreateOptions{})
	if err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionCreate, obj.GetName(), r.resourceName, err)...)
		return
	}

	if diag := SaveResourceToModel(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	setState(data)
}

// Update updates the Grafana resource.
func (r *Resource[T, L]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data, diags := getResourceModelFromData(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secureVersion := types.Int64Null()
	if r.hasSecureSchema() {
		secureVersion, diags = getSecureVersionFromData(ctx, req.Plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	r.updateModel(ctx, req.Config, data, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(r.setState(ctx, &resp.State, updated, secureVersion)...)
	})
}

func (r *Resource[T, L]) updateModel(
	ctx context.Context,
	cfg tfsdk.Config,
	data ResourceModel,
	resp *resource.UpdateResponse,
	setState func(updated ResourceModel),
) {
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

	if diag := r.applySecureValues(ctx, cfg, obj, true); diag.HasError() {
		resp.Diagnostics.Append(diag...)
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
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionUpdate, obj.GetName(), r.resourceName, err)...)
		return
	}

	if diag := SaveResourceToModel(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	setState(data)
}

// Delete deletes the Grafana resource.
func (r *Resource[T, L]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data, diags := getResourceModelFromData(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.deleteModel(ctx, data, resp)
}

func (r *Resource[T, L]) deleteModel(ctx context.Context, data ResourceModel, resp *resource.DeleteResponse) {
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
		if apierrors.IsNotFound(err) {
			return
		}

		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionDelete, obj.GetName(), r.resourceName, err)...)
		return
	}
}

// ImportState imports the state of the Grafana resource.
func (r *Resource[T, L]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.importStateModel(ctx, req, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(r.setState(ctx, &resp.State, updated, types.Int64Null())...)
	})
}

func (r *Resource[T, L]) importStateModel(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
	setState func(updated ResourceModel),
) {
	res, err := r.client.Get(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionRead, req.ID, r.resourceName, err)...)
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

	setState(data)
}

// SpecParser is a function that parses a resource spec from a Terraform model.
type SpecParser[T sdkresource.Object] func(ctx context.Context, src types.Object, dst T) diag.Diagnostics

// SecureParser is a function that parses secure values from Terraform config into the Grafana resource object.
type SecureParser[T sdkresource.Object] func(ctx context.Context, secure types.Object, dst T) diag.Diagnostics

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
				"uuid":        types.StringType,
				"uid":         types.StringType,
				"folder_uid":  types.StringType,
				"version":     types.StringType,
				"url":         types.StringType,
				"annotations": types.MapType{ElemType: types.StringType},
			},
			meta,
		)

		if diag.HasError() {
			return diag
		}
	}

	dst.ID = meta.UUID

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

	if annotations := meta.GetAnnotations(); len(annotations) > 0 {
		dst.Annotations, _ = types.MapValueFrom(ctx, types.StringType, annotations)
	} else {
		dst.Annotations = types.MapNull(types.StringType)
	}

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
		// This should never happen, but we'll add this error for extra safety.
		return fmt.Errorf("failed to configure resource metadata: %w", err)
	}

	ex, found := meta.GetManagerProperties()
	changed := !found
	if found {
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

func formatResourceType(kind sdkresource.Kind) string {
	g := strings.Split(kind.Group(), ".")[0]
	return fmt.Sprintf("grafana_apps_%s_%s_%s", g, strings.ToLower(kind.Kind()), kind.Version())
}

func (r *Resource[T, L]) hasSecureSchema() bool {
	return len(r.config.Schema.SecureValueAttributes) > 0
}

func (r *Resource[T, L]) secureAttrTypes() map[string]attr.Type {
	attrTypes := make(map[string]attr.Type, len(r.config.Schema.SecureValueAttributes))
	for name := range r.config.Schema.SecureValueAttributes {
		attrTypes[name] = secureValueObjectType()
	}

	return attrTypes
}

func secureValueObjectType() attr.Type {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":   types.StringType,
			"create": types.StringType,
		},
	}
}

func (r *Resource[T, L]) nullSecureObject() types.Object {
	return types.ObjectNull(r.secureAttrTypes())
}

func (r *Resource[T, L]) parseSecureValues(ctx context.Context, cfg tfsdk.Config, dst T) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	secureObj := r.nullSecureObject()

	if r.config.SecureParser == nil {
		diags.AddError("failed to parse secure values", "SecureValueAttributes is configured, but SecureParser is nil.")
		return secureObj, diags
	}

	diags.Append(cfg.GetAttribute(ctx, path.Root("secure"), &secureObj)...)
	if diags.HasError() {
		return secureObj, diags
	}

	parserCtx := withSecureAPINameMappings(ctx, r.config.Schema.SecureValueAttributes)
	diags.Append(r.config.SecureParser(parserCtx, secureObj, dst)...)
	return secureObj, diags
}

func (r *Resource[T, L]) applySecureValues(
	ctx context.Context,
	cfg tfsdk.Config,
	dst T,
	reconcileMissingKeys bool,
) diag.Diagnostics {
	var diags diag.Diagnostics
	if !r.hasSecureSchema() {
		return diags
	}

	secureObj, parseDiags := r.parseSecureValues(ctx, cfg, dst)
	diags.Append(parseDiags...)
	if diags.HasError() {
		return diags
	}

	if reconcileMissingKeys {
		if err := applySchemaBasedSecureRemovals(dst, secureObj, r.config.Schema.SecureValueAttributes); err != nil {
			diags.AddError("failed to reconcile secure values", err.Error())
		}
	}

	return diags
}

type secureAPINameContextKey struct{}

func withSecureAPINameMappings(ctx context.Context, attrs map[string]SecureValueAttribute) context.Context {
	if len(attrs) == 0 {
		return ctx
	}

	mappings := make(map[string]string, len(attrs))
	for terraformKey, secureAttr := range attrs {
		mappings[terraformKey] = secureValueAPIName(terraformKey, secureAttr)
	}

	return context.WithValue(ctx, secureAPINameContextKey{}, mappings)
}

func secureValueAPINameFromContext(ctx context.Context, terraformKey string) string {
	mappings, ok := ctx.Value(secureAPINameContextKey{}).(map[string]string)
	if !ok || len(mappings) == 0 {
		return terraformKey
	}

	apiName, found := mappings[terraformKey]
	if !found || apiName == "" {
		return terraformKey
	}

	return apiName
}

func secureValueAPIName(terraformKey string, secureAttr SecureValueAttribute) string {
	if secureAttr.APIName != "" {
		return secureAttr.APIName
	}
	return terraformKey
}

// DefaultSecureParser converts secure fields into InlineSecureValues and writes
// them to dst's Secure map/struct fields.
// Supported field shape for each secure key:
// - object with exactly one of `name` or `create`
func DefaultSecureParser[T sdkresource.Object](ctx context.Context, secure types.Object, dst T) diag.Diagnostics {
	var diags diag.Diagnostics
	if secure.IsNull() || secure.IsUnknown() {
		return diags
	}

	secureValues := make(apicommon.InlineSecureValues)
	for fieldName, fieldValue := range secure.Attributes() {
		parsedValue, shouldSet, fieldDiags := parseInlineSecureValue(fieldName, fieldValue)
		diags.Append(fieldDiags...)
		if fieldDiags.HasError() || !shouldSet {
			continue
		}

		apiName := secureValueAPINameFromContext(ctx, fieldName)
		secureValues[apiName] = parsedValue
	}

	if diags.HasError() || len(secureValues) == 0 {
		return diags
	}

	if err := setDefaultSecureValues(dst, secureValues); err != nil {
		diags.AddError("failed to parse secure values", fmt.Sprintf("failed to set secure values: %s", err.Error()))
		return diags
	}

	return diags
}

func parseInlineSecureValue(fieldName string, fieldValue attr.Value) (apicommon.InlineSecureValue, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if fieldValue == nil || fieldValue.IsNull() || fieldValue.IsUnknown() {
		return apicommon.InlineSecureValue{}, false, diags
	}

	secureObject, ok := fieldValue.(types.Object)
	if !ok {
		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf(
				"secure field %q has unsupported type %T; expected object with `name` or `create`",
				fieldName,
				fieldValue,
			),
		)

		return apicommon.InlineSecureValue{}, false, diags
	}

	attrs := secureObject.Attributes()
	for key := range attrs {
		if key == "name" || key == "create" {
			continue
		}

		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf("secure field %q object contains unsupported key %q; only `name` and `create` are allowed", fieldName, key),
		)
	}
	if diags.HasError() {
		return apicommon.InlineSecureValue{}, false, diags
	}

	nameValue, hasName := attrs["name"]
	createValue, hasCreate := attrs["create"]
	if !hasName || !hasCreate {
		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf("secure field %q object must include both `name` and `create` attributes", fieldName),
		)
		return apicommon.InlineSecureValue{}, false, diags
	}

	name, hasConfiguredName, nameDiags := configuredSecureStringValue(fieldName, "name", nameValue)
	diags.Append(nameDiags...)
	create, hasConfiguredCreate, createDiags := configuredSecureStringValue(fieldName, "create", createValue)
	diags.Append(createDiags...)
	if diags.HasError() {
		return apicommon.InlineSecureValue{}, false, diags
	}

	switch {
	case hasConfiguredName && hasConfiguredCreate:
		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf("secure field %q object must set exactly one of `name` or `create`", fieldName),
		)
		return apicommon.InlineSecureValue{}, false, diags
	case hasConfiguredName:
		if strings.TrimSpace(name) == "" {
			diags.AddError(
				"failed to parse secure values",
				fmt.Sprintf("secure field %q object `name` must not be empty", fieldName),
			)
			return apicommon.InlineSecureValue{}, false, diags
		}
		return apicommon.InlineSecureValue{Name: name}, true, diags
	case hasConfiguredCreate:
		if create == "" {
			diags.AddError(
				"failed to parse secure values",
				fmt.Sprintf("secure field %q object `create` must not be empty", fieldName),
			)
			return apicommon.InlineSecureValue{}, false, diags
		}
		return apicommon.InlineSecureValue{Create: apicommon.NewSecretValue(create)}, true, diags
	default:
		return apicommon.InlineSecureValue{}, false, diags
	}
}

func configuredSecureStringValue(fieldName, attributeName string, value attr.Value) (string, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value == nil || value.IsNull() || value.IsUnknown() {
		return "", false, diags
	}

	stringValue, ok := value.(types.String)
	if !ok {
		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf(
				"secure field %q object `%s` has unsupported type %T; expected string",
				fieldName,
				attributeName,
				value,
			),
		)
		return "", false, diags
	}

	return stringValue.ValueString(), true, diags
}

func setDefaultSecureValues[T sdkresource.Object](dst T, secureValues apicommon.InlineSecureValues) error {
	v := reflect.ValueOf(dst)
	if !v.IsValid() {
		return fmt.Errorf("destination object is invalid")
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return fmt.Errorf("destination object is nil")
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("destination object type %T is not a struct", dst)
	}

	secureField := v.FieldByName("Secure")
	if !secureField.IsValid() || !secureField.CanSet() {
		return fmt.Errorf("destination object type %T does not have a settable Secure field", dst)
	}

	for secureField.Kind() == reflect.Pointer {
		if secureField.IsNil() {
			secureField.Set(reflect.New(secureField.Type().Elem()))
		}
		secureField = secureField.Elem()
	}

	switch secureField.Kind() {
	case reflect.Struct:
		return setStructSecureValues(secureField, secureValues)
	case reflect.Map:
		return setMapSecureValues(secureField, secureValues)
	default:
		return fmt.Errorf(
			"destination object type %T has unsupported Secure field kind %s; expected struct or map",
			dst,
			secureField.Kind(),
		)
	}
}

func setStructSecureValues(v reflect.Value, secureValues apicommon.InlineSecureValues) error {
	indexByKey := make(map[string]int, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			continue
		}

		fieldType := v.Type().Field(i)
		jsonName := jsonFieldName(fieldType)
		if jsonName == "-" {
			continue
		}

		indexByKey[jsonName] = i
	}

	var unknownKeys []string
	for key, value := range secureValues {
		fieldIndex, ok := indexByKey[key]
		if !ok {
			unknownKeys = append(unknownKeys, key)
			continue
		}

		field := v.Field(fieldIndex)
		incoming := reflect.ValueOf(value)
		if incoming.Type().AssignableTo(field.Type()) {
			field.Set(incoming)
			continue
		}
		if incoming.Type().ConvertibleTo(field.Type()) {
			field.Set(incoming.Convert(field.Type()))
			continue
		}

		return fmt.Errorf("secure field %q has unsupported destination type %s", key, field.Type())
	}

	if len(unknownKeys) > 0 {
		sort.Strings(unknownKeys)
		return fmt.Errorf("invalid secure value key: %v", unknownKeys)
	}

	return nil
}

func setMapSecureValues(v reflect.Value, secureValues apicommon.InlineSecureValues) error {
	if v.IsNil() {
		v.Set(reflect.MakeMapWithSize(v.Type(), len(secureValues)))
	}

	keyType := v.Type().Key()
	elemType := v.Type().Elem()

	for key, value := range secureValues {
		mapKey := reflect.ValueOf(key)
		if !mapKey.Type().AssignableTo(keyType) {
			if !mapKey.Type().ConvertibleTo(keyType) {
				return fmt.Errorf("secure map key type %s is not assignable to %s", mapKey.Type(), keyType)
			}
			mapKey = mapKey.Convert(keyType)
		}

		mapValue := reflect.ValueOf(value)
		if !mapValue.Type().AssignableTo(elemType) {
			if !mapValue.Type().ConvertibleTo(elemType) {
				return fmt.Errorf("secure map value type %s is not assignable to %s", mapValue.Type(), elemType)
			}
			mapValue = mapValue.Convert(elemType)
		}

		v.SetMapIndex(mapKey, mapValue)
	}

	return nil
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return field.Name
	}

	return name
}

func buildSecureValueSchemaAttributes(attrs map[string]SecureValueAttribute) (map[string]schema.Attribute, diag.Diagnostics) {
	secureAttrs := make(map[string]schema.Attribute, len(attrs))
	var diags diag.Diagnostics
	apiNameToTerraformKey := make(map[string]string, len(attrs))

	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		secureAttr := attrs[name]
		switch {
		case secureAttr.Required && secureAttr.Optional:
			diags.AddError(
				"Invalid secure value attribute configuration",
				fmt.Sprintf("Secure value attribute %q cannot be both required and optional.", name),
			)
			continue
		case !secureAttr.Required && !secureAttr.Optional:
			secureAttr.Optional = true
		}

		apiName := secureValueAPIName(name, secureAttr)
		if strings.TrimSpace(apiName) == "" {
			diags.AddError(
				"Invalid secure value attribute configuration",
				fmt.Sprintf("Secure value attribute %q has an empty APIName; provide a non-empty value or omit APIName.", name),
			)
			continue
		}

		if existingTerraformKey, exists := apiNameToTerraformKey[apiName]; exists {
			diags.AddError(
				"Invalid secure value attribute configuration",
				fmt.Sprintf(
					"Secure value attributes %q and %q map to the same APIName %q; APIName values must be unique.",
					existingTerraformKey,
					name,
					apiName,
				),
			)
			continue
		}
		apiNameToTerraformKey[apiName] = name

		secureAttrs[name] = schema.SingleNestedAttribute{
			Description:         secureAttr.Description,
			MarkdownDescription: secureAttr.MarkdownDescription,
			DeprecationMessage:  secureAttr.DeprecationMessage,
			Required:            secureAttr.Required,
			Optional:            secureAttr.Optional,
			WriteOnly:           true,
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Optional:    true,
					WriteOnly:   true,
					Description: "Reference an existing secret by name.",
				},
				"create": schema.StringAttribute{
					Optional:    true,
					WriteOnly:   true,
					Description: "Provide a new secret value. This value is write-only and never stored in Terraform state.",
				},
			},
			Validators: []validator.Object{
				objectvalidator.ExactlyOneOf(
					path.MatchRelative().AtName("name"),
					path.MatchRelative().AtName("create"),
				),
			},
		}
	}

	return secureAttrs, diags
}

func configuredSecureAPIKeySet(secure types.Object, attrs map[string]SecureValueAttribute) map[string]struct{} {
	configured := make(map[string]struct{}, len(attrs))
	if secure.IsNull() || secure.IsUnknown() {
		return configured
	}

	for key, value := range secure.Attributes() {
		if !hasConfiguredSecureValue(value) {
			continue
		}

		configured[secureValueAPIName(key, attrs[key])] = struct{}{}
	}

	return configured
}

func hasConfiguredSecureValue(value attr.Value) bool {
	if value == nil || value.IsNull() || value.IsUnknown() {
		return false
	}

	secureObject, ok := value.(types.Object)
	if !ok {
		return false
	}

	for _, nestedValue := range secureObject.Attributes() {
		if nestedValue == nil || nestedValue.IsNull() || nestedValue.IsUnknown() {
			continue
		}

		return true
	}

	return false
}

func applySchemaBasedSecureRemovals[T sdkresource.Object](dst T, secure types.Object, attrs map[string]SecureValueAttribute) error {
	configured := configuredSecureAPIKeySet(secure, attrs)
	removals := make(apicommon.InlineSecureValues)

	for terraformKey, secureAttr := range attrs {
		apiKey := secureValueAPIName(terraformKey, secureAttr)
		if _, found := configured[apiKey]; found {
			continue
		}

		removals[apiKey] = apicommon.InlineSecureValue{Remove: true}
	}

	if len(removals) == 0 {
		return nil
	}

	return setDefaultSecureValues(dst, removals)
}

type resourceData interface {
	GetAttribute(ctx context.Context, path path.Path, target interface{}) diag.Diagnostics
}

type stateData interface {
	SetAttribute(ctx context.Context, path path.Path, val interface{}) diag.Diagnostics
}

func getResourceModelFromData(ctx context.Context, src resourceData) (ResourceModel, diag.Diagnostics) {
	var (
		data  ResourceModel
		diags diag.Diagnostics
	)

	diags.Append(src.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	diags.Append(src.GetAttribute(ctx, path.Root("metadata"), &data.Metadata)...)
	diags.Append(src.GetAttribute(ctx, path.Root("spec"), &data.Spec)...)
	diags.Append(src.GetAttribute(ctx, path.Root("options"), &data.Options)...)

	return data, diags
}

func getSecureVersionFromData(ctx context.Context, src resourceData) (types.Int64, diag.Diagnostics) {
	var (
		secureVersion types.Int64
		diags         diag.Diagnostics
	)

	diags.Append(src.GetAttribute(ctx, path.Root("secure_version"), &secureVersion)...)
	return secureVersion, diags
}

func setBaseState(ctx context.Context, state stateData, data ResourceModel) diag.Diagnostics {
	// IMPORTANT: keep base attributes in sync with ResourceModel fields.
	var diags diag.Diagnostics

	diags.Append(state.SetAttribute(ctx, path.Root("id"), data.ID)...)
	diags.Append(state.SetAttribute(ctx, path.Root("metadata"), data.Metadata)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec"), data.Spec)...)
	diags.Append(state.SetAttribute(ctx, path.Root("options"), data.Options)...)

	return diags
}

func (r *Resource[T, L]) setState(
	ctx context.Context,
	state stateData,
	data ResourceModel,
	secureVersion types.Int64,
) diag.Diagnostics {
	if !r.hasSecureSchema() {
		return setBaseState(ctx, state, data)
	}

	return r.setSecureState(ctx, state, data, secureVersion)
}

func (r *Resource[T, L]) setSecureState(
	ctx context.Context,
	state stateData,
	data ResourceModel,
	secureVersion types.Int64,
) diag.Diagnostics {
	diags := setBaseState(ctx, state, data)
	diags.Append(state.SetAttribute(ctx, path.Root("secure"), r.nullSecureObject())...)
	diags.Append(state.SetAttribute(ctx, path.Root("secure_version"), secureVersion)...)

	return diags
}

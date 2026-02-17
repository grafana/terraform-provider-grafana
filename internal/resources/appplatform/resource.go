package appplatform

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	apicommon "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	Description         string
	MarkdownDescription string
	DeprecationMessage  string
	SpecAttributes      map[string]schema.Attribute
	SpecBlocks          map[string]schema.Block
	SecureAttributes    map[string]schema.Attribute
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

	if len(sch.SecureAttributes) > 0 {
		if r.config.SecureParser == nil {
			res.Diagnostics.AddError(
				"Invalid resource secure configuration",
				"SecureAttributes is configured, but SecureParser is nil.",
			)
		}

		for secureAttrName, secureAttr := range sch.SecureAttributes {
			if !secureAttr.IsWriteOnly() {
				res.Diagnostics.AddError(
					"Invalid secure attribute configuration",
					fmt.Sprintf("Secure attribute %q must set WriteOnly: true.", secureAttrName),
				)
			}
		}

		blocks["secure"] = schema.SingleNestedBlock{
			Description: "Sensitive credentials. Values are write-only and never stored in Terraform state.",
			Attributes:  sch.SecureAttributes,
		}
		attrs["secure_version"] = schema.Int64Attribute{
			Optional:    true,
			Description: "Increment this value to trigger re-application of all secure values.",
		}
	} else if r.config.SecureParser != nil {
		res.Diagnostics.AddError(
			"Invalid resource secure configuration",
			"SecureParser is configured, but SecureAttributes is empty.",
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
	if r.hasSecureSchema() {
		data, diags := getResourceModelFromData(ctx, req.State)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		secureVersion, diags := getSecureVersionFromData(ctx, req.State)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		r.readModel(ctx, data, resp, func(updated ResourceModel) {
			resp.Diagnostics.Append(r.setSecureState(ctx, &resp.State, updated, secureVersion)...)
		})
		return
	}

	var data ResourceModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	r.readModel(ctx, data, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
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
	if r.hasSecureSchema() {
		data, diags := getResourceModelFromData(ctx, req.Plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		secureVersion, diags := getSecureVersionFromData(ctx, req.Plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		r.createModel(ctx, req.Config, data, resp, func(updated ResourceModel) {
			resp.Diagnostics.Append(r.setSecureState(ctx, &resp.State, updated, secureVersion)...)
		})
		return
	}

	var data ResourceModel
	if diag := req.Plan.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	r.createModel(ctx, req.Config, data, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
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

	if r.hasSecureSchema() {
		if diag := r.parseSecureValues(ctx, cfg, obj); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
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
	if r.hasSecureSchema() {
		data, diags := getResourceModelFromData(ctx, req.Plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		secureVersion, diags := getSecureVersionFromData(ctx, req.Plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		r.updateModel(ctx, req.Config, data, resp, func(updated ResourceModel) {
			resp.Diagnostics.Append(r.setSecureState(ctx, &resp.State, updated, secureVersion)...)
		})
		return
	}

	var data ResourceModel
	if diag := req.Plan.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	r.updateModel(ctx, req.Config, data, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
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

	if r.hasSecureSchema() {
		if diag := r.parseSecureValues(ctx, cfg, obj); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
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
	if r.hasSecureSchema() {
		data, diags := getResourceModelFromData(ctx, req.State)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.deleteModel(ctx, data, resp)
		return
	}

	var data ResourceModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
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
	if r.hasSecureSchema() {
		r.importStateModel(ctx, req, resp, func(updated ResourceModel) {
			resp.Diagnostics.Append(r.setSecureState(ctx, &resp.State, updated, types.Int64Null())...)
		})
		return
	}

	r.importStateModel(ctx, req, resp, func(updated ResourceModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
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
	return len(r.config.Schema.SecureAttributes) > 0
}

func (r *Resource[T, L]) secureAttrTypes() map[string]attr.Type {
	attrTypes := make(map[string]attr.Type, len(r.config.Schema.SecureAttributes))
	for name, secureAttr := range r.config.Schema.SecureAttributes {
		attrTypes[name] = secureAttr.GetType()
	}

	return attrTypes
}

func (r *Resource[T, L]) nullSecureObject() types.Object {
	return types.ObjectNull(r.secureAttrTypes())
}

func (r *Resource[T, L]) parseSecureValues(ctx context.Context, cfg tfsdk.Config, dst T) diag.Diagnostics {
	var diags diag.Diagnostics

	if r.config.SecureParser == nil {
		diags.AddError("failed to parse secure values", "SecureAttributes is configured, but SecureParser is nil.")
		return diags
	}

	var secureObj types.Object
	diags.Append(cfg.GetAttribute(ctx, path.Root("secure"), &secureObj)...)
	if diags.HasError() {
		return diags
	}

	diags.Append(r.config.SecureParser(ctx, secureObj, dst)...)
	return diags
}

// DefaultSecureParser converts all non-null string fields from secure into InlineSecureValues
// and writes them to dst's Secure map/struct fields.
func DefaultSecureParser[T sdkresource.Object](ctx context.Context, secure types.Object, dst T) diag.Diagnostics {
	var diags diag.Diagnostics
	if secure.IsNull() || secure.IsUnknown() {
		return diags
	}

	secureValues := make(apicommon.InlineSecureValues)
	for fieldName, fieldValue := range secure.Attributes() {
		if fieldValue.IsNull() || fieldValue.IsUnknown() {
			continue
		}

		stringValue, ok := fieldValue.(types.String)
		if !ok {
			diags.AddError(
				"failed to parse secure values",
				fmt.Sprintf("secure field %q has unsupported type %T; only string secure attributes are supported", fieldName, fieldValue),
			)
			continue
		}

		secureValues[fieldName] = apicommon.InlineSecureValue{
			Create: apicommon.NewSecretValue(stringValue.ValueString()),
		}
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
		return setSecureValuesWithMetaAccessor(dst, secureValues)
	}

	secureField := v.FieldByName("Secure")
	if !secureField.IsValid() || !secureField.CanSet() {
		return setSecureValuesWithMetaAccessor(dst, secureValues)
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
		return setSecureValuesWithMetaAccessor(dst, secureValues)
	}
}

// setSecureValuesWithMetaAccessor is a fallback when the destination object does not expose
// a directly settable Secure field. Keys should already be normalized to API-style names.
func setSecureValuesWithMetaAccessor(dst sdkresource.Object, secureValues apicommon.InlineSecureValues) error {
	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		return fmt.Errorf("failed to get metadata accessor: %w", err)
	}
	if err := meta.SetSecureValues(secureValues); err != nil {
		return err
	}
	return nil
}

func setStructSecureValues(v reflect.Value, secureValues apicommon.InlineSecureValues) error {
	indexByKey := make(map[string]int, v.NumField()*2)

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
		indexByKey[toSnakeCase(jsonName)] = i
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
		normalizedKey := key
		if keyType.Kind() == reflect.String {
			normalizedKey = toLowerCamelCase(key)
		}

		mapKey := reflect.ValueOf(normalizedKey)
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

func toSnakeCase(v string) string {
	runes := []rune(v)

	var b strings.Builder
	for i, r := range runes {
		if !unicode.IsUpper(r) {
			b.WriteRune(r)
			continue
		}

		if i > 0 {
			prev := runes[i-1]
			hasNext := i+1 < len(runes)
			nextIsLower := hasNext && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower) {
				b.WriteByte('_')
			}
		}

		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
}

// toLowerCamelCase normalizes snake_case identifiers to lowerCamelCase.
// This is intentionally a normalization helper (not a reversible transform).
func toLowerCamelCase(v string) string {
	if !strings.Contains(v, "_") {
		return v
	}

	parts := strings.Split(v, "_")
	var b strings.Builder
	wrotePart := false

	for _, part := range parts {
		if part == "" {
			continue
		}

		lowerPart := strings.ToLower(part)
		if !wrotePart {
			b.WriteString(lowerPart)
			wrotePart = true
			continue
		}

		partRunes := []rune(lowerPart)
		partRunes[0] = unicode.ToUpper(partRunes[0])
		b.WriteString(string(partRunes))
		wrotePart = true
	}

	if !wrotePart {
		return v
	}

	return b.String()
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

func (r *Resource[T, L]) setSecureState(
	ctx context.Context,
	state *tfsdk.State,
	data ResourceModel,
	secureVersion types.Int64,
) diag.Diagnostics {
	return r.setSecureStateWithData(ctx, state, data, secureVersion)
}

func (r *Resource[T, L]) setSecureStateWithData(
	ctx context.Context,
	state stateData,
	data ResourceModel,
	secureVersion types.Int64,
) diag.Diagnostics {
	// IMPORTANT: keep base attributes in sync with ResourceModel fields.
	var diags diag.Diagnostics

	diags.Append(state.SetAttribute(ctx, path.Root("id"), data.ID)...)
	diags.Append(state.SetAttribute(ctx, path.Root("metadata"), data.Metadata)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec"), data.Spec)...)
	diags.Append(state.SetAttribute(ctx, path.Root("options"), data.Options)...)
	diags.Append(state.SetAttribute(ctx, path.Root("secure"), r.nullSecureObject())...)
	diags.Append(state.SetAttribute(ctx, path.Root("secure_version"), secureVersion)...)

	return diags
}

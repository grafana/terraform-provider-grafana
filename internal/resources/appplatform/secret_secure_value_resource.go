package appplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/secret/pkg/apis/secret/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type SecureValueSpecModel struct {
	Description types.String `tfsdk:"description"`
	Decrypters  types.List   `tfsdk:"decrypters"`
	Value       types.String `tfsdk:"value"`
	ValueHash   types.String `tfsdk:"value_hash"`
	Ref         types.String `tfsdk:"ref"`
}

type secureValueResource struct {
	client   *sdkresource.NamespacedClient[*v1beta1.SecureValue, *v1beta1.SecureValueList]
	clientID string
}

func SecureValue() NamedResource {
	return NamedResource{
		Resource: &secureValueResource{},
		Name:     formatResourceType(v1beta1.SecureValueKind()),
		Category: common.CategoryGrafanaEnterprise,
	}
}

func (r *secureValueResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = formatResourceType(v1beta1.SecureValueKind())
}

func (r *secureValueResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	exactlyOne := stringvalidator.ExactlyOneOf(
		path.MatchRelative().AtParent().AtName("value"),
		path.MatchRelative().AtParent().AtName("ref"),
	)

	resp.Schema = schema.Schema{
		Description: "Manages a Secrets Management secure value.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the resource derived from UID.",
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": secretMetadataBlock(DNS1123SubdomainValidator{}),
			"spec": schema.SingleNestedBlock{
				Description: "The spec of the secure value.",
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "Secure value description.",
						Validators: []validator.String{
							stringvalidator.UTF8LengthBetween(1, 25),
						},
					},
					"decrypters": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "List of decrypters allowed to read this secure value.",
						Validators: []validator.List{
							listvalidator.SizeAtMost(64),
							listvalidator.UniqueValues(),
						},
					},
					"value": schema.StringAttribute{
						Optional:    true,
						WriteOnly:   true,
						Sensitive:   true,
						Description: "Plaintext value to store. This value is write-only.",
						Validators: []validator.String{
							exactlyOne,
							stringvalidator.UTF8LengthBetween(1, 24576),
						},
					},
					"value_hash": schema.StringAttribute{
						Computed:    true,
						Sensitive:   true,
						Description: "Hash of the stored plaintext value.",
					},
					"ref": schema.StringAttribute{
						Optional:    true,
						Description: "Reference to an existing secret managed by the keeper.",
						Validators: []validator.String{
							exactlyOne,
							stringvalidator.UTF8LengthBetween(1, 1024),
						},
					},
				},
			},
			"options": secretOptionsBlock(),
		},
	}
}

func (r *secureValueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v1beta1.SecureValueKind())
	if err != nil {
		resp.Diagnostics.AddError("Error creating Grafana App Platform API client", err.Error())
		return
	}

	ns, errMsg := namespaceForClient(client.GrafanaOrgID, client.GrafanaStackID)
	if errMsg != "" {
		resp.Diagnostics.AddError("Error creating Grafana App Platform API client", errMsg)
		return
	}

	r.client = sdkresource.NewNamespaced(sdkresource.NewTypedClient[*v1beta1.SecureValue, *v1beta1.SecureValueList](rcli, v1beta1.SecureValueKind()), ns)
	r.clientID = client.GrafanaAppPlatformAPIClientID
}

func (r *secureValueResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Config.Raw.IsNull() || !req.Config.Raw.IsKnown() || req.Plan.Raw.IsNull() || !req.Plan.Raw.IsKnown() {
		return
	}

	var value types.String
	if diag := req.Config.GetAttribute(ctx, path.Root("spec").AtName("value"), &value); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	switch {
	case value.IsNull():
		resp.Plan.SetAttribute(ctx, path.Root("spec").AtName("value_hash"), basetypes.NewStringNull())
	case value.IsUnknown():
		resp.Plan.SetAttribute(ctx, path.Root("spec").AtName("value_hash"), basetypes.NewStringUnknown())
	default:
		hash := hashSensitiveValue(value.ValueString())
		resp.Plan.SetAttribute(ctx, path.Root("spec").AtName("value_hash"), basetypes.NewStringValue(hash))
	}

	shouldRelaxMetadata := false
	if req.State.Raw.IsNull() || !req.State.Raw.IsKnown() {
		shouldRelaxMetadata = true
	} else if !req.Plan.Raw.Equal(req.State.Raw) {
		shouldRelaxMetadata = true
	}

	if !shouldRelaxMetadata && !value.IsNull() && !value.IsUnknown() {
		var stateValueHash types.String
		if diag := req.State.GetAttribute(ctx, path.Root("spec").AtName("value_hash"), &stateValueHash); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		if stateValueHash.IsNull() || stateValueHash.IsUnknown() {
			shouldRelaxMetadata = true
		} else if stateValueHash.ValueString() != hashSensitiveValue(value.ValueString()) {
			shouldRelaxMetadata = true
		}
	}

	if shouldRelaxMetadata {
		resp.Plan.SetAttribute(ctx, path.Root("metadata").AtName("uuid"), basetypes.NewStringUnknown())
		resp.Plan.SetAttribute(ctx, path.Root("metadata").AtName("version"), basetypes.NewStringUnknown())
		resp.Plan.SetAttribute(ctx, path.Root("metadata").AtName("annotations"), types.MapUnknown(types.StringType))
	}
}

func (r *secureValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config *ResourceModel
	if diag := req.Config.Get(ctx, &config); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if config == nil {
		return
	}

	obj := v1beta1.SecureValueKind().Schema.ZeroValue().(*v1beta1.SecureValue)

	if err := setManagerProperties(obj, r.clientID); err != nil {
		resp.Diagnostics.AddError("failed to set manager properties", err.Error())
		return
	}

	if diag := SetMetadataFromModel(ctx, config.Metadata, obj); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	spec, rawValue, diags := parseSecureValueSpec(ctx, config.Spec)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if err := obj.SetSpec(spec); err != nil {
		resp.Diagnostics.AddError("failed to set spec", err.Error())
		return
	}

	res, err := r.client.Create(ctx, obj, sdkresource.CreateOptions{})
	if err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionCreate, obj.GetName(), formatResourceType(v1beta1.SecureValueKind()), err)...)
		return
	}

	state, diags := saveSecureValueState(ctx, res, config, rawValue)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *secureValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if req.State.Raw.IsNull() || !req.State.Raw.IsKnown() {
		return
	}

	var state *ResourceModel
	if diag := req.State.Get(ctx, &state); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if state == nil {
		return
	}

	uid, diag := metadataUID(ctx, state.Metadata)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	res, err := r.client.Get(ctx, uid)
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionRead, uid, formatResourceType(v1beta1.SecureValueKind()), err)...)
		return
	}

	stateValue, diags := saveSecureValueState(ctx, res, state, "")
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, stateValue)...)
}

func (r *secureValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *ResourceModel
	if diag := req.Plan.Get(ctx, &plan); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if plan == nil {
		return
	}

	var config *ResourceModel
	if diag := req.Config.Get(ctx, &config); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if config == nil {
		return
	}

	var prior *ResourceModel
	if diag := req.State.Get(ctx, &prior); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if prior == nil {
		return
	}

	uid, diag := metadataUID(ctx, plan.Metadata)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	obj, err := r.client.Get(ctx, uid)
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionRead, uid, formatResourceType(v1beta1.SecureValueKind()), err)...)
		return
	}

	spec, rawValue, diags := parseSecureValueSpec(ctx, config.Spec)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	value, diag := maybeSkipUnchangedValue(ctx, rawValue, prior.Spec)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	rawValue = value
	if value == "" {
		spec.Value = nil
	} else {
		exposed := v1beta1.SecureValueExposedSecureValue(value)
		spec.Value = &exposed
	}
	if err := obj.SetSpec(spec); err != nil {
		resp.Diagnostics.AddError("failed to set spec", err.Error())
		return
	}

	if err := setManagerProperties(obj, r.clientID); err != nil {
		resp.Diagnostics.AddError("failed to set manager properties", err.Error())
		return
	}

	opts := sdkresource.UpdateOptions{
		ResourceVersion: obj.GetResourceVersion(),
	}
	var resourceOptions ResourceOptions
	if diag := ParseResourceOptionsFromModel(ctx, *plan, &resourceOptions); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if resourceOptions.Overwrite {
		opts.ResourceVersion = ""
	}

	res, err := r.client.Update(ctx, obj, opts)
	if err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionUpdate, obj.GetName(), formatResourceType(v1beta1.SecureValueKind()), err)...)
		return
	}

	state, diags := saveSecureValueState(ctx, res, plan, rawValue)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *secureValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if req.State.Raw.IsNull() || !req.State.Raw.IsKnown() {
		return
	}

	var data *ResourceModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if data == nil {
		return
	}

	uid, diag := metadataUID(ctx, data.Metadata)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if err := r.client.Delete(ctx, uid, sdkresource.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionDelete, uid, formatResourceType(v1beta1.SecureValueKind()), err)...)
		return
	}
}

func (r *secureValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	res, err := r.client.Get(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.Append(ErrorToDiagnostics(ResourceActionRead, req.ID, formatResourceType(v1beta1.SecureValueKind()), err)...)
		return
	}

	data := ResourceModel{
		Metadata: emptyMetadataObject(),
		Spec:     emptySecureValueSpecObject(),
		Options:  emptyOptionsObject(),
	}

	state, diags := saveSecureValueState(ctx, res, &data, "")
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	options, diag := optionsOverwriteState(ctx)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	state.Options = options
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func parseSecureValueSpec(ctx context.Context, src types.Object) (v1beta1.SecureValueSpec, string, diag.Diagnostics) {
	if src.IsNull() || src.IsUnknown() {
		return v1beta1.SecureValueSpec{}, "", nil
	}

	var data SecureValueSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v1beta1.SecureValueSpec{}, "", diag
	}

	spec := v1beta1.SecureValueSpec{
		Description: data.Description.ValueString(),
	}

	if !data.Decrypters.IsNull() && !data.Decrypters.IsUnknown() {
		decrypters := make([]string, 0, len(data.Decrypters.Elements()))
		for _, elem := range data.Decrypters.Elements() {
			value, ok := elem.(types.String)
			if !ok || value.IsNull() || value.IsUnknown() {
				continue
			}
			decrypters = append(decrypters, value.ValueString())
		}
		spec.Decrypters = decrypters
	}

	var rawValue string
	if !data.Value.IsNull() && !data.Value.IsUnknown() {
		rawValue = data.Value.ValueString()
		value := v1beta1.SecureValueExposedSecureValue(rawValue)
		spec.Value = &value
	}

	if !data.Ref.IsNull() && !data.Ref.IsUnknown() {
		ref := data.Ref.ValueString()
		spec.Ref = &ref
	}

	return spec, rawValue, nil
}

func saveSecureValueState(ctx context.Context, src *v1beta1.SecureValue, existing *ResourceModel, rawValue string) (*ResourceModel, diag.Diagnostics) {
	if existing.Metadata.IsNull() || existing.Metadata.IsUnknown() {
		existing.Metadata = emptyMetadataObject()
	}
	if existing.Spec.IsNull() || existing.Spec.IsUnknown() {
		existing.Spec = emptySecureValueSpecObject()
	}
	if existing.Options.IsNull() || existing.Options.IsUnknown() {
		existing.Options = emptyOptionsObject()
	}

	if d := SaveResourceToModel(ctx, src, existing); d.HasError() {
		return existing, d
	}

	if name := src.GetName(); name != "" {
		existing.ID = types.StringValue(name)
	}

	var prior SecureValueSpecModel
	if diag := existing.Spec.As(ctx, &prior, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return existing, diag
	}

	valueHash := prior.ValueHash
	switch {
	case rawValue != "":
		valueHash = types.StringValue(hashSensitiveValue(rawValue))
	case src.Spec.Ref != nil && *src.Spec.Ref != "":
		valueHash = types.StringNull()
	}

	spec := SecureValueSpecModel{
		Description: types.StringValue(src.Spec.Description),
		Value:       types.StringNull(),
		ValueHash:   valueHash,
	}

	if len(src.Spec.Decrypters) > 0 {
		list, diag := types.ListValueFrom(ctx, types.StringType, src.Spec.Decrypters)
		if diag.HasError() {
			return existing, diag
		}
		spec.Decrypters = list
	} else {
		spec.Decrypters = types.ListNull(types.StringType)
	}

	if src.Spec.Ref != nil && *src.Spec.Ref != "" {
		spec.Ref = types.StringValue(*src.Spec.Ref)
	} else {
		spec.Ref = types.StringNull()
	}

	specObj, diag := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"description": types.StringType,
		"decrypters":  types.ListType{ElemType: types.StringType},
		"value":       types.StringType,
		"value_hash":  types.StringType,
		"ref":         types.StringType,
	}, spec)
	if diag.HasError() {
		return existing, diag
	}
	existing.Spec = specObj

	return existing, nil
}

func emptySecureValueSpecObject() types.Object {
	return types.ObjectNull(map[string]attr.Type{
		"description": types.StringType,
		"decrypters":  types.ListType{ElemType: types.StringType},
		"value":       types.StringType,
		"value_hash":  types.StringType,
		"ref":         types.StringType,
	})
}

func maybeSkipUnchangedValue(ctx context.Context, value string, priorSpec types.Object) (string, diag.Diagnostics) {
	if value == "" {
		return value, nil
	}

	if priorSpec.IsNull() || priorSpec.IsUnknown() {
		return value, nil
	}

	var prior SecureValueSpecModel
	if diag := priorSpec.As(ctx, &prior, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return value, diag
	}

	if prior.ValueHash.IsNull() || prior.ValueHash.IsUnknown() {
		return value, nil
	}

	if hashSensitiveValue(value) == prior.ValueHash.ValueString() {
		return "", nil
	}

	return value, nil
}

func hashSensitiveValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

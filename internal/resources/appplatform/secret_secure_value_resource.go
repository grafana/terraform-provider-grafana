package appplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

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
)

type SecureValueSpecModel struct {
	Description types.String `tfsdk:"description" json:"description"`
	Decrypters  types.List   `tfsdk:"decrypters" json:"decrypters"`
	Value       types.String `tfsdk:"value" json:"value"`
	ValueHash   types.String `tfsdk:"value_hash"`
	Ref         types.String `tfsdk:"ref" json:"ref"`
}

func SecureValue() NamedResource {
	exactlyOne := stringvalidator.ExactlyOneOf(
		path.MatchRelative().AtParent().AtName("value"),
		path.MatchRelative().AtParent().AtName("ref"),
	)

	return NewNamedResource[*v1beta1.SecureValue, *v1beta1.SecureValueList](
		common.CategoryGrafanaEnterprise,
		ResourceConfig[*v1beta1.SecureValue]{
			Kind: v1beta1.SecureValueKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages a Secrets Management secure value.",
				SpecAttributes: map[string]schema.Attribute{
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
			SpecParser:    parseSecureValueSpec,
			SpecSaver:     saveSecureValueSpec,
			PlanModifier:  secureValuePlanModifier,
			UpdateDecider: secureValueUpdateDecider,
			UseConfigSpec: true,
		},
	)
}

func secureValuePlanModifier(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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

func parseSecureValueSpec(ctx context.Context, src types.Object, dst *v1beta1.SecureValue) diag.Diagnostics {
	if src.IsNull() || src.IsUnknown() {
		return nil
	}

	var data SecureValueSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
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

	if !data.Value.IsNull() && !data.Value.IsUnknown() {
		value := v1beta1.SecureValueExposedSecureValue(data.Value.ValueString())
		spec.Value = &value
	}

	if !data.Ref.IsNull() && !data.Ref.IsUnknown() {
		ref := data.Ref.ValueString()
		spec.Ref = &ref
	}

	diags := diag.Diagnostics{}
	if err := dst.SetSpec(spec); err != nil {
		diags.AddError("failed to set spec", err.Error())
		return diags
	}

	return diags
}

func saveSecureValueSpec(ctx context.Context, src *v1beta1.SecureValue, dst *ResourceModel) diag.Diagnostics {
	var data SecureValueSpecModel
	if diags := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	data.Description = types.StringValue(src.Spec.Description)
	data.Value = types.StringNull()
	data.ValueHash = types.StringNull()

	if len(src.Spec.Decrypters) > 0 {
		list, diag := types.ListValueFrom(ctx, types.StringType, src.Spec.Decrypters)
		if diag.HasError() {
			return diag
		}
		data.Decrypters = list
	} else {
		data.Decrypters = types.ListNull(types.StringType)
	}

	if src.Spec.Ref != nil && *src.Spec.Ref != "" {
		data.Ref = types.StringValue(*src.Spec.Ref)
	} else {
		data.Ref = types.StringNull()
	}

	specObj, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"description": types.StringType,
		"decrypters":  types.ListType{ElemType: types.StringType},
		"value":       types.StringType,
		"value_hash":  types.StringType,
		"ref":         types.StringType,
	}, data)
	if diags.HasError() {
		return diags
	}

	dst.Spec = specObj
	return nil
}

func secureValueUpdateDecider(ctx context.Context, _ resource.UpdateRequest, plan ResourceModel, prior ResourceModel) (bool, diag.Diagnostics) {
	if plan.Spec.IsNull() || plan.Spec.IsUnknown() || prior.Spec.IsNull() || prior.Spec.IsUnknown() {
		return false, nil
	}

	planSpec, diags := secureValueSpecFromObject(ctx, plan.Spec)
	if diags.HasError() {
		return false, diags
	}
	priorSpec, diags := secureValueSpecFromObject(ctx, prior.Spec)
	if diags.HasError() {
		return false, diags
	}

	if planSpec.ValueHash.IsNull() || planSpec.ValueHash.IsUnknown() ||
		priorSpec.ValueHash.IsNull() || priorSpec.ValueHash.IsUnknown() {
		return false, nil
	}

	if planSpec.ValueHash.ValueString() != priorSpec.ValueHash.ValueString() {
		return false, nil
	}

	if !secureValueStringEqual(planSpec.Description, priorSpec.Description) {
		return false, nil
	}
	if !secureValueStringEqual(planSpec.Ref, priorSpec.Ref) {
		return false, nil
	}
	if !secureValueListEqual(planSpec.Decrypters, priorSpec.Decrypters) {
		return false, nil
	}

	return true, nil
}

func secureValueSpecFromObject(ctx context.Context, src types.Object) (SecureValueSpecModel, diag.Diagnostics) {
	var data SecureValueSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return SecureValueSpecModel{}, diag
	}

	return data, nil
}

func secureValueStringEqual(a, b types.String) bool {
	if a.IsUnknown() || b.IsUnknown() {
		return false
	}

	if a.IsNull() || b.IsNull() {
		return a.IsNull() && b.IsNull()
	}

	return a.ValueString() == b.ValueString()
}

func secureValueListEqual(a, b types.List) bool {
	if a.IsUnknown() || b.IsUnknown() {
		return false
	}
	if a.IsNull() || b.IsNull() {
		return a.IsNull() && b.IsNull()
	}

	aElems := a.Elements()
	bElems := b.Elements()
	if len(aElems) != len(bElems) {
		return false
	}

	for i := range aElems {
		aVal, ok := aElems[i].(types.String)
		if !ok || aVal.IsNull() || aVal.IsUnknown() {
			return false
		}
		bVal, ok := bElems[i].(types.String)
		if !ok || bVal.IsNull() || bVal.IsUnknown() {
			return false
		}
		if aVal.ValueString() != bVal.ValueString() {
			return false
		}
	}

	return true
}

func hashSensitiveValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

package appplatform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func secretMetadataBlock(validators ...validator.String) schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "The metadata of the resource.",
		Attributes: map[string]schema.Attribute{
			"uid": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the resource.",
				Validators:  validators,
			},
			"folder_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The UID of the folder to save the resource in.",
			},
			"annotations": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Annotations of the resource.",
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The globally unique identifier of a resource, used by the API for tracking.",
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
	}
}

// emptyMetadataObject returns a metadata object with every shared attribute null. The values are
// derived from baseMetadataTypeMap rather than listed here so that adding a shared attribute
// cannot leave the two maps disagreeing, which types.ObjectValueMust would panic on.
func emptyMetadataObject() types.Object {
	attrTypes := baseMetadataTypeMap()

	values := make(map[string]attr.Value, len(attrTypes))
	for name, attrType := range attrTypes {
		values[name] = newNullValueOfType(attrType)
	}

	return types.ObjectValueMust(attrTypes, values)
}

// newNullValueOfType returns the null value for an attribute type. attr.Type carries the
// knowledge of its own null representation via ValueFromTerraform, so this stays correct for
// attribute types added later.
func newNullValueOfType(attrType attr.Type) attr.Value {
	value, err := attrType.ValueFromTerraform(context.Background(), tftypes.NewValue(attrType.TerraformType(context.Background()), nil))
	if err != nil {
		// Unreachable: every attr.Type accepts a null of its own Terraform type.
		panic(fmt.Sprintf("failed to build null value for %s: %s", attrType, err))
	}
	return value
}

func metadataUID(ctx context.Context, metadata types.Object) (string, diag.Diagnostics) {
	if metadata.IsNull() || metadata.IsUnknown() {
		return "", diag.Diagnostics{diag.NewErrorDiagnostic("missing metadata", "metadata.uid is required")}
	}

	var mod ResourceMetadataModel
	if diag := metadata.As(ctx, &mod, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return "", diag
	}

	if mod.UID.IsNull() || mod.UID.IsUnknown() || mod.UID.ValueString() == "" {
		return "", diag.Diagnostics{diag.NewErrorDiagnostic("missing metadata uid", "metadata.uid must be set")}
	}

	return mod.UID.ValueString(), nil
}

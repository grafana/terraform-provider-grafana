package appplatform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

// emptyMetadataObject returns a metadata object with every shared attribute null.
//
// The attribute types come from baseMetadataTypeMap while the values are listed here, so the two
// could in principle disagree — which types.ObjectValueMust panics on. That is guarded by
// TestEmptyMetadataObjectMatchesBaseMetadataTypes rather than by deriving the values, which
// needed reflection for no real benefit.
func emptyMetadataObject() types.Object {
	return types.ObjectValueMust(
		baseMetadataTypeMap(),
		map[string]attr.Value{
			"uuid":        types.StringNull(),
			"uid":         types.StringNull(),
			"folder_uid":  types.StringNull(),
			"version":     types.StringNull(),
			"url":         types.StringNull(),
			"annotations": types.MapNull(types.StringType),
		},
	)
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

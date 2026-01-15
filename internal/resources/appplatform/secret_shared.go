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

func secretOptionsBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Options for applying the resource.",
		Attributes: map[string]schema.Attribute{
			"overwrite": schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true if you want to overwrite existing resource with newer version, same resource title in folder or same resource uid.",
			},
		},
	}
}

func emptyMetadataObject() types.Object {
	return types.ObjectValueMust(
		map[string]attr.Type{
			"uuid":        types.StringType,
			"uid":         types.StringType,
			"folder_uid":  types.StringType,
			"version":     types.StringType,
			"url":         types.StringType,
			"annotations": types.MapType{ElemType: types.StringType},
		},
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

func emptyOptionsObject() types.Object {
	return types.ObjectNull(map[string]attr.Type{
		"overwrite": types.BoolType,
	})
}

func optionsOverwriteState(ctx context.Context) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, map[string]attr.Type{
		"overwrite": types.BoolType,
	}, ResourceOptionsModel{
		Overwrite: types.BoolValue(true),
	})
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

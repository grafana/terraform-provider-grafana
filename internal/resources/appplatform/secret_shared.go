package appplatform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// secretMetadataModel mirrors the secret metadata block (secretMetadataBlock). It is kept
// separate from the generic ResourceMetadataModel because the secret metadata block omits
// some generic attributes (and the Terraform framework requires struct fields and object
// attributes to match exactly), but it carries the same org_id override for multi-org use.
type secretMetadataModel struct {
	UUID        types.String `tfsdk:"uuid"`
	UID         types.String `tfsdk:"uid"`
	OrgID       types.Int64  `tfsdk:"org_id"`
	FolderUID   types.String `tfsdk:"folder_uid"`
	Version     types.String `tfsdk:"version"`
	URL         types.String `tfsdk:"url"`
	Annotations types.Map    `tfsdk:"annotations"`
}

// secretMetadataAttrTypes is the attribute-type map for the secret metadata object. It must
// stay in sync with secretMetadataBlock and secretMetadataModel.
var secretMetadataAttrTypes = map[string]attr.Type{
	"uuid":        types.StringType,
	"uid":         types.StringType,
	"org_id":      types.Int64Type,
	"folder_uid":  types.StringType,
	"version":     types.StringType,
	"url":         types.StringType,
	"annotations": types.MapType{ElemType: types.StringType},
}

func secretMetadataBlock(validators ...validator.String) schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "The metadata of the resource.",
		Attributes: map[string]schema.Attribute{
			"uid": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the resource.",
				Validators:  validators,
			},
			"org_id": schema.Int64Attribute{
				Optional: true,
				Description: "The Grafana organization ID this resource belongs to, for self-hosted OSS or Enterprise Grafana. " +
					"When set, it overrides the provider's `org_id` for this resource only, so a single provider configuration " +
					"can manage resources across multiple organizations. Not supported on Grafana Cloud (configure a stack with " +
					"`stack_id` instead). Changing this value forces the resource to be recreated in the new organization.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
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

func emptyMetadataObject() types.Object {
	return types.ObjectValueMust(
		secretMetadataAttrTypes,
		map[string]attr.Value{
			"uuid":        types.StringNull(),
			"uid":         types.StringNull(),
			"org_id":      types.Int64Null(),
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

	var mod secretMetadataModel
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

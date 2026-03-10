package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/secret/pkg/apis/secret/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type KeeperSpecModel struct {
	Description types.String `tfsdk:"description"`
	AWS         types.Object `tfsdk:"aws"`
}

type KeeperAWSModel struct {
	Region     types.String `tfsdk:"region"`
	AssumeRole types.Object `tfsdk:"assume_role"`
}

type KeeperAWSAssumeRoleModel struct {
	AssumeRoleARN types.String `tfsdk:"assume_role_arn"`
	ExternalID    types.String `tfsdk:"external_id"`
}

func Keeper() NamedResource {
	return NewNamedResource[*v1beta1.Keeper, *v1beta1.KeeperList](
		common.CategoryGrafanaEnterprise,
		ResourceConfig[*v1beta1.Keeper]{
			Kind: v1beta1.KeeperKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages a Secrets Management keeper.",
				SpecAttributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "Keeper description.",
						Validators: []validator.String{
							stringvalidator.UTF8LengthBetween(1, 253),
						},
					},
				},
				SpecBlocks: map[string]schema.Block{
					"aws": schema.SingleNestedBlock{
						Description: "AWS Secrets Manager configuration.",
						Attributes: map[string]schema.Attribute{
							"region": schema.StringAttribute{
								Required:    true,
								Description: "AWS region.",
							},
						},
						Blocks: map[string]schema.Block{
							"assume_role": schema.SingleNestedBlock{
								Description: "IAM role assumption configuration.",
								Validators: []validator.Object{
									requireAttrsWhenPresent("assume_role_arn", "external_id"),
								},
								Attributes: map[string]schema.Attribute{
									"assume_role_arn": schema.StringAttribute{
										Optional:    true,
										Description: "Assume role ARN.",
									},
									"external_id": schema.StringAttribute{
										Optional:    true,
										Description: "Assume role external ID.",
									},
								},
							},
						},
					},
				},
			},
			SpecParser: parseKeeperSpec,
			SpecSaver:  saveKeeperSpec,
		},
	)
}

func parseKeeperSpec(ctx context.Context, src types.Object, dst *v1beta1.Keeper) diag.Diagnostics {
	if src.IsNull() || src.IsUnknown() {
		return nil
	}

	var data KeeperSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	spec := v1beta1.KeeperSpec{
		Description: data.Description.ValueString(),
	}

	if !data.AWS.IsNull() && !data.AWS.IsUnknown() {
		var aws KeeperAWSModel
		if diag := data.AWS.As(ctx, &aws, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diag.HasError() {
			return diag
		}

		awsSpec := &v1beta1.KeeperAWSConfig{
			Region: aws.Region.ValueString(),
		}

		if !aws.AssumeRole.IsNull() && !aws.AssumeRole.IsUnknown() {
			var assume KeeperAWSAssumeRoleModel
			if diag := aws.AssumeRole.As(ctx, &assume, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			}); diag.HasError() {
				return diag
			}
			awsSpec.AssumeRole = &v1beta1.KeeperAWSAssumeRole{
				AssumeRoleArn: assume.AssumeRoleARN.ValueString(),
				ExternalID:    assume.ExternalID.ValueString(),
			}
		}

		spec.Aws = awsSpec
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return nil
}

func saveKeeperSpec(ctx context.Context, src *v1beta1.Keeper, dst *ResourceModel) diag.Diagnostics {
	var data KeeperSpecModel
	if diags := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	data.Description = types.StringValue(src.Spec.Description)

	if src.Spec.Aws != nil {
		assumeObj := types.ObjectNull(map[string]attr.Type{
			"assume_role_arn": types.StringType,
			"external_id":     types.StringType,
		})
		if src.Spec.Aws.AssumeRole != nil {
			var diags diag.Diagnostics
			assumeObj, diags = types.ObjectValueFrom(ctx, map[string]attr.Type{
				"assume_role_arn": types.StringType,
				"external_id":     types.StringType,
			}, KeeperAWSAssumeRoleModel{
				AssumeRoleARN: types.StringValue(src.Spec.Aws.AssumeRole.AssumeRoleArn),
				ExternalID:    types.StringValue(src.Spec.Aws.AssumeRole.ExternalID),
			})
			if diags.HasError() {
				return diags
			}
		}

		awsObj, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"region":      types.StringType,
			"assume_role": types.ObjectType{AttrTypes: map[string]attr.Type{"assume_role_arn": types.StringType, "external_id": types.StringType}},
		}, KeeperAWSModel{
			Region:     types.StringValue(src.Spec.Aws.Region),
			AssumeRole: assumeObj,
		})
		if diags.HasError() {
			return diags
		}
		data.AWS = awsObj
	} else {
		data.AWS = types.ObjectNull(map[string]attr.Type{
			"region":      types.StringType,
			"assume_role": types.ObjectType{AttrTypes: map[string]attr.Type{"assume_role_arn": types.StringType, "external_id": types.StringType}},
		})
	}

	specObj, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"description": types.StringType,
		"aws": types.ObjectType{AttrTypes: map[string]attr.Type{
			"region":      types.StringType,
			"assume_role": types.ObjectType{AttrTypes: map[string]attr.Type{"assume_role_arn": types.StringType, "external_id": types.StringType}},
		}},
	}, &data)
	if diags.HasError() {
		return diags
	}

	dst.Spec = specObj
	return nil
}

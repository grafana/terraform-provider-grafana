package appplatform

import (
	"context"

	"github.com/grafana/grafana/pkg/extensions/apis/scim/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// SCIMConfigSpecModel is a model for the SCIMConfig spec.
type SCIMConfigSpecModel struct {
	EnableUserSync  types.Bool `tfsdk:"enable_user_sync"`
	EnableGroupSync types.Bool `tfsdk:"enable_group_sync"`
}

// SCIMConfig creates a new Grafana SCIMConfig resource.
func SCIMConfig() NamedResource {
	return NewNamedResource[*v0alpha1.SCIMConfig, *v0alpha1.SCIMConfigList](
		common.CategoryGrafanaApps,
		ResourceConfig[*v0alpha1.SCIMConfig]{
			Kind: v0alpha1.SCIMConfigKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana SCIM configuration.",
				MarkdownDescription: `
Manages Grafana SCIM configuration using the new app platform APIs.

* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/scim/)
`,
				SpecAttributes: map[string]schema.Attribute{
					"enable_user_sync": schema.BoolAttribute{
						Required:    true,
						Description: "Whether user synchronization is enabled.",
					},
					"enable_group_sync": schema.BoolAttribute{
						Required:    true,
						Description: "Whether group synchronization is enabled.",
					},
				},
			},
			SpecParser: func(ctx context.Context, spec types.Object, dst *v0alpha1.SCIMConfig) diag.Diagnostics {
				var data SCIMConfigSpecModel
				if diag := spec.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}
				dst.Spec.EnableUserSync = data.EnableUserSync.ValueBool()
				dst.Spec.EnableGroupSync = data.EnableGroupSync.ValueBool()
				return diag.Diagnostics{}
			},
			SpecSaver: func(ctx context.Context, obj *v0alpha1.SCIMConfig, dst *ResourceModel) diag.Diagnostics {
				data := SCIMConfigSpecModel{
					EnableUserSync:  types.BoolValue(obj.Spec.EnableUserSync),
					EnableGroupSync: types.BoolValue(obj.Spec.EnableGroupSync),
				}
				spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
					"enable_user_sync":  types.BoolType,
					"enable_group_sync": types.BoolType,
				}, &data)
				if diags.HasError() {
					return diags
				}
				dst.Spec = spec
				return diag.Diagnostics{}
			},
		},
	)
}

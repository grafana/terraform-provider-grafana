package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type templateGroupSpecModel struct {
	Title             types.String `tfsdk:"title"`
	Content           types.String `tfsdk:"content"`
	DisableProvenance types.Bool   `tfsdk:"disable_provenance"`
}

func TemplateGroup() NamedResource {
	return NewNamedResource[*v1beta1.TemplateGroup, *v1beta1.TemplateGroupList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.TemplateGroup]{
			Kind: v1beta1.TemplateGroupKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages a Grafana Alerting notification template group containing one or more notification templates. Requires Grafana 13.4.0 or later.",
				MarkdownDescription: `
Manages a Grafana Alerting notification template group containing one or more notification templates.

Requires Grafana 13.4.0 or later.

This resource replaces ` + "`grafana_message_template`" + `, which uses the deprecated legacy provisioning API scheduled for removal in Grafana 14.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/configure-notifications/template-notifications/)
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "The display name of the notification template group.",
					},
					"content": schema.StringAttribute{
						Required:    true,
						Description: "The notification template definitions, containing one or more Go template `define` blocks.",
					},
					"disable_provenance": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Set to `true` to allow editing this resource from other sources (UI, API). Defaults to `false`, which locks the resource to Terraform management only.",
					},
				},
			},
			// Shares the org's Alertmanager config with other resources, so concurrent creates can 409.
			RetryCreateOnConflict: true,
			SpecParser:            parseTemplateGroupSpec,
			SpecSaver:             saveTemplateGroupSpec,
		})
}

func parseTemplateGroupSpec(ctx context.Context, src types.Object, dst *v1beta1.TemplateGroup) diag.Diagnostics {
	var data templateGroupSpecModel
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	if err := dst.SetSpec(v1beta1.TemplateGroupSpec{
		Title:   data.Title.ValueString(),
		Content: data.Content.ValueString(),
		Kind:    v1beta1.TemplateGroupTemplateKindGrafana,
	}); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to get metadata accessor", err.Error()),
		}
	}
	if data.DisableProvenance.ValueBool() {
		meta.SetAnnotation(v1beta1.ProvenanceStatusAnnotationKey, provenanceNone)
	} else {
		meta.SetAnnotation(v1beta1.ProvenanceStatusAnnotationKey, provenanceAPI)
	}

	return diag.Diagnostics{}
}

func saveTemplateGroupSpec(_ context.Context, src *v1beta1.TemplateGroup, dst *ResourceModel) diag.Diagnostics {
	disableProvenance := false
	if meta, err := utils.MetaAccessor(src); err == nil {
		disableProvenance = meta.GetAnnotation(v1beta1.ProvenanceStatusAnnotationKey) == provenanceNone
	}

	spec, diags := types.ObjectValue(
		map[string]attr.Type{
			"title":              types.StringType,
			"content":            types.StringType,
			"disable_provenance": types.BoolType,
		},
		map[string]attr.Value{
			"title":              types.StringValue(src.Spec.Title),
			"content":            types.StringValue(src.Spec.Content),
			"disable_provenance": types.BoolValue(disableProvenance),
		},
	)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec
	return diag.Diagnostics{}
}

package fleetmanagement

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.ResourceWithConfigValidators = &pipelineResource{}

func (r *pipelineResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&pipelineContentsValidator{},
	}
}

type pipelineContentsValidator struct{}

func (v *pipelineContentsValidator) Description(ctx context.Context) string {
	return "Validates pipeline contents based on config_type (ALLOY uses River syntax, OTEL uses YAML)"
}

func (v *pipelineContentsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *pipelineContentsValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config pipelineModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Contents.IsNull() || config.Contents.IsUnknown() || config.ConfigType.IsUnknown() {
		return
	}

	contents := config.Contents.ValueString()
	configType := config.ConfigType.ValueString()

	diags = validatePipelineContents(contents, configType)
	resp.Diagnostics.Append(diags...)
}

// validatePipelineContents validates the contents field based on the config_type.
// This is extracted to allow direct testing without mocking Terraform's config framework.
func validatePipelineContents(contents, configType string) diag.Diagnostics {
	var diags diag.Diagnostics
	if configType == "" {
		configType = ConfigTypeAlloy
	}

	switch configType {
	case ConfigTypeAlloy:
		_, err := parseRiver(contents)
		if err != nil {
			diags.AddAttributeError(
				path.Root("contents"),
				"Invalid Alloy configuration",
				"The contents field is not valid Alloy/River configuration format.\n\n"+
					"Error: "+err.Error(),
			)
		}
	case ConfigTypeOtel:
		_, err := parseYAML(contents)
		if err != nil {
			diags.AddAttributeError(
				path.Root("contents"),
				"Invalid OTEL configuration",
				"The contents field is not valid YAML format.\n\n"+
					"Error: "+err.Error(),
			)
		}
	}

	return diags
}

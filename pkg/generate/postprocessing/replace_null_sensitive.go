package postprocessing

import (
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// frameworkSensitiveAttrs maps Plugin Framework resource types to attribute names
// that are required and sensitive, for replacing null with a placeholder in generated config.
var frameworkSensitiveAttrs = map[string][]string{
	"grafana_user": {"password"},
}

func ReplaceNullSensitiveAttributes(fpath string) error {
	providerResources := provider.ResourcesMap()
	return postprocessFile(fpath, func(file *hclwrite.File) error {
		for _, block := range file.Body().Blocks() {
			if block.Type() != "resource" {
				continue
			}

			resourceType := block.Labels()[0]
			resourceInfo := providerResources[resourceType]
			resourceSchema := resourceInfo.Schema

			if resourceSchema != nil {
				// Legacy SDK resource: use schema to find required sensitive attributes
				for key := range block.Body().Attributes() {
					attrSchema := resourceSchema.Schema[key]
					if attrSchema == nil {
						continue
					}
					if attrSchema.Sensitive && attrSchema.Required {
						block.Body().SetAttributeValue(key, cty.StringVal("SENSITIVE_VALUE_TO_REPLACE"))
					}
				}
				continue
			}

			// Plugin Framework resource: use static map for known sensitive attributes
			if attrs, ok := frameworkSensitiveAttrs[resourceType]; ok {
				for _, key := range attrs {
					if _, has := block.Body().Attributes()[key]; has {
						block.Body().SetAttributeValue(key, cty.StringVal("SENSITIVE_VALUE_TO_REPLACE"))
					}
				}
			}
		}
		return nil
	})
}

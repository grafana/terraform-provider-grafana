package postprocessing

import (
	"log"

	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

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
			if resourceSchema == nil {
				// Plugin Framework schema not implemented because we have no resources with sensitive attributes in it yet
				log.Printf("resource %s doesn't use the legacy SDK", resourceType)
				continue
			}

			for key := range block.Body().Attributes() {
				attrSchema := resourceSchema.Schema[key]
				if attrSchema == nil {
					// Attribute not found in schema
					continue
				}
				if attrSchema.Sensitive && attrSchema.Required {
					block.Body().SetAttributeValue(key, cty.StringVal("SENSITIVE_VALUE_TO_REPLACE"))
				}
			}
		}
		return nil
	})
}

package postprocessing

import (
	"regexp"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/pkg/provider"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var allowedTerraformChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// UsePreferredResourceNames replaces the resource name with the value of the preferred resource name field.
// The input files (resources.tf + imports.tf) are modified in place.
func UsePreferredResourceNames(fpaths ...string) error {
	providerResources := provider.ResourcesMap()
	replaceMap := map[string]hcl.Traversal{}

	// Go through all resource blocks first
	if err := postprocessFiles(fpaths, func(file *hclwrite.File) error {
		for _, block := range file.Body().Blocks() {
			if block.Type() != "resource" {
				continue
			}

			resourceType := block.Labels()[0]
			resourceInfo := providerResources[resourceType]

			if resourceInfo.PreferredResourceNameField == "" {
				continue
			}

			nameAttr := block.Body().GetAttribute(resourceInfo.PreferredResourceNameField)
			if nameAttr == nil {
				continue
			}
			newResourceName := strings.Trim(string(nameAttr.Expr().BuildTokens(nil).Bytes()), "\" ") // Unquote + trim spaces
			newResourceName = CleanResourceName(newResourceName)

			replaceMap[strings.Join(block.Labels(), ".")] = traversal(resourceType, newResourceName)
			block.SetLabels([]string{resourceType, newResourceName})
		}
		return nil
	}); err != nil {
		return err
	}

	// Go through all import blocks
	return postprocessFiles(fpaths, func(file *hclwrite.File) error {
		for _, block := range file.Body().Blocks() {
			if block.Type() != "import" {
				continue
			}

			resourceTo := strings.TrimSpace(string(block.Body().GetAttribute("to").Expr().BuildTokens(nil).Bytes()))
			if newResourceTo, ok := replaceMap[resourceTo]; ok {
				block.Body().SetAttributeTraversal("to", newResourceTo)
			}
		}

		return nil
	})
}

func CleanResourceName(name string) string {
	cleaned := allowedTerraformChars.ReplaceAllString(name, "_")
	if cleaned[0] >= '0' && cleaned[0] <= '9' {
		cleaned = "_" + cleaned
	}
	if cleaned[0] == '-' {
		cleaned = "_" + cleaned
	}
	return cleaned
}

package testutils

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// WithoutResource removes a resource from a Terraform configuration.
func WithoutResource(t *testing.T, tfCode string, resourceNames ...string) string {
	tfConfig, err := hclwrite.ParseConfig([]byte(tfCode), "", hcl.Pos{Line: 1, Column: 1})
	if err != nil {
		t.Fatalf("failed to parse HCL: %v", err)
	}

	for _, resourceName := range resourceNames {
		block := tfConfig.Body().FirstMatchingBlock("resource", strings.Split(resourceName, "."))
		if block == nil {
			t.Fatalf("failed to find resource %q", resourceName)
		}
		tfConfig.Body().RemoveBlock(block)
	}

	return string(tfConfig.Bytes())
}

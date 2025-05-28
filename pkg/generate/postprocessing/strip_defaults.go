package postprocessing

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

func StripDefaults(fpath string, extraFieldsToRemove map[string]any) error {
	return postprocessFile(fpath, func(file *hclwrite.File) error {
		for _, block := range file.Body().Blocks() {
			stripDefaultsFromBlock(block, extraFieldsToRemove)
		}
		return nil
	})
}

func stripDefaultsFromBlock(block *hclwrite.Block, extraFieldsToRemove map[string]any) {
	for _, innblock := range block.Body().Blocks() {
		stripDefaultsFromBlock(innblock, extraFieldsToRemove)
		if len(innblock.Body().Attributes()) == 0 && len(innblock.Body().Blocks()) == 0 {
			block.Body().RemoveBlock(innblock)
		}
	}
	for name, attribute := range block.Body().Attributes() {
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " null" {
			block.Body().RemoveAttribute(name)
		}
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " {}" {
			block.Body().RemoveAttribute(name)
		}
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " []" {
			block.Body().RemoveAttribute(name)
		}
		for key, valueToRemove := range extraFieldsToRemove {
			if name == key {
				toRemove := false
				fieldValue := strings.TrimSpace(string(attribute.Expr().BuildTokens(nil).Bytes()))
				extractedValue, err := extractJSONEncode(fieldValue)
				if err != nil {
					continue
				}

				// Use extracted value if it's not empty (for jsonencode fields), otherwise use original
				compareValue := fieldValue
				if extractedValue != "" {
					compareValue = extractedValue
				}

				if v, ok := valueToRemove.(bool); ok && v {
					toRemove = true
				} else if v, ok := valueToRemove.(string); ok && v == compareValue {
					toRemove = true
				}
				if toRemove {
					block.Body().RemoveAttribute(name)
				}
			}
		}
	}
}

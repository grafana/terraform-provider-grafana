package postprocessing

import "github.com/hashicorp/hcl/v2/hclwrite"

func WrapJSONFieldsInFunction(fpath string) error {
	return postprocessFile(fpath, func(file *hclwrite.File) error {
		// Find json attributes and use jsonencode
		for _, block := range file.Body().Blocks() {
			for key, attr := range block.Body().Attributes() {
				asMap, err := attributeToMap(attr)
				if err != nil || asMap == nil {
					continue
				}
				tokens := hclwrite.TokensForValue(hcl2ValueFromConfigValue(asMap))
				block.Body().SetAttributeRaw(key, hclwrite.TokensForFunctionCall("jsonencode", tokens))
			}
		}

		return nil
	})
}

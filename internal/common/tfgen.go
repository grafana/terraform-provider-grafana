package common

import (
	"errors"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func StripDefaults(fpath string, extraFieldsToRemove map[string]string) error {
	src, err := os.ReadFile(fpath)
	if err != nil {
		panic(err)
	}

	file, diags := hclwrite.ParseConfig(src, fpath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		err := errors.New("an error occurred")
		if err != nil {
			return err
		}
	}
	hasChanges := false
	for _, block := range file.Body().Blocks() {
		for name, attribute := range block.Body().Attributes() {
			if string(attribute.Expr().BuildTokens(nil).Bytes()) == " null" {
				hasChanges = true
				block.Body().RemoveAttribute(name)
			}
			if string(attribute.Expr().BuildTokens(nil).Bytes()) == " {}" {
				hasChanges = true
				block.Body().RemoveAttribute(name)
			}
			for key, value := range extraFieldsToRemove {
				if name == key && string(attribute.Expr().BuildTokens(nil).Bytes()) == value {
					hasChanges = true
					block.Body().RemoveAttribute(name)
				}
			}
		}
	}
	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		return os.WriteFile(fpath, file.Bytes(), 0644)
	}
	return nil
}

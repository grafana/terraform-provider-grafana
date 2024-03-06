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
		if s := stripDefaultsFromBlock(block, extraFieldsToRemove); s {
			hasChanges = true
		}
	}
	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		return os.WriteFile(fpath, file.Bytes(), 0644)
	}
	return nil
}

func stripDefaultsFromBlock(block *hclwrite.Block, extraFieldsToRemove map[string]string) bool {
	hasChanges := false
	for _, innblock := range block.Body().Blocks() {
		if s := stripDefaultsFromBlock(innblock, extraFieldsToRemove); s {
			hasChanges = true
		}
		if len(innblock.Body().Attributes()) == 0 {
			if rm := block.Body().RemoveBlock(innblock); rm {
				hasChanges = true
			}
		}
	}
	for name, attribute := range block.Body().Attributes() {
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " null" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " {}" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		for key, value := range extraFieldsToRemove {
			if name == key && string(attribute.Expr().BuildTokens(nil).Bytes()) == value {
				if rm := block.Body().RemoveAttribute(name); rm != nil {
					hasChanges = true
				}
			}
		}
	}
	return hasChanges
}

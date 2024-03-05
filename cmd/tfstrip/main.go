package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func main() {
	path := os.Getenv("TFGEN_OUT_PATH")
	if path == "" {
		log.Fatal("TFGEN_OUT_PATH environment variable must be set")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}
	items, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for _, item := range items {
		if item.IsDir() {
			continue
		}

		if !strings.HasSuffix(item.Name(), ".tf") {
			continue
		}

		fpath := filepath.Join(path, item.Name())
		src, err := os.ReadFile(fpath)
		if err != nil {
			panic(err)
		}

		file, diags := hclwrite.ParseConfig(src, item.Name(), hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			err := errors.New("an error occurred")
			if err != nil {
				panic(err)
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
			}
		}
		if hasChanges {
			log.Printf("Updating file: %s\n", item.Name())
			if err := os.WriteFile(fpath, file.Bytes(), 0644); err != nil {
				panic(err)
			}
		}
	}
}

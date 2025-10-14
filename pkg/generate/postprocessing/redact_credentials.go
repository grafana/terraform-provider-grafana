package postprocessing

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func RedactCredentials(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".tf") {
			continue
		}
		fpath := filepath.Join(dir, file.Name())
		err := postprocessFile(fpath, func(file *hclwrite.File) error {
			for _, block := range file.Body().Blocks() {
				if block.Type() != "provider" {
					continue
				}
				for name := range block.Body().Attributes() {
					if strings.Contains(name, "auth") || strings.Contains(name, "token") {
						block.Body().SetAttributeValue(name, cty.StringVal("REDACTED"))
					}
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

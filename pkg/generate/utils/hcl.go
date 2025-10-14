package utils

import (
	"errors"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func ReadHCLFile(fpath string) (*hclwrite.File, error) {
	src, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	file, diags := hclwrite.ParseConfig(src, fpath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, errors.New(diags.Error())
	}

	return file, nil
}

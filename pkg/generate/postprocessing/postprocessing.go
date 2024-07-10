package postprocessing

import (
	"errors"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type postprocessingFunc func(*hclwrite.File) error

func postprocessFile(fpath string, fn postprocessingFunc) error {
	src, err := os.ReadFile(fpath)
	if err != nil {
		return err
	}

	file, diags := hclwrite.ParseConfig(src, fpath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return errors.New(diags.Error())
	}
	initialBytes := file.Bytes()

	if err := fn(file); err != nil {
		return err
	}

	// Write the file only if it has changed
	if string(initialBytes) != string(file.Bytes()) {
		stat, err := os.Stat(fpath)
		if err != nil {
			return err
		}

		if err := os.WriteFile(fpath, file.Bytes(), stat.Mode()); err != nil {
			return err
		}
	}

	return nil
}

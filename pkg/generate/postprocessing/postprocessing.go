package postprocessing

import (
	"os"

	"github.com/grafana/terraform-provider-grafana/v4/pkg/generate/utils"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type postprocessingFunc func(*hclwrite.File) error

func postprocessFile(fpath string, fn postprocessingFunc) error {
	file, err := utils.ReadHCLFile(fpath)
	if err != nil {
		return err
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

func postprocessFiles(fpaths []string, fn postprocessingFunc) error {
	for _, fpath := range fpaths {
		if err := postprocessFile(fpath, fn); err != nil {
			return err
		}
	}
	return nil
}

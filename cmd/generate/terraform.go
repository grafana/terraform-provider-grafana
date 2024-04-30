package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tmccombs/hcl2json/convert"
)

func runTerraform(dir string, command ...string) error {
	cmd := exec.Command("terraform", command...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeBlocks(filepath string, blocks ...*hclwrite.Block) error {
	contents := hclwrite.NewFile()
	for i, b := range blocks {
		if i > 0 {
			contents.Body().AppendNewline()
		}
		contents.Body().AppendBlock(b)
	}

	hclFile, err := os.Create(filepath)
	if err != nil {
		return err
	}
	if _, err := contents.WriteTo(hclFile); err != nil {
		return err
	}
	return hclFile.Close()
}

func convertToTFJSON(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, dirEntry := range entries {
		filePath := filepath.Join(dir, dirEntry.Name())

		hclFile, diags := hclparse.NewParser().ParseHCLFile(filePath)
		if diags.HasErrors() {
			return errors.Join(diags.Errs()...)
		}
		if err := os.Remove(filePath); err != nil {
			return err
		}
		jsonFilePath := filePath + ".json"
		jsonFile, err := os.Create(jsonFilePath)
		if err != nil {
			return err
		}
		converted, err := convert.ConvertFile(hclFile, convert.Options{})
		if err != nil {
			return err
		}

		enc := json.NewEncoder(jsonFile)
		enc.SetIndent("", "  ")
		if err := enc.Encode(converted); err != nil {
			return err
		}
	}

	return nil
}

func traversal(root string, attrs ...string) hcl.Traversal {
	tr := hcl.Traversal{hcl.TraverseRoot{Name: root}}
	for _, attr := range attrs {
		tr = append(tr, hcl.TraverseAttr{Name: attr})
	}
	return tr
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tmccombs/hcl2json/convert"
)

func runTerraformWithOutput(dir string, command ...string) ([]byte, error) {
	cmd := exec.Command("terraform", command...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func runTerraform(dir string, command ...string) error {
	out, err := runTerraformWithOutput(dir, command...)
	fmt.Println(string(out))
	return err
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
		if dirEntry.IsDir() {
			continue
		}
		if filepath.Ext(dirEntry.Name()) != ".tf" {
			continue
		}

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

		converted = fixJSON(converted)

		enc := json.NewEncoder(jsonFile)
		enc.SetIndent("", "  ")
		if err := enc.Encode(converted); err != nil {
			return err
		}
	}

	return nil
}

// Walk the JSON objects and turn back "provider": ${grafana...} into "provider": "grafana..."
func fixJSON(obj map[string]interface{}) map[string]interface{} {
	for key, val := range obj {
		if key == "provider" || key == "to" {
			if s, ok := val.(string); ok {
				obj[key] = strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}")
			}
		}
		if asMap, ok := val.(map[string]interface{}); ok {
			obj[key] = fixJSON(asMap)
		}
		if asArray, ok := val.([]interface{}); ok {
			for idx, arrayVal := range asArray {
				if m, ok := arrayVal.(map[string]interface{}); ok {
					asArray[idx] = fixJSON(m)
				}
			}
		}
	}
	return obj
}

func traversal(root string, attrs ...string) hcl.Traversal {
	tr := hcl.Traversal{hcl.TraverseRoot{Name: root}}
	for _, attr := range attrs {
		tr = append(tr, hcl.TraverseAttr{Name: attr})
	}
	return tr
}

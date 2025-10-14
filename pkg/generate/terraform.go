package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/tmccombs/hcl2json/convert"
)

func setupTerraform(cfg *Config) (*tfexec.Terraform, error) {
	var err error

	tfVersion := cfg.TerraformInstallConfig.Version
	if tfVersion == nil {
		// Not using latest to avoid unexpected breaking changes
		log.Printf("No Terraform version specified, defaulting to version 1.8.5")
		tfVersion = version.Must(version.NewVersion("1.8.5"))
	}

	// Check if Terraform is already installed
	var execPath string
	if cfg.TerraformInstallConfig.InstallDir != "" {
		finder := fs.ExactVersion{
			Product: product.Terraform,
			Version: tfVersion,
			ExtraPaths: []string{
				cfg.TerraformInstallConfig.InstallDir,
			},
		}

		if execPath, err = finder.Find(context.Background()); err == nil {
			log.Printf("Terraform %s already installed at %s", tfVersion, execPath)
		}
	}

	// Install Terraform if not found
	if execPath == "" {
		log.Printf("Installing Terraform %s", tfVersion)
		installer := &releases.ExactVersion{
			Product:    product.Terraform,
			Version:    tfVersion,
			InstallDir: cfg.TerraformInstallConfig.InstallDir,
		}
		if execPath, err = installer.Install(context.Background()); err != nil {
			return nil, fmt.Errorf("error installing Terraform: %s", err)
		}
	}

	tf, err := tfexec.NewTerraform(cfg.OutputDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("error running NewTerraform: %s", err)
	}

	initOptions := []tfexec.InitOption{
		tfexec.Upgrade(true),
	}
	if cfg.TerraformInstallConfig.PluginDir != "" {
		initOptions = append(initOptions, tfexec.PluginDir(cfg.TerraformInstallConfig.PluginDir))
	}

	err = tf.Init(context.Background(), initOptions...)
	if err != nil {
		return nil, fmt.Errorf("error running Init: %w", err)
	}

	return tf, nil
}

func writeBlocks(filepath string, blocks ...*hclwrite.Block) error {
	return writeBlocksFile(filepath, false, blocks...)
}

func writeBlocksFile(filepath string, new bool, blocks ...*hclwrite.Block) error {
	contents := hclwrite.NewFile()
	if !new {
		if fileBytes, err := os.ReadFile(filepath); err == nil {
			var diags hcl.Diagnostics
			contents, diags = hclwrite.ParseConfig(fileBytes, filepath, hcl.InitialPos)
			if diags.HasErrors() {
				return errors.Join(diags.Errs()...)
			}
		}
	}

	for _, b := range blocks {
		if len(contents.Body().Blocks()) > 0 {
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
func fixJSON(obj map[string]any) map[string]any {
	for key, val := range obj {
		if key == "provider" || key == "to" {
			if s, ok := val.(string); ok {
				obj[key] = strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}")
			}
		}
		if asMap, ok := val.(map[string]any); ok {
			obj[key] = fixJSON(asMap)
		}
		if asArray, ok := val.([]any); ok {
			for idx, arrayVal := range asArray {
				if m, ok := arrayVal.(map[string]any); ok {
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

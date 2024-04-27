package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type outputFormat string

const (
	outputFormatJSON       outputFormat = "json"
	outputFormatHCL        outputFormat = "hcl"
	outputFormatCrossplane outputFormat = "crossplane"
)

var outputFormats = []outputFormat{outputFormatJSON, outputFormatHCL, outputFormatCrossplane}

type config struct {
	outputDir       string
	clobber         bool
	format          outputFormat
	providerVersion string

	grafanaURL  string
	grafanaAuth string

	cloudAccessPolicyToken         string
	cloudOrg                       string
	cloudCreateStackServiceAccount bool
	cloudStackServiceAccountName   string
}

func generate(ctx context.Context, cfg *config) error {
	if _, err := os.Stat(cfg.outputDir); err == nil && cfg.clobber {
		log.Printf("Deleting all files in %s", cfg.outputDir)
		if err := os.RemoveAll(cfg.outputDir); err != nil {
			return fmt.Errorf("failed to delete %s: %s", cfg.outputDir, err)
		}
	} else if err == nil && !cfg.clobber {
		return fmt.Errorf("output dir %q already exists. Use --clobber to delete it", cfg.outputDir)
	}

	log.Printf("Generating resources to %s", cfg.outputDir)
	if err := os.MkdirAll(cfg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %s", cfg.outputDir, err)
	}

	// Generate provider installation block
	providerBlock := hclwrite.NewBlock("terraform", nil)
	requiredProvidersBlock := hclwrite.NewBlock("required_providers", nil)
	requiredProvidersBlock.Body().SetAttributeValue("grafana", cty.ObjectVal(map[string]cty.Value{
		"source":  cty.StringVal("grafana/grafana"),
		"version": cty.StringVal(strings.TrimPrefix(cfg.providerVersion, "v")),
	}))
	providerBlock.Body().AppendBlock(requiredProvidersBlock)
	if err := writeBlocks(filepath.Join(cfg.outputDir, "provider.tf"), providerBlock); err != nil {
		log.Fatal(err)
	}

	// Terraform init to download the provider
	if err := runTerraform(cfg.outputDir, "init"); err != nil {
		return fmt.Errorf("failed to run terraform init: %w", err)
	}

	if cfg.cloudAccessPolicyToken != "" {
		if err := generateCloudResources(ctx, cfg.cloudAccessPolicyToken, cfg.cloudOrg); err != nil {
			return err
		}
	} else {
		if err := generateGrafanaResources(ctx, cfg.grafanaURL, cfg.grafanaAuth); err != nil {
			return err
		}
	}

	if cfg.format == outputFormatJSON {
		return convertToTFJSON(cfg.outputDir)
	}

	return nil
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

type outputFormat string

const (
	outputFormatJSON       outputFormat = "json"
	outputFormatHCL        outputFormat = "hcl"
	outputFormatCrossplane outputFormat = "crossplane"
)

var outputFormats = []outputFormat{outputFormatJSON, outputFormatHCL, outputFormatCrossplane}

type config struct {
	outputDir string
	clobber   bool
	format    outputFormat

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

	if cfg.cloudAccessPolicyToken != "" {
		return generateCloudResources(cfg.cloudAccessPolicyToken, cfg.cloudOrg)
	}

	return generateGrafanaResources(cfg.grafanaURL, cfg.grafanaAuth)
}

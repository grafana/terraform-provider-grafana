package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type outputFormat string

const (
	outputFormatJSON       outputFormat = "json"
	outputFormatHCL        outputFormat = "hcl"
	outputFormatCrossplane outputFormat = "crossplane"
)

var (
	outputFormats         = []outputFormat{outputFormatJSON, outputFormatHCL, outputFormatCrossplane}
	allowedTerraformChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

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
		stacks, err := generateCloudResources(ctx, cfg)
		if err != nil {
			return err
		}

		for _, stack := range stacks {
			if err := generateGrafanaResources(ctx, stack.managementKey, stack.url, "stack-"+stack.slug, false, cfg.outputDir, stack.smURL, stack.smToken); err != nil {
				return err
			}
		}
	}

	if cfg.grafanaAuth != "" {
		grafanaURLParsed, err := url.Parse(cfg.grafanaURL)
		if err != nil {
			return err
		}

		if err := generateGrafanaResources(ctx, cfg.grafanaAuth, cfg.grafanaURL, grafanaURLParsed.Hostname(), true, cfg.outputDir, "", ""); err != nil {
			return err
		}
	}

	if cfg.format == outputFormatJSON {
		return convertToTFJSON(cfg.outputDir)
	}
	if cfg.format == outputFormatCrossplane {
		return errors.New("crossplane output format is not yet supported")
	}

	return nil
}

func generateImportBlocks(ctx context.Context, client *common.Client, listerData any, resources []*common.Resource, outPath, provider string) error {
	// Generate HCL blocks in parallel with a wait group
	wg := sync.WaitGroup{}
	wg.Add(len(resources))
	type result struct {
		resource *common.Resource
		blocks   []*hclwrite.Block
		err      error
	}
	results := make(chan result, len(resources))

	for _, resource := range resources {
		go func(resource *common.Resource) {
			lister := resource.ListIDsFunc
			if lister == nil {
				log.Printf("skipping %s because it does not have a lister\n", resource.Name)
				wg.Done()
				results <- result{
					resource: resource,
				}
				return
			}

			log.Printf("generating %s resources\n", resource.Name)
			ids, err := lister(ctx, client, listerData)
			if err != nil {
				wg.Done()
				results <- result{
					resource: resource,
					err:      err,
				}
				return
			}

			// Write blocks like these
			// import {
			//   to = aws_iot_thing.bar
			//   id = "foo"
			// }
			blocks := make([]*hclwrite.Block, len(ids))
			for i, id := range ids {
				cleanedID := allowedTerraformChars.ReplaceAllString(id, "_")
				if provider != "cloud" {
					cleanedID = strings.ReplaceAll(provider, "-", "_") + "_" + cleanedID
				}

				b := hclwrite.NewBlock("import", nil)
				b.Body().SetAttributeTraversal("provider", traversal("grafana", provider))
				b.Body().SetAttributeTraversal("to", traversal(resource.Name, cleanedID))
				b.Body().SetAttributeValue("id", cty.StringVal(id))

				blocks[i] = b
				// TODO: Match and update existing import blocks
			}

			wg.Done()
			results <- result{
				resource: resource,
				blocks:   blocks,
			}
			log.Printf("finished generating blocks for %s resources\n", resource.Name)
		}(resource)
	}

	// Wait for all results
	wg.Wait()
	close(results)

	// Collect results
	allBlocks := []*hclwrite.Block{}
	for r := range results {
		if r.err != nil {
			return fmt.Errorf("failed to generate %s resources: %w", r.resource.Name, r.err)
		}
		allBlocks = append(allBlocks, r.blocks...)
	}

	if err := writeBlocks(filepath.Join(outPath, provider+"-imports.tf"), allBlocks...); err != nil {
		return err
	}

	generatedFilename := fmt.Sprintf("%s-resources.tf", provider)
	return runTerraform(outPath, "plan", "-generate-config-out="+generatedFilename)
}

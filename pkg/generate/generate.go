package generate

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"
)

var (
	allowedTerraformChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

func Generate(ctx context.Context, cfg *Config) error {
	var err error
	if !filepath.IsAbs(cfg.OutputDir) {
		if cfg.OutputDir, err = filepath.Abs(cfg.OutputDir); err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", cfg.OutputDir, err)
		}
	}

	if _, err := os.Stat(cfg.OutputDir); err == nil && cfg.Clobber {
		log.Printf("Deleting all files in %s", cfg.OutputDir)
		if err := os.RemoveAll(cfg.OutputDir); err != nil {
			return fmt.Errorf("failed to delete %s: %s", cfg.OutputDir, err)
		}
	} else if err == nil && !cfg.Clobber {
		return fmt.Errorf("output dir %q already exists. Use the clobber option to delete it", cfg.OutputDir)
	}

	log.Printf("Generating resources to %s", cfg.OutputDir)
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %s", cfg.OutputDir, err)
	}

	// Generate provider installation block
	providerBlock := hclwrite.NewBlock("terraform", nil)
	requiredProvidersBlock := hclwrite.NewBlock("required_providers", nil)
	requiredProvidersBlock.Body().SetAttributeValue("grafana", cty.ObjectVal(map[string]cty.Value{
		"source":  cty.StringVal("grafana/grafana"),
		"version": cty.StringVal(strings.TrimPrefix(cfg.ProviderVersion, "v")),
	}))
	providerBlock.Body().AppendBlock(requiredProvidersBlock)
	if err := writeBlocks(filepath.Join(cfg.OutputDir, "provider.tf"), providerBlock); err != nil {
		log.Fatal(err)
	}

	tf, err := setupTerraform(cfg)
	// Terraform init to download the provider
	if err != nil {
		return fmt.Errorf("failed to run terraform init: %w", err)
	}
	cfg.Terraform = tf

	if cfg.Cloud != nil {
		log.Printf("Generating cloud resources")
		stacks, err := generateCloudResources(ctx, cfg)
		if err != nil {
			return err
		}

		for _, stack := range stacks {
			stack.name = "stack-" + stack.slug
			if err := generateGrafanaResources(ctx, cfg, stack, false); err != nil {
				return err
			}
		}
	}

	if cfg.Grafana != nil {
		stack := stack{
			managementKey: cfg.Grafana.Auth,
			url:           cfg.Grafana.URL,
			isCloud:       cfg.Grafana.IsGrafanaCloudStack,
			smToken:       cfg.Grafana.SMAccessToken,
			smURL:         cfg.Grafana.SMURL,
			onCallToken:   cfg.Grafana.OnCallAccessToken,
			onCallURL:     cfg.Grafana.OnCallURL,
		}
		log.Printf("Generating Grafana resources")
		if err := generateGrafanaResources(ctx, cfg, stack, true); err != nil {
			return err
		}
	}

	if cfg.Format == OutputFormatCrossplane {
		return convertToCrossplane(cfg)
	}

	if !cfg.OutputCredentials {
		if err := redactCredentials(cfg.OutputDir); err != nil {
			return fmt.Errorf("failed to redact credentials: %w", err)
		}
	}

	if cfg.Format == OutputFormatJSON {
		return convertToTFJSON(cfg.OutputDir)
	}

	return nil
}

func generateImportBlocks(ctx context.Context, client *common.Client, listerData any, resources []*common.Resource, cfg *Config, provider string) error {
	generatedFilename := func(suffix string) string {
		if provider == "" {
			return filepath.Join(cfg.OutputDir, suffix)
		}

		return filepath.Join(cfg.OutputDir, provider+"-"+suffix)
	}

	resources, err := filterResources(resources, cfg.IncludeResources)
	if err != nil {
		return err
	}

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
			sort.Strings(ids)

			// Write blocks like these
			// import {
			//   to = aws_iot_thing.bar
			//   id = "foo"
			// }
			var blocks []*hclwrite.Block
			for _, id := range ids {
				cleanedID := allowedTerraformChars.ReplaceAllString(id, "_")
				if provider != "cloud" {
					cleanedID = strings.ReplaceAll(provider, "-", "_") + "_" + cleanedID
				}

				matched, err := filterResourceByName(resource.Name, cleanedID, cfg.IncludeResources)
				if err != nil {
					wg.Done()
					results <- result{
						resource: resource,
						err:      err,
					}
					return
				}
				if !matched {
					continue
				}

				b := hclwrite.NewBlock("import", nil)
				b.Body().SetAttributeTraversal("to", traversal(resource.Name, cleanedID))
				b.Body().SetAttributeValue("id", cty.StringVal(id))
				if provider != "" {
					b.Body().SetAttributeTraversal("provider", traversal("grafana", provider))
				}

				blocks = append(blocks, b)
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

	resultsSlice := []result{}
	for r := range results {
		if r.err != nil {
			return fmt.Errorf("failed to generate %s resources: %w", r.resource.Name, r.err)
		}
		resultsSlice = append(resultsSlice, r)
	}
	sort.Slice(resultsSlice, func(i, j int) bool {
		return resultsSlice[i].resource.Name < resultsSlice[j].resource.Name
	})

	// Collect results
	allBlocks := []*hclwrite.Block{}
	for _, r := range resultsSlice {
		allBlocks = append(allBlocks, r.blocks...)
	}

	if len(allBlocks) == 0 {
		if err := os.WriteFile(generatedFilename("resources.tf"), []byte("# No resources were found\n"), 0600); err != nil {
			return err
		}
		if err := os.WriteFile(generatedFilename("imports.tf"), []byte("# No resources were found\n"), 0600); err != nil {
			return err
		}
		return nil
	}

	if err := writeBlocks(generatedFilename("imports.tf"), allBlocks...); err != nil {
		return err
	}
	_, err = cfg.Terraform.Plan(ctx, tfexec.GenerateConfigOut(generatedFilename("resources.tf")))
	if err != nil {
		return fmt.Errorf("failed to generate resources: %w", err)
	}
	return sortResourcesFile(generatedFilename("resources.tf"))
}

func filterResources(resources []*common.Resource, includedResources []string) ([]*common.Resource, error) {
	if len(includedResources) == 0 {
		return resources, nil
	}

	filteredResources := []*common.Resource{}
	allowedResourceTypes := []string{}
	for _, included := range includedResources {
		if !strings.Contains(included, ".") {
			return nil, fmt.Errorf("included resource %q is not in the format <type>.<name>", included)
		}
		allowedResourceTypes = append(allowedResourceTypes, strings.Split(included, ".")[0])
	}

	for _, resource := range resources {
		for _, allowedResourceType := range allowedResourceTypes {
			matched, err := filepath.Match(allowedResourceType, resource.Name)
			if err != nil {
				return nil, err
			}
			if matched {
				filteredResources = append(filteredResources, resource)
				break
			}
		}
	}
	return filteredResources, nil
}

func filterResourceByName(resourceType, resourceName string, includedResources []string) (bool, error) {
	if len(includedResources) == 0 {
		return true, nil
	}

	for _, included := range includedResources {
		matched, err := filepath.Match(included, resourceType+"."+resourceName)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

package generate

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/generate/postprocessing"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/generate/utils"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"
)

// NonCriticalError is an error that is not critical to the generation process.
// It can be handled differently by the caller.
type NonCriticalError interface {
	NonCriticalError()
}

// ResourceError is an error that occurred while generating a resource.
type ResourceError struct {
	Resource *common.Resource
	Err      error
}

func (e ResourceError) Error() string {
	return fmt.Sprintf("resource %s: %v", e.Resource.Name, e.Err)
}

func (ResourceError) NonCriticalError() {}

type NonCriticalGenerationFailure struct{ error }

func (f NonCriticalGenerationFailure) NonCriticalError() {}

type GenerationSuccess struct {
	Resource *common.Resource
	Blocks   int
}

type GenerationResult struct {
	Success []GenerationSuccess
	Errors  []error
}

func (r GenerationResult) Blocks() int {
	blocks := 0
	for _, s := range r.Success {
		blocks += s.Blocks
	}
	return blocks
}

func failure(err error) GenerationResult {
	return GenerationResult{
		Errors: []error{err},
	}
}

func failuref(format string, args ...any) GenerationResult {
	return failure(fmt.Errorf(format, args...))
}

func Generate(ctx context.Context, cfg *Config) GenerationResult {
	var err error
	if !filepath.IsAbs(cfg.OutputDir) {
		if cfg.OutputDir, err = filepath.Abs(cfg.OutputDir); err != nil {
			return failuref("failed to get absolute path for %s: %w", cfg.OutputDir, err)
		}
	}

	if _, err := os.Stat(cfg.OutputDir); err == nil && cfg.Clobber {
		log.Printf("Deleting all files in %s", cfg.OutputDir)
		if err := os.RemoveAll(cfg.OutputDir); err != nil {
			return failuref("failed to delete %s: %s", cfg.OutputDir, err)
		}
	} else if err == nil && !cfg.Clobber {
		return failuref("output dir %q already exists. Use the clobber option to delete it", cfg.OutputDir)
	}

	log.Printf("Generating resources to %s", cfg.OutputDir)
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return failuref("failed to create output directory %s: %s", cfg.OutputDir, err)
	}

	// Enable "unsensitive" mode for the provider
	os.Setenv(provider.EnableGenerateEnvVar, "true")
	defer os.Unsetenv(provider.EnableGenerateEnvVar)
	if err := os.WriteFile(filepath.Join(cfg.OutputDir, provider.EnableGenerateMarkerFile), []byte("unsensitive!"), 0600); err != nil {
		return failuref("failed to write marker file: %w", err)
	}
	defer os.Remove(filepath.Join(cfg.OutputDir, provider.EnableGenerateMarkerFile))

	// Generate provider installation block
	providerBlock := hclwrite.NewBlock("terraform", nil)
	requiredProvidersBlock := hclwrite.NewBlock("required_providers", nil)
	requiredProvidersBlock.Body().SetAttributeValue("grafana", cty.ObjectVal(map[string]cty.Value{
		"source":  cty.StringVal("grafana/grafana"),
		"version": cty.StringVal(strings.TrimPrefix(cfg.ProviderVersion, "v")),
	}))
	providerBlock.Body().AppendBlock(requiredProvidersBlock)
	if err := writeBlocks(filepath.Join(cfg.OutputDir, "provider.tf"), providerBlock); err != nil {
		return failure(err)
	}

	tf, err := setupTerraform(cfg)
	// Terraform init to download the provider
	if err != nil {
		return failuref("failed to run terraform init: %w", err)
	}
	cfg.Terraform = tf

	var returnResult GenerationResult
	if cfg.Cloud != nil {
		log.Printf("Generating cloud resources")
		var stacks []stack
		stacks, returnResult = generateCloudResources(ctx, cfg)

		for _, stack := range stacks {
			stack.name = "stack-" + stack.slug
			stackResult := generateGrafanaResources(ctx, cfg, stack, false)
			returnResult.Success = append(returnResult.Success, stackResult.Success...)
			returnResult.Errors = append(returnResult.Errors, stackResult.Errors...)
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
		returnResult = generateGrafanaResources(ctx, cfg, stack, true)
	}

	if !cfg.OutputCredentials && cfg.Format != OutputFormatCrossplane {
		if err := postprocessing.RedactCredentials(cfg.OutputDir); err != nil {
			return failuref("failed to redact credentials: %w", err)
		}
	}

	if returnResult.Blocks() == 0 {
		if err := os.WriteFile(filepath.Join(cfg.OutputDir, "resources.tf"), []byte("# No resources were found\n"), 0600); err != nil {
			return failure(err)
		}
		if err := os.WriteFile(filepath.Join(cfg.OutputDir, "imports.tf"), []byte("# No resources were found\n"), 0600); err != nil {
			return failure(err)
		}
		return returnResult
	}

	if cfg.Format == OutputFormatCrossplane {
		if err := convertToCrossplane(cfg); err != nil {
			return failure(err)
		}
		return returnResult
	}

	if cfg.Format == OutputFormatJSON {
		if err := convertToTFJSON(cfg.OutputDir); err != nil {
			return failure(err)
		}
	}

	return returnResult
}

func generateImportBlocks(ctx context.Context, client *common.Client, listerData any, resources []*common.Resource, cfg *Config, provider string) GenerationResult {
	generatedFilename := func(suffix string) string {
		if provider == "" {
			return filepath.Join(cfg.OutputDir, suffix)
		}

		return filepath.Join(cfg.OutputDir, provider+"-"+suffix)
	}

	resources, err := filterResources(resources, cfg.IncludeResources)
	if err != nil {
		return failure(err)
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
				results <- result{
					resource: resource,
				}
				wg.Done()
				return
			}

			log.Printf("generating %s resources\n", resource.Name)
			listedIDs, err := lister(ctx, client, listerData)
			if err != nil {
				results <- result{
					resource: resource,
					err:      err,
				}
				wg.Done()
				return
			}

			// Make sure IDs are unique. If an API returns the same ID multiple times for any reason, we only want to import it once.
			idMap := map[string]struct{}{}
			for _, id := range listedIDs {
				idMap[id] = struct{}{}
			}
			ids := []string{}
			for id := range idMap {
				ids = append(ids, id)
			}
			sort.Strings(ids)

			// Write blocks like these
			// import {
			//   to = aws_iot_thing.bar
			//   id = "foo"
			// }
			var blocks []*hclwrite.Block
			for _, id := range ids {
				matched, err := filterResourceByName(resource.Name, id, cfg.IncludeResources)
				if err != nil {
					results <- result{
						resource: resource,
						err:      err,
					}
					wg.Done()
					return
				}
				if !matched {
					continue
				}

				if provider != "cloud" && provider != "" {
					id = provider + "_" + id
				}
				resourceName := postprocessing.CleanResourceName(id)

				b := hclwrite.NewBlock("import", nil)
				b.Body().SetAttributeTraversal("to", traversal(resource.Name, resourceName))
				b.Body().SetAttributeValue("id", cty.StringVal(id))
				if provider != "" {
					b.Body().SetAttributeTraversal("provider", traversal("grafana", provider))
				}

				blocks = append(blocks, b)
			}

			results <- result{
				resource: resource,
				blocks:   blocks,
			}
			wg.Done()
			log.Printf("finished generating blocks for %s resources\n", resource.Name)
		}(resource)
	}

	// Wait for all results
	wg.Wait()
	close(results)

	returnResult := GenerationResult{}
	resultsSlice := []result{}
	for r := range results {
		if r.err != nil {
			returnResult.Errors = append(returnResult.Errors, ResourceError{
				Resource: r.resource,
				Err:      r.err,
			})
		} else {
			resultsSlice = append(resultsSlice, r)
			returnResult.Success = append(returnResult.Success, GenerationSuccess{
				Resource: r.resource,
				Blocks:   len(r.blocks),
			})
		}
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
		return returnResult
	}

	if err := writeBlocks(generatedFilename("imports.tf"), allBlocks...); err != nil {
		return failure(err)
	}
	_, err = cfg.Terraform.Plan(ctx, tfexec.GenerateConfigOut(generatedFilename("resources.tf")))
	if err != nil && !strings.Contains(err.Error(), "Missing required argument") {
		// If resources.tf was created and is not empty, return the error as a "non-critical" error
		if stat, statErr := os.Stat(generatedFilename("resources.tf")); statErr == nil && stat.Size() > 0 {
			returnResult.Errors = append(returnResult.Errors, NonCriticalGenerationFailure{err})
		} else {
			return failuref("failed to generate resources: %w", err)
		}
	}

	for _, err := range []error{
		postprocessing.ReplaceNullSensitiveAttributes(generatedFilename("resources.tf")),
		removeOrphanedImports(generatedFilename("imports.tf"), generatedFilename("resources.tf")),
		postprocessing.UsePreferredResourceNames(generatedFilename("resources.tf"), generatedFilename("imports.tf")),
		sortResourcesFile(generatedFilename("resources.tf")),
		postprocessing.WrapJSONFieldsInFunction(generatedFilename("resources.tf")),
	} {
		if err != nil {
			return failure(err)
		}
	}

	return returnResult
}

// removeOrphanedImports removes import blocks that do not have a corresponding resource block in the resources file.
// These happen when the Terraform plan command has failed for some resources.
func removeOrphanedImports(importsFile, resourcesFile string) error {
	imports, err := utils.ReadHCLFile(importsFile)
	if err != nil {
		return err
	}

	resources, err := utils.ReadHCLFile(resourcesFile)
	if err != nil {
		return err
	}

	resourcesMap := map[string]struct{}{}
	for _, block := range resources.Body().Blocks() {
		if block.Type() != "resource" {
			continue
		}

		resourcesMap[strings.Join(block.Labels(), ".")] = struct{}{}
	}

	for _, block := range imports.Body().Blocks() {
		if block.Type() != "import" {
			continue
		}

		importTo := strings.TrimSpace(string(block.Body().GetAttribute("to").Expr().BuildTokens(nil).Bytes()))
		if _, ok := resourcesMap[importTo]; !ok {
			imports.Body().RemoveBlock(block)
		}
	}

	return writeBlocksFile(importsFile, true, imports.Body().Blocks()...)
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

func filterResourceByName(resourceType, resourceID string, includedResources []string) (bool, error) {
	if len(includedResources) == 0 {
		return true, nil
	}

	for _, included := range includedResources {
		matched, err := filepath.Match(included, resourceType+"."+resourceID)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

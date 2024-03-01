package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v2/pkg/provider"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
)

var allowedTerraformChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func main() {
	ctx := context.Background()

	outPath := os.Getenv("TFGEN_OUT_PATH") // TODO: CLI flag
	if outPath == "" {
		log.Fatal("TFGEN_OUT_PATH environment variable must be set")
	}
	outPath, err := filepath.Abs(outPath)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Remove this once we can do updates
	if err := os.RemoveAll(outPath); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(outPath, 0755); err != nil {
		log.Fatal(err)
	}

	// Install provider
	providerFilePath := filepath.Join(outPath, "provider.tf")
	providerFile, err := os.Create(providerFilePath)
	if err != nil {
		log.Fatal(err)
	}
	providerContents := hclwrite.NewFile()
	providerBlock := hclwrite.NewBlock("terraform", nil)
	requiredProvidersBlock := hclwrite.NewBlock("required_providers", nil)
	requiredProvidersBlock.Body().SetAttributeValue("grafana", cty.ObjectVal(map[string]cty.Value{
		"source":  cty.StringVal("grafana/grafana"),
		"version": cty.StringVal("2.12.2"), // TODO: Get latest (or current??)
	}))

	providerBlock.Body().AppendBlock(requiredProvidersBlock)
	providerContents.Body().AppendBlock(providerBlock)
	if _, err := providerContents.WriteTo(providerFile); err != nil {
		log.Fatal(err)
	}
	if err := providerFile.Close(); err != nil {
		log.Fatal(err)
	}

	// tf init
	initCmd := exec.Command("terraform", "init")
	initCmd.Dir = outPath
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		log.Fatal(err)
	}

	cloudAPIKey := os.Getenv("GRAFANA_CLOUD_API_KEY") // TODO: CLI flag
	if cloudAPIKey != "" {
		orgSlug := os.Getenv("GRAFANA_CLOUD_ORG") // TODO: CLI flag
		if orgSlug == "" {
			log.Fatal("GRAFANA_CLOUD_ORG environment variable must be set")
		}
		err := genCloudResources(ctx, cloudAPIKey, orgSlug, outPath)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func genCloudResources(ctx context.Context, apiKey, orgSlug, outPath string) error {
	// Gen provider
	cloudProviderFilePath := filepath.Join(outPath, "cloud-provider.tf")
	cloudProviderFile, err := os.Create(cloudProviderFilePath)
	if err != nil {
		return err
	}
	cloudProviderContents := hclwrite.NewFile()
	providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
	providerBlock.Body().SetAttributeValue("alias", cty.StringVal("cloud"))
	providerBlock.Body().SetAttributeValue("cloud_access_policy_token", cty.StringVal(apiKey))
	cloudProviderContents.Body().AppendBlock(providerBlock)
	if _, err := cloudProviderContents.WriteTo(cloudProviderFile); err != nil {
		return err
	}
	if err := cloudProviderFile.Close(); err != nil {
		return err
	}

	// Generate imports
	config := provider.FrameworkProviderConfig{
		CloudAccessPolicyToken: types.StringValue(apiKey),
	}
	if err := config.SetDefaults(); err != nil {
		return err
	}

	client, err := provider.CreateClients(config)
	if err != nil {
		return err
	}

	cloudResources := cloud.Resources
	cache := sync.Map{}
	cache.Store("org", orgSlug)

	// TODO: Parse and read file
	f := hclwrite.NewFile()

	// Generate HCL blocks in parallel with a wait group
	wg := sync.WaitGroup{}
	wg.Add(len(cloudResources))
	type result struct {
		resource *common.Resource
		blocks   []*hclwrite.Block
		err      error
	}
	results := make(chan result, len(cloudResources))

	for _, resource := range cloudResources {
		go func(resource *common.Resource) {
			lister := resource.ListIDsFunc
			if lister == nil {
				log.Printf("Skipping %s because it does not have a lister\n", resource.Name)
				wg.Done()
				results <- result{
					resource: resource,
				}
				return
			}

			log.Printf("Generating %s resources\n", resource.Name)
			ids, err := lister(ctx, &cache, client)
			if err != nil {
				wg.Done()
				results <- result{
					resource: resource,
					err:      err,
				}
				return
			}

			// Write blocks like these
			//import {
			//   to = aws_iot_thing.bar
			//   id = "foo"
			// }
			blocks := make([]*hclwrite.Block, len(ids))
			for i, id := range ids {
				b := hclwrite.NewBlock("import", nil)
				b.Body().SetAttributeTraversal("provider", hcl.Traversal{
					hcl.TraverseRoot{
						Name: "grafana",
					},
					hcl.TraverseAttr{
						Name: "cloud",
					},
				})
				b.Body().SetAttributeTraversal("to", hcl.Traversal{
					hcl.TraverseRoot{
						Name: resource.Name,
					},
					hcl.TraverseAttr{
						Name: allowedTerraformChars.ReplaceAllString(id, "_"),
					},
				})
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
	for r := range results {
		if r.err != nil {
			return fmt.Errorf("failed to generate %s resources: %w", r.resource.Name, r.err)
		}
		for _, b := range r.blocks {
			f.Body().AppendBlock(b)
		}
	}

	importsFilePath := filepath.Join(outPath, "cloud-imports.tf")
	importsFile, err := os.Create(importsFilePath)
	if err != nil {
		return err
	}
	if _, err := f.WriteTo(importsFile); err != nil {
		return err
	}
	if err := importsFile.Close(); err != nil {
		return err
	}

	genCommand := exec.Command("terraform", "plan", "-generate-config-out=cloud-resources.tf")
	genCommand.Dir = outPath
	genCommand.Stdout = os.Stdout
	genCommand.Stderr = os.Stderr
	if err := genCommand.Run(); err != nil {
		return err
	}

	return nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v2/pkg/provider"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
)

var allowedTerraformChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

const managementServiceAccountName = "tfgen-management"

type stack struct {
	slug          string
	url           string
	managementKey string
}

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
	providerBlock := hclwrite.NewBlock("terraform", nil)
	requiredProvidersBlock := hclwrite.NewBlock("required_providers", nil)
	requiredProvidersBlock.Body().SetAttributeValue("grafana", cty.ObjectVal(map[string]cty.Value{
		"source":  cty.StringVal("grafana/grafana"),
		"version": cty.StringVal("2.12.2"), // TODO: Get latest (or current??)
	}))

	providerBlock.Body().AppendBlock(requiredProvidersBlock)
	if err := writeBlocks(filepath.Join(outPath, "provider.tf"), []*hclwrite.Block{providerBlock}); err != nil {
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
		stacks, err := genCloudResources(ctx, cloudAPIKey, orgSlug, os.Getenv("GEN_ENTRYPOINT_INTO_STACKS") == "true", outPath)
		if err != nil {
			log.Fatal(err)
		}

		for _, stack := range stacks {
			if err := genGrafanaResources(ctx, stack.managementKey, stack.url, "stack-"+stack.slug, false, outPath); err != nil {
				log.Fatal(err)
			}
		}
		return
	}

	// Else
	grafanaAuth := os.Getenv("GRAFANA_AUTH") // TODO: CLI flag
	grafanaUrl := os.Getenv("GRAFANA_URL")   // TODO: CLI flag
	if grafanaAuth == "" || grafanaUrl == "" {
		log.Fatal("GRAFANA_AUTH and GRAFANA_URL environment variables must be set")
	}

	grafanaUrlParsed, err := url.Parse(grafanaUrl)
	if err != nil {
		log.Fatal(err)
	}

	if err := genGrafanaResources(ctx, grafanaAuth, grafanaUrl, grafanaUrlParsed.Host, true, outPath); err != nil {
		log.Fatal(err)
	}
}

func genCloudResources(ctx context.Context, apiKey, orgSlug string, addManagementKey bool, outPath string) ([]stack, error) {
	// Gen provider
	providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
	providerBlock.Body().SetAttributeValue("alias", cty.StringVal("cloud"))
	providerBlock.Body().SetAttributeValue("cloud_access_policy_token", cty.StringVal(apiKey))
	if err := writeBlocks(filepath.Join(outPath, "cloud-provider.tf"), []*hclwrite.Block{providerBlock}); err != nil {
		return nil, err
	}

	// Generate imports
	config := provider.FrameworkProviderConfig{
		CloudAccessPolicyToken: types.StringValue(apiKey),
	}
	if err := config.SetDefaults(); err != nil {
		return nil, err
	}

	client, err := provider.CreateClients(config)
	if err != nil {
		return nil, err
	}
	cloudClient := client.GrafanaCloudAPI

	cache := sync.Map{}
	cache.Store("org", orgSlug)

	if err := generateImportBlocks(ctx, client, &cache, cloud.Resources, outPath, "cloud"); err != nil {
		return nil, err
	}

	if !addManagementKey {
		return nil, nil
	}

	// Add management service account (grafana_cloud_stack_service_account)
	// This one needs to be applied to prevent https://github.com/grafana/terraform-provider-grafana/issues/960
	stacks, _, err := cloudClient.InstancesAPI.GetInstances(ctx).Execute()
	if err != nil {
		return nil, err
	}
	stacksBySlug := map[string]gcom.FormattedApiInstance{}

	for _, stack := range stacks.Items {
		stacksBySlug[stack.Slug] = stack
		// TODO: Make sure the instance is not paused (by curling it)
		// When the instance is paused, we can't create service accounts in it

		tempClient, cleanup, err := cloud.CreateTemporaryStackGrafanaClient(ctx, cloudClient, stack.Slug, "temp-sa-")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary client for stack %q: %w", stack.Slug, err)
		}

		serviceAccountsResp, err := tempClient.ServiceAccounts.SearchOrgServiceAccountsWithPaging(service_accounts.NewSearchOrgServiceAccountsWithPagingParams())
		if err != nil {
			return nil, fmt.Errorf("failed to search service accounts for stack %q: %w", stack.Slug, err)
		}
		for _, sa := range serviceAccountsResp.Payload.ServiceAccounts {
			if sa.Name == managementServiceAccountName {
				log.Printf("found existing management service account (%s) in stack %q\n", managementServiceAccountName, stack.Slug)
				// Delete the SA to recreate it via TF
				_, err := tempClient.ServiceAccounts.DeleteServiceAccount(sa.ID)
				if err != nil {
					return nil, fmt.Errorf("failed to delete existing management service account (%s) in stack %q: %w", managementServiceAccountName, stack.Slug, err)
				}
				break
			}
		}

		if err := cleanup(); err != nil {
			return nil, fmt.Errorf("failed to cleanup temporary client for stack %q: %w", stack.Slug, err)
		}

		// Create the management service account
		saBlock := hclwrite.NewBlock("resource", []string{"grafana_cloud_stack_service_account", stack.Slug})
		saBlock.Body().SetAttributeTraversal("provider", traversal("grafana", "cloud"))
		saBlock.Body().SetAttributeValue("stack_slug", cty.StringVal(stack.Slug))
		saBlock.Body().SetAttributeValue("name", cty.StringVal(managementServiceAccountName))
		saBlock.Body().SetAttributeValue("role", cty.StringVal("Admin"))

		saTokenBlock := hclwrite.NewBlock("resource", []string{"grafana_cloud_stack_service_account_token", stack.Slug})
		saTokenBlock.Body().SetAttributeTraversal("provider", traversal("grafana", "cloud"))
		saTokenBlock.Body().SetAttributeValue("stack_slug", cty.StringVal(stack.Slug))
		saTokenBlock.Body().SetAttributeTraversal("service_account_id", traversal("grafana_cloud_stack_service_account", stack.Slug, "id"))
		saTokenBlock.Body().SetAttributeValue("name", cty.StringVal(managementServiceAccountName))

		providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
		providerBlock.Body().SetAttributeValue("alias", cty.StringVal("stack-"+stack.Slug))
		providerBlock.Body().SetAttributeTraversal("url", traversal("grafana_cloud_stack", stack.Slug, "url"))
		providerBlock.Body().SetAttributeTraversal("auth", traversal("grafana_cloud_stack_service_account_token", stack.Slug, "key"))

		if err := writeBlocks(filepath.Join(outPath, fmt.Sprintf("stack-%s-provider.tf", stack.Slug)), []*hclwrite.Block{saBlock, saTokenBlock, providerBlock}); err != nil {
			return nil, fmt.Errorf("failed to write management service account blocks for stack %q: %w", stack.Slug, err)
		}

		// TODO: Terraform apply -t sa+token
		// Then go into the state and find the management key
		applyCommand := exec.Command("terraform", "apply", "-auto-approve", "-target=grafana_cloud_stack_service_account."+stack.Slug, "-target=grafana_cloud_stack_service_account_token."+stack.Slug)
		applyCommand.Dir = outPath
		applyCommand.Stdout = os.Stdout
		applyCommand.Stderr = os.Stderr
		if err := applyCommand.Run(); err != nil {
			return nil, fmt.Errorf("failed to apply management service account blocks for stack %q: %w", stack.Slug, err)
		}
	}

	managedStacks := []stack{}
	stateCmd := exec.Command("terraform", "show", "-json")
	stateCmd.Dir = outPath
	state, err := stateCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform state: %w", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(state, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse terraform state: %w", err)
	}
	values := parsed["values"].(map[string]interface{})
	rootModule := values["root_module"].(map[string]interface{})
	resources := rootModule["resources"].([]interface{})
	for _, resource := range resources {
		resource := resource.(map[string]interface{})
		if resource["type"].(string) == "grafana_cloud_stack_service_account_token" {
			slug := resource["values"].(map[string]interface{})["stack_slug"].(string)
			managedStacks = append(managedStacks, stack{
				slug:          slug,
				url:           stacksBySlug[slug].Url,
				managementKey: resource["values"].(map[string]interface{})["key"].(string),
			})
		}
	}

	return managedStacks, nil
}

func writeBlocks(filepath string, blocks []*hclwrite.Block) error {
	contents := hclwrite.NewFile()
	for _, b := range blocks {
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

func genGrafanaResources(ctx context.Context, auth, url, stackName string, genProvider bool, outPath string) error {
	if genProvider {
		providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
		providerBlock.Body().SetAttributeValue("alias", cty.StringVal(stackName))
		providerBlock.Body().SetAttributeValue("url", cty.StringVal(url))
		providerBlock.Body().SetAttributeValue("auth", cty.StringVal(auth))
		if err := writeBlocks(filepath.Join(outPath, stackName+"-provider.tf"), []*hclwrite.Block{providerBlock}); err != nil {
			return err
		}
	}

	// Generate resources
	config := provider.FrameworkProviderConfig{
		URL:  types.StringValue(url),
		Auth: types.StringValue(auth),
	}
	if err := config.SetDefaults(); err != nil {
		return err
	}

	client, err := provider.CreateClients(config)
	if err != nil {
		return err
	}

	cache := sync.Map{}

	if err := generateImportBlocks(ctx, client, &cache, grafana.Resources, outPath, stackName); err != nil {
		return err
	}

	return nil
}

func generateImportBlocks(ctx context.Context, client *common.Client, cache *sync.Map, resources []*common.Resource, outPath, provider string) error {
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
				log.Printf("Skipping %s because it does not have a lister\n", resource.Name)
				wg.Done()
				results <- result{
					resource: resource,
				}
				return
			}

			log.Printf("Generating %s resources\n", resource.Name)
			ids, err := lister(ctx, cache, client)
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
				b.Body().SetAttributeTraversal("provider", traversal("grafana", provider))
				b.Body().SetAttributeTraversal("to", traversal(resource.Name, allowedTerraformChars.ReplaceAllString(id, "_")))
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

	if err := writeBlocks(filepath.Join(outPath, provider+"-imports.tf"), allBlocks); err != nil {
		return err
	}

	genCommand := exec.Command("terraform", "plan", fmt.Sprintf("-generate-config-out=%s-resources.tf", provider))
	genCommand.Dir = outPath
	genCommand.Stdout = os.Stdout
	genCommand.Stderr = os.Stderr
	if err := genCommand.Run(); err != nil {
		return err
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

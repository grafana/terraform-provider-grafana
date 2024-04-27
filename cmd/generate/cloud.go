package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v2/pkg/provider"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
)

type stack struct {
	slug          string
	url           string
	managementKey string
	smURL         string
	smToken       string
}

func generateCloudResources(ctx context.Context, cfg *config) ([]stack, error) {
	// Gen provider
	providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
	providerBlock.Body().SetAttributeValue("alias", cty.StringVal("cloud"))
	providerBlock.Body().SetAttributeValue("cloud_access_policy_token", cty.StringVal(cfg.cloudAccessPolicyToken))
	if err := writeBlocks(filepath.Join(cfg.outputDir, "cloud-provider.tf"), providerBlock); err != nil {
		return nil, err
	}

	// Generate imports
	config := provider.ProviderConfig{
		CloudAccessPolicyToken: types.StringValue(cfg.cloudAccessPolicyToken),
	}
	if err := config.SetDefaults(); err != nil {
		return nil, err
	}

	client, err := provider.CreateClients(config)
	if err != nil {
		return nil, err
	}
	cloudClient := client.GrafanaCloudAPI

	stacks, _, err := cloudClient.InstancesAPI.GetInstances(ctx).Execute()
	if err != nil {
		return nil, err
	}

	// Cleanup SAs
	managementServiceAccountName := cfg.cloudStackServiceAccountName
	smAccessPolicy := func(stack gcom.FormattedApiInstance) string {
		return stack.Slug + "-sm-metrics-publish"
	}
	if cfg.cloudCreateStackServiceAccount {
		for _, stack := range stacks.Items {
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

			// Delete existing SM installation
			resp, _, err := cloudClient.AccesspoliciesAPI.GetAccessPolicies(ctx).OrgId(int32(stack.OrgId)).Region(stack.RegionSlug).Execute()
			if err != nil {
				return nil, err
			}
			for _, policy := range resp.Items {
				if policy.Name == smAccessPolicy(stack) {
					log.Printf("found existing SM installation (%s) in stack %q\n", smAccessPolicy(stack), stack.Slug)
					_, _, err := cloudClient.AccesspoliciesAPI.DeleteAccessPolicy(ctx, *policy.Id).XRequestId("tf-gen").OrgId(int32(stack.OrgId)).Region(stack.RegionSlug).Execute()
					if err != nil {
						return nil, fmt.Errorf("failed to delete existing SM installation (%s) in stack %q: %w", smAccessPolicy(stack), stack.Slug, err)
					}
					break
				}
			}
		}
	}

	cache := sync.Map{}
	cache.Store("org", cfg.cloudOrg)

	if err := generateImportBlocks(ctx, client, &cache, cloud.Resources, cfg.outputDir, "cloud"); err != nil {
		return nil, err
	}

	if !cfg.cloudCreateStackServiceAccount {
		return nil, nil
	}

	// Add management service account (grafana_cloud_stack_service_account)
	// This one needs to be applied to prevent https://github.com/grafana/terraform-provider-grafana/issues/960
	stacksBySlug := map[string]gcom.FormattedApiInstance{}
	stacksByID := map[int]gcom.FormattedApiInstance{}
	for _, stack := range stacks.Items {
		stacksBySlug[stack.Slug] = stack
		stacksByID[int(stack.Id)] = stack
		// TODO: Make sure the instance is not paused (by curling it)
		// When the instance is paused, we can't create service accounts in it

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

		// Create the SM installation
		policyResourceName := stack.Slug + "_sm_metrics_publish"
		smInstallationMetricsPublishBlock := hclwrite.NewBlock("resource", []string{"grafana_cloud_access_policy", policyResourceName})
		smInstallationMetricsPublishBlock.Body().SetAttributeTraversal("provider", traversal("grafana", "cloud"))
		smInstallationMetricsPublishBlock.Body().SetAttributeValue("region", cty.StringVal(stack.RegionSlug))
		smInstallationMetricsPublishBlock.Body().SetAttributeValue("name", cty.StringVal(smAccessPolicy(stack)))
		smInstallationMetricsPublishBlock.Body().SetAttributeValue("scopes", cty.ListVal([]cty.Value{cty.StringVal("metrics:write"), cty.StringVal("stacks:read")}))
		smInstallationMetricsPublishRealmBlock := hclwrite.NewBlock("realm", nil)
		smInstallationMetricsPublishRealmBlock.Body().SetAttributeValue("type", cty.StringVal("stack"))
		smInstallationMetricsPublishRealmBlock.Body().SetAttributeTraversal("identifier", traversal("grafana_cloud_stack", stack.Slug, "id"))
		smInstallationMetricsPublishBlock.Body().AppendBlock(smInstallationMetricsPublishRealmBlock)

		smInstallationTokenBlock := hclwrite.NewBlock("resource", []string{"grafana_cloud_access_policy_token", policyResourceName})
		smInstallationTokenBlock.Body().SetAttributeTraversal("provider", traversal("grafana", "cloud"))
		smInstallationTokenBlock.Body().SetAttributeValue("region", cty.StringVal(stack.RegionSlug))
		smInstallationTokenBlock.Body().SetAttributeTraversal("access_policy_id", traversal("grafana_cloud_access_policy", policyResourceName, "policy_id"))
		smInstallationTokenBlock.Body().SetAttributeValue("name", cty.StringVal(smAccessPolicy(stack)))

		smInstallationBlock := hclwrite.NewBlock("resource", []string{"grafana_synthetic_monitoring_installation", stack.Slug})
		smInstallationBlock.Body().SetAttributeTraversal("provider", traversal("grafana", "cloud"))
		smInstallationBlock.Body().SetAttributeTraversal("stack_id", traversal("grafana_cloud_stack", stack.Slug, "id"))
		smInstallationBlock.Body().SetAttributeTraversal("metrics_publisher_key", traversal("grafana_cloud_access_policy_token", policyResourceName, "token"))

		providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
		providerBlock.Body().SetAttributeValue("alias", cty.StringVal("stack-"+stack.Slug))
		providerBlock.Body().SetAttributeTraversal("url", traversal("grafana_cloud_stack", stack.Slug, "url"))
		providerBlock.Body().SetAttributeTraversal("auth", traversal("grafana_cloud_stack_service_account_token", stack.Slug, "key"))
		providerBlock.Body().SetAttributeTraversal("sm_access_token", traversal("grafana_synthetic_monitoring_installation", stack.Slug, "sm_access_token"))
		providerBlock.Body().SetAttributeTraversal("sm_url", traversal("grafana_synthetic_monitoring_installation", stack.Slug, "stack_sm_api_url"))

		if err := writeBlocks(filepath.Join(cfg.outputDir, fmt.Sprintf("stack-%s-provider.tf", stack.Slug)), saBlock, saTokenBlock, smInstallationMetricsPublishBlock, smInstallationTokenBlock, smInstallationBlock, providerBlock); err != nil {
			return nil, fmt.Errorf("failed to write management service account blocks for stack %q: %w", stack.Slug, err)
		}

		// Apply then go into the state and find the management key
		err := runTerraform("apply", "-auto-approve",
			"-target=grafana_cloud_stack_service_account."+stack.Slug,
			"-target=grafana_cloud_stack_service_account_token."+stack.Slug,
			"-target=grafana_cloud_access_policy."+policyResourceName,
			"-target=grafana_cloud_access_policy_token."+policyResourceName,
			"-target=grafana_synthetic_monitoring_installation."+stack.Slug,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to apply management service account blocks for stack %q: %w", stack.Slug, err)
		}
	}

	managedStacks := []stack{}
	state, err := getState(cfg.outputDir)
	if err != nil {
		return nil, err
	}
	stacksMap := map[string]stack{}
	for _, resource := range state.resources() {
		if resource.resourceType() == "grafana_cloud_stack_service_account_token" {
			slug := resource.values()["stack_slug"].(string)
			stack := stacksMap[slug]
			stack.slug = slug
			stack.url = stacksBySlug[slug].Url
			stack.managementKey = resource.values()["key"].(string)
			stacksMap[slug] = stack
		}
		if resource.resourceType() == "grafana_synthetic_monitoring_installation" {
			idStr := resource.values()["stack_id"].(string)
			slug := idStr
			if id, err := strconv.Atoi(idStr); err == nil {
				slug = stacksByID[id].Slug
			}
			stack := stacksMap[slug]
			stack.smToken = resource.values()["sm_access_token"].(string)
			stack.smURL = resource.values()["stack_sm_api_url"].(string)
			stacksMap[slug] = stack
		}
	}

	for _, stack := range stacksMap {
		managedStacks = append(managedStacks, stack)
	}

	return managedStacks, nil
}

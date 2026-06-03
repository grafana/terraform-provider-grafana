package generate

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/generate/postprocessing"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
)

type stack struct {
	name          string
	slug          string
	isCloud       bool
	url           string
	managementKey string
	smURL         string
	smToken       string

	onCallURL   string
	onCallToken string
}

func generateCloudResources(ctx context.Context, cfg *Config) ([]stack, GenerationResult) {
	// Gen provider
	providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
	providerBlock.Body().SetAttributeValue("alias", cty.StringVal("cloud"))
	providerBlock.Body().SetAttributeValue("cloud_access_policy_token", cty.StringVal(cfg.Cloud.AccessPolicyToken))
	if err := writeBlocks(filepath.Join(cfg.OutputDir, "cloud-provider.tf"), providerBlock); err != nil {
		return nil, failure(err)
	}

	// Generate imports
	config := provider.ProviderConfig{
		CloudAccessPolicyToken: types.StringValue(cfg.Cloud.AccessPolicyToken),
	}
	if err := config.SetDefaults(); err != nil {
		return nil, failure(err)
	}

	client, err := provider.CreateClients(config)
	if err != nil {
		return nil, failure(err)
	}
	cloudClient := client.GrafanaCloudAPI

	stacks, _, err := cloudClient.InstancesAPI.GetInstances(ctx).Execute()
	if err != nil {
		return nil, failure(err)
	}

	// Cleanup SAs
	managementServiceAccountName := cfg.Cloud.StackServiceAccountName

	if cfg.Cloud.CreateStackServiceAccount {
		for _, stack := range stacks.Items {
			if err := createManagementStackServiceAccount(ctx, cloudClient, stack, managementServiceAccountName); err != nil {
				return nil, failure(err)
			}
		}
	}

	data := cloud.NewListerData(cfg.Cloud.Org)
	returnResult := generateImportBlocks(ctx, client, data, cloud.Resources, cfg, "cloud")
	if returnResult.Blocks() == 0 { // Skip if no resources were found
		return nil, returnResult
	}

	plannedState, err := getPlannedState(ctx, cfg)
	if err != nil {
		return nil, failure(err)
	}
	if err := postprocessing.StripDefaults(filepath.Join(cfg.OutputDir, "cloud-resources.tf"), nil); err != nil {
		return nil, failure(err)
	}
	if err := postprocessing.ReplaceReferences(filepath.Join(cfg.OutputDir, "cloud-resources.tf"), plannedState, nil); err != nil {
		return nil, failure(err)
	}

	if !cfg.Cloud.CreateStackServiceAccount {
		return nil, returnResult
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
		smInstallationMetricsPublishBlock.Body().SetAttributeValue("name", cty.StringVal(smAccessPolicyName(stack)))
		smInstallationMetricsPublishBlock.Body().SetAttributeValue("scopes", cty.ListVal([]cty.Value{cty.StringVal("metrics:write"), cty.StringVal("stacks:read")}))
		smInstallationMetricsPublishRealmBlock := hclwrite.NewBlock("realm", nil)
		smInstallationMetricsPublishRealmBlock.Body().SetAttributeValue("type", cty.StringVal("stack"))
		smInstallationMetricsPublishRealmBlock.Body().SetAttributeTraversal("identifier", traversal("grafana_cloud_stack", stack.Slug, "id"))
		smInstallationMetricsPublishBlock.Body().AppendBlock(smInstallationMetricsPublishRealmBlock)

		smInstallationTokenBlock := hclwrite.NewBlock("resource", []string{"grafana_cloud_access_policy_token", policyResourceName})
		smInstallationTokenBlock.Body().SetAttributeTraversal("provider", traversal("grafana", "cloud"))
		smInstallationTokenBlock.Body().SetAttributeValue("region", cty.StringVal(stack.RegionSlug))
		smInstallationTokenBlock.Body().SetAttributeTraversal("access_policy_id", traversal("grafana_cloud_access_policy", policyResourceName, "policy_id"))
		smInstallationTokenBlock.Body().SetAttributeValue("name", cty.StringVal(smAccessPolicyName(stack)))

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

		if err := writeBlocks(filepath.Join(cfg.OutputDir, fmt.Sprintf("stack-%s-provider.tf", stack.Slug)), saBlock, saTokenBlock, smInstallationMetricsPublishBlock, smInstallationTokenBlock, smInstallationBlock, providerBlock); err != nil {
			return nil, failuref("failed to write management service account blocks for stack %q: %w", stack.Slug, err)
		}

		// Apply then go into the state and find the management key
		err := cfg.Terraform.Apply(ctx,
			tfexec.Target("grafana_cloud_stack_service_account."+stack.Slug),
			tfexec.Target("grafana_cloud_stack_service_account_token."+stack.Slug),
			tfexec.Target("grafana_cloud_access_policy."+policyResourceName),
			tfexec.Target("grafana_cloud_access_policy_token."+policyResourceName),
			tfexec.Target("grafana_synthetic_monitoring_installation."+stack.Slug),
		)
		if err != nil {
			return nil, failuref("failed to apply management service account blocks for stack %q: %w", stack.Slug, err)
		}
	}

	managedStacks := []stack{}
	state, err := getState(ctx, cfg)
	if err != nil {
		return nil, failure(err)
	}
	stacksMap := map[string]stack{}
	for _, resource := range state.Values.RootModule.Resources {
		if resource.Type == "grafana_cloud_stack_service_account_token" {
			slug := resource.AttributeValues["stack_slug"].(string)
			stack := stacksMap[slug]
			stack.isCloud = true
			stack.slug = slug
			stack.url = stacksBySlug[slug].Url
			stack.managementKey = resource.AttributeValues["key"].(string)
			stacksMap[slug] = stack
		}
		if resource.Type == "grafana_synthetic_monitoring_installation" {
			idStr := resource.AttributeValues["stack_id"].(string)
			slug := idStr
			if id, err := strconv.Atoi(idStr); err == nil {
				slug = stacksByID[id].Slug
			}
			stack := stacksMap[slug]
			stack.smToken = resource.AttributeValues["sm_access_token"].(string)
			stack.smURL = resource.AttributeValues["stack_sm_api_url"].(string)
			stacksMap[slug] = stack
		}
	}

	for _, stack := range stacksMap {
		managedStacks = append(managedStacks, stack)
	}

	return managedStacks, returnResult
}

func createManagementStackServiceAccount(ctx context.Context, cloudClient *gcom.APIClient, stack gcom.FormattedApiInstance, saName string) error {
	log.Printf("Waiting until %s is ready...\n", stack.Slug)
	if err := waitForSuccessfulGET(stack.Url, 2*time.Minute); err != nil {
		return err
	}

	tempClient, cleanup, err := cloud.CreateTemporaryStackGrafanaClient(ctx, cloudClient, stack.Slug, "temp-sa-")
	if err != nil {
		return fmt.Errorf("failed to create temporary client for stack %q: %w", stack.Slug, err)
	}

	serviceAccountsResp, err := tempClient.ServiceAccounts.SearchOrgServiceAccountsWithPaging(service_accounts.NewSearchOrgServiceAccountsWithPagingParams())
	if err != nil {
		return fmt.Errorf("failed to search service accounts for stack %q: %w", stack.Slug, err)
	}
	for _, sa := range serviceAccountsResp.Payload.ServiceAccounts {
		if sa.Name == saName {
			log.Printf("found existing management service account (%s) in stack %q\n", saName, stack.Slug)
			// Delete the SA to recreate it via TF
			_, err := tempClient.ServiceAccounts.DeleteServiceAccount(sa.ID)
			if err != nil {
				return fmt.Errorf("failed to delete existing management service account (%s) in stack %q: %w", saName, stack.Slug, err)
			}
			break
		}
	}

	if err := cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup temporary client for stack %q: %w", stack.Slug, err)
	}

	// Delete existing SM installation
	resp, _, err := cloudClient.AccesspoliciesAPI.GetAccessPolicies(ctx).OrgId(int32(stack.OrgId)).Region(stack.RegionSlug).Execute()
	if err != nil {
		return err
	}
	for _, policy := range resp.Items {
		if policy.Name == smAccessPolicyName(stack) {
			log.Printf("found existing SM installation (%s) in stack %q\n", smAccessPolicyName(stack), stack.Slug)
			_, err := cloudClient.AccesspoliciesAPI.DeleteAccessPolicy(ctx, *policy.Id).XRequestId("tf-gen").OrgId(int32(stack.OrgId)).Region(stack.RegionSlug).Execute()
			if err != nil {
				return fmt.Errorf("failed to delete existing SM installation (%s) in stack %q: %w", smAccessPolicyName(stack), stack.Slug, err)
			}
			break
		}
	}

	return nil
}

func waitForSuccessfulGET(url string, timeout time.Duration) error {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timed out waiting for %s to be ready", url)
		}

		// HTTP GET request to the stack URL
		// If it returns 200, break
		resp, err := http.Get(url) // nolint:gosec
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}

func smAccessPolicyName(stack gcom.FormattedApiInstance) string {
	return stack.Slug + "-sm-metrics-publish"
}

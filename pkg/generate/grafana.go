package generate

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/oncall"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/syntheticmonitoring"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/generate/postprocessing"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/provider"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
)

func generateGrafanaResources(ctx context.Context, cfg *Config, stack stack, genProvider bool) GenerationResult {
	generatedFilename := func(suffix string) string {
		if stack.name == "" {
			return filepath.Join(cfg.OutputDir, suffix)
		}

		return filepath.Join(cfg.OutputDir, stack.name+"-"+suffix)
	}

	if genProvider {
		providerBlock := hclwrite.NewBlock("provider", []string{"grafana"})
		providerBlock.Body().SetAttributeValue("url", cty.StringVal(stack.url))
		providerBlock.Body().SetAttributeValue("auth", cty.StringVal(stack.managementKey))
		if stack.smToken != "" && stack.smURL != "" {
			providerBlock.Body().SetAttributeValue("sm_url", cty.StringVal(stack.smURL))
			providerBlock.Body().SetAttributeValue("sm_access_token", cty.StringVal(stack.smToken))
		}
		if stack.onCallToken != "" && stack.onCallURL != "" {
			providerBlock.Body().SetAttributeValue("oncall_url", cty.StringVal(stack.onCallURL))
			providerBlock.Body().SetAttributeValue("oncall_access_token", cty.StringVal(stack.onCallToken))
		}
		if stack.name != "" {
			providerBlock.Body().SetAttributeValue("alias", cty.StringVal(stack.name))
		}
		if err := writeBlocks(generatedFilename("provider.tf"), providerBlock); err != nil {
			return failure(err)
		}
	}

	singleOrg := !strings.Contains(stack.managementKey, ":")
	listerData := grafana.NewListerData(singleOrg, true)

	// Generate resources
	config := provider.ProviderConfig{
		URL:  types.StringValue(stack.url),
		Auth: types.StringValue(stack.managementKey),
	}
	resources := grafana.Resources
	if stack.smToken != "" && stack.smURL != "" {
		resources = append(resources, syntheticmonitoring.Resources...)
		config.SMURL = types.StringValue(stack.smURL)
		config.SMAccessToken = types.StringValue(stack.smToken)
	}
	if stack.onCallToken != "" && stack.onCallURL != "" {
		resources = append(resources, oncall.Resources...)
		config.OncallAccessToken = types.StringValue(stack.onCallToken)
		config.OncallURL = types.StringValue(stack.onCallURL)
	}
	if err := config.SetDefaults(); err != nil {
		return failure(err)
	}

	client, err := provider.CreateClients(config)
	if err != nil {
		return failure(err)
	}

	if stack.isCloud {
		resources = append(resources, slo.Resources...)
		resources = append(resources, machinelearning.Resources...)
	}

	returnResult := generateImportBlocks(ctx, client, listerData, resources, cfg, stack.name)
	if returnResult.Blocks() == 0 { // Skip if no resources were found
		return returnResult
	}

	stripDefaultsExtraFields := map[string]any{}
	if singleOrg {
		stripDefaultsExtraFields["org_id"] = true // Always remove org_id if single org
	} else {
		stripDefaultsExtraFields["org_id"] = `"1"` // Remove org_id if it's the default
	}
	stripDefaultsExtraFields["missing_series_evals_to_resolve"] = "0" // Remove missing_series_evals_to_resolve if it's the default

	plannedState, err := getPlannedState(ctx, cfg)
	if err != nil {
		return failure(err)
	}
	if err := postprocessing.StripDefaults(generatedFilename("resources.tf"), stripDefaultsExtraFields); err != nil {
		return failure(err)
	}
	if err := postprocessing.ExtractDashboards(generatedFilename("resources.tf"), plannedState); err != nil {
		return failure(err)
	}
	if err := postprocessing.ReplaceReferences(generatedFilename("resources.tf"), plannedState, []string{
		"*.org_id=grafana_organization.id",
	}); err != nil {
		return failure(err)
	}

	return returnResult
}

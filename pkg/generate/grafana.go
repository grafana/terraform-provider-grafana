package generate

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/syntheticmonitoring"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/provider"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
)

func generateGrafanaResources(ctx context.Context, cfg *Config, stack stack, genProvider bool) error {
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
		if stack.name != "" {
			providerBlock.Body().SetAttributeValue("alias", cty.StringVal(stack.name))
		}
		if err := writeBlocks(generatedFilename("provider.tf"), providerBlock); err != nil {
			return err
		}
	}

	singleOrg := !strings.Contains(stack.managementKey, ":")
	listerData := grafana.NewListerData(singleOrg)

	// Generate resources
	config := provider.ProviderConfig{
		URL:  types.StringValue(stack.url),
		Auth: types.StringValue(stack.managementKey),
	}
	if stack.smToken != "" {
		config.SMAccessToken = types.StringValue(stack.smToken)
	}
	if stack.smURL != "" {
		config.SMURL = types.StringValue(stack.smURL)
	}
	if err := config.SetDefaults(); err != nil {
		return err
	}

	client, err := provider.CreateClients(config)
	if err != nil {
		return err
	}

	resources := grafana.Resources
	if strings.HasPrefix(stack.name, "stack-") { // TODO: is cloud. Find a better way to detect this
		resources = append(resources, slo.Resources...)
		resources = append(resources, machinelearning.Resources...)
		resources = append(resources, syntheticmonitoring.Resources...)
	}
	if err := generateImportBlocks(ctx, client, listerData, resources, cfg, stack.name); err != nil {
		return err
	}

	stripDefaultsExtraFields := map[string]any{}
	if singleOrg {
		stripDefaultsExtraFields["org_id"] = true // Always remove org_id if single org
	} else {
		stripDefaultsExtraFields["org_id"] = `"1"` // Remove org_id if it's the default
	}

	if err := stripDefaults(generatedFilename("resources.tf"), stripDefaultsExtraFields); err != nil {
		return err
	}
	if err := abstractDashboards(generatedFilename("resources.tf")); err != nil {
		return err
	}
	if err := wrapJSONFieldsInFunction(generatedFilename("resources.tf")); err != nil {
		return err
	}

	return nil
}

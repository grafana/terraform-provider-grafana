// Package agento11y implements Terraform resources for the Grafana Agent
// Observability (Sigil) evaluation control plane, reached through the
// grafana-agento11y-app plugin backend proxy.
package agento11y

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
)

// Resources are the Terraform resources exposed by this package.
var Resources = []*common.Resource{
	makeResourceEvaluator().WithLister(listEvaluatorIDs),
	makeResourceEvaluationRule().WithLister(listRuleIDs),
	makeResourceRuleAction(),
	makeResourceHookRule().WithLister(listHookRuleIDs),
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*agento11yapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil, fmt.Errorf("unexpected Resource Configure Type: %T", req.ProviderData)
	}

	if client.Agento11yAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana Agent Observability API.",
			"Please ensure that url and auth are set in the provider configuration and the Grafana Agent Observability (grafana-agento11y-app) plugin is installed.",
		)
		return nil, errors.New("Agento11yAPIClient is nil")
	}

	return client.Agento11yAPIClient, nil
}

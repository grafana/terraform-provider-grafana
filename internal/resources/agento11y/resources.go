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

// writePermissionDescription closes every resource description in this package.
// All five resources write through the same plugin route and the same access
// check, so they state the same requirement.
const writePermissionDescription = "Writes require a user or service account with the `grafana-agento11y-app.eval:write` permission, which only the Admin basic role grants by default."

// Resources are the Terraform resources exposed by this package.
var Resources = []*common.Resource{
	makeResourceCollection().WithLister(listCollectionIDs),
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

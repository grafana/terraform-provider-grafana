package k6

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/k6providerapi"
)

type basePluginFrameworkResource struct {
	client *k6.APIClient
	config *k6providerapi.K6APIConfig
}

func (r *basePluginFrameworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client.K6APIClient == nil || client.K6APIConfig == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing the configuration for the k6 Cloud API.",
			"Please ensure that k6_access_token and k6_stack_id are set in the provider configuration.",
		)

		return
	}

	r.client = client.K6APIClient
	r.config = client.K6APIConfig
}

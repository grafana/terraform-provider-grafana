package slo

import (
	"context"
	"fmt"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// configureSLOClient is a helper to configure the SLO client from provider data
func configureSLOClient(providerData any, currentClient *slo.APIClient) (*slo.APIClient, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Already configured
	if providerData == nil || currentClient != nil {
		return currentClient, diags
	}

	client, ok := providerData.(*common.Client)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil, diags
	}

	if client.SLOClient == nil {
		diags.AddError(
			"The Grafana Provider is missing a configuration for the SLO API.",
			"Please ensure that the SLO API client is configured in the provider.",
		)
		return nil, diags
	}

	return client.SLOClient, diags
}

// basePluginFrameworkDataSource is the base struct for SLO framework data sources
type basePluginFrameworkDataSource struct {
	client *slo.APIClient
}

// Configure is called by the framework to configure the data source with provider data
func (r *basePluginFrameworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := configureSLOClient(req.ProviderData, r.client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// basePluginFrameworkResource is the base struct for SLO framework resources
type basePluginFrameworkResource struct {
	client *slo.APIClient
}

// Configure is called by the framework to configure the resource with provider data
func (r *basePluginFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := configureSLOClient(req.ProviderData, r.client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

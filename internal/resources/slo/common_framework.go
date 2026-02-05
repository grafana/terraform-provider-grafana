package slo

import (
	"context"
	"fmt"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// basePluginFrameworkDataSource is the base struct for SLO framework data sources
type basePluginFrameworkDataSource struct {
	client *slo.APIClient
}

// Configure is called by the framework to configure the data source with provider data
func (r *basePluginFrameworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available)
	// We only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if client.SLOClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the SLO API.",
			"Please ensure that the SLO API client is configured in the provider.",
		)
		return
	}

	r.client = client.SLOClient
}

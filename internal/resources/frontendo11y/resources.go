package frontendo11y

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
)

var DataSources = []*common.DataSource{
	makeFrontendO11yAppDataSource(),
}

var Resources = []*common.Resource{
	makeResourceFrontendO11yApp(),
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*frontendo11yapi.Client, *gcom.APIClient, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.FrontendO11yAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Frontend Observability API.",
			"Please ensure that frontend_o11y_api_access_token are set in the provider configuration.",
		)

		return nil, nil, fmt.Errorf("frontendo11yapi is nil")
	}

	if client.GrafanaCloudAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana Cloud API.",
			"Please ensure that cloud_access_policy_token are set in the provider configuration.",
		)

		return nil, nil, fmt.Errorf("gcomapi is nil")
	}

	return client.FrontendO11yAPIClient, client.GrafanaCloudAPI, nil
}

func withClientForDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) (*frontendo11yapi.Client, *gcom.APIClient, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, nil, fmt.Errorf("unexpected DataSource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.FrontendO11yAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Frontend Observability API.",
			"Please ensure that cloud_access_policy_token are set in the provider configuration.",
		)

		return nil, nil, fmt.Errorf("gcomapi is nil")
	}

	return client.FrontendO11yAPIClient, client.GrafanaCloudAPI, nil
}

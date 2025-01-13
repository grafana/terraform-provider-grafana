package cloudprovider

import (
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var DataSources = []*common.DataSource{
	makeDataSourceAWSAccount(),
	makeDatasourceAWSCloudWatchScrapeJob(),
	makeDatasourceAWSCloudWatchScrapeJobs(),
	makeDataSourceAzureCredential(),
}

var Resources = []*common.Resource{
	makeResourceAWSAccount(),
	makeResourceAWSCloudWatchScrapeJob(),
	makeResourceAzureCredential(),
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*cloudproviderapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.CloudProviderAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Cloud Provider API.",
			"Please ensure that cloud_provider_url and cloud_provider_access_token are set in the provider configuration.",
		)

		return nil, fmt.Errorf("CloudProviderAPI is nil")
	}

	return client.CloudProviderAPI, nil
}

func withClientForDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) (*cloudproviderapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected DataSource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.CloudProviderAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Cloud Provider API.",
			"Please ensure that cloud_provider_url and cloud_provider_access_token are set in the provider configuration.",
		)

		return nil, fmt.Errorf("CloudProviderAPI is nil")
	}

	return client.CloudProviderAPI, nil
}

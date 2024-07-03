package cloudprovider

import (
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*cloudproviderapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *common.Client", req.ProviderData)
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

	return client.CloudProviderAPI, nil
}

var DataSources = []*common.DataSource{
	makeDataSourceAWSAccount(),
	makeDatasourceAWSCloudWatchScrapeJob(),
	makeDatasourceAWSCloudWatchScrapeJobs(),
}

var Resources = []*common.Resource{
	makeResourceAWSAccount(),
	makeResourceAWSCloudWatchScrapeJob(),
}

package connections

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/connectionsapi"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
)

var DataSources = []*common.DataSource{
	makeDatasourceMetricsEndpointScrapeJob(),
}

var Resources = []*common.Resource{
	makeResourceMetricsEndpointScrapeJob(),
}

type HTTPSURLValidator struct{}

func (v HTTPSURLValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v HTTPSURLValidator) MarkdownDescription(_ context.Context) string {
	return "value must be valid URL with HTTPS"
}

func (v HTTPSURLValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()

	if value == "" {
		response.Diagnostics.AddAttributeError(
			request.Path,
			v.Description(ctx),
			"A valid URL is required.\n\n"+
				fmt.Sprintf("Given Value: %q\n", value),
		)
		return
	}

	u, err := url.Parse(value)
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			v.Description(ctx),
			"A string value was provided that is not a valid URL.\n\n"+
				"Given Value: "+value+"\n"+
				"Error: "+err.Error(),
		)
		return
	}

	if u.Scheme != "https" {
		response.Diagnostics.AddAttributeError(
			request.Path,
			v.Description(ctx),
			"A URL was provided, protocol must be HTTPS.\n\n"+
				fmt.Sprintf("Given Value: %q\n", value),
		)
		return
	}
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*connectionsapi.Client, error) {
	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *client.Client", req.ProviderData)
	}

	if client.ConnectionsAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Connections API.",
			"Please ensure that connections_api_url and connections_api_access_token are set in the provider configuration.",
		)

		return nil, fmt.Errorf("ConnectionsAPI is nil")
	}

	return client.ConnectionsAPIClient, nil
}

func withClientForDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) (*connectionsapi.Client, error) {
	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected DataSource Configure Type: %T, expected *client.Client", req.ProviderData)
	}

	if client.ConnectionsAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Connections API.",
			"Please ensure that connections_api_url and connections_access_token are set in the provider configuration.",
		)

		return nil, fmt.Errorf("ConnectionsAPI is nil")
	}

	return client.ConnectionsAPIClient, nil
}

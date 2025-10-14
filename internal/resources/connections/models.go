package connections

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/connectionsapi"
)

type metricsEndpointScrapeJobTFModel struct {
	ID                          types.String `tfsdk:"id"`
	StackID                     types.String `tfsdk:"stack_id"`
	Name                        types.String `tfsdk:"name"`
	Enabled                     types.Bool   `tfsdk:"enabled"`
	AuthenticationMethod        types.String `tfsdk:"authentication_method"`
	AuthenticationBearerToken   types.String `tfsdk:"authentication_bearer_token"`
	AuthenticationBasicUsername types.String `tfsdk:"authentication_basic_username"`
	AuthenticationBasicPassword types.String `tfsdk:"authentication_basic_password"`
	URL                         types.String `tfsdk:"url"`
	ScrapeIntervalSeconds       types.Int64  `tfsdk:"scrape_interval_seconds"`
}

// convertJobTFModelToClientModel converts a metricsEndpointScrapeJobTFModel instance to a connectionsapi.MetricsEndpointScrapeJob instance.
// A special converter is needed because the TFModel uses special Terraform types that build upon their underlying Go types for
// supporting Terraform's state management/dependency analysis of the resource and its data.
func convertJobTFModelToClientModel(tfData metricsEndpointScrapeJobTFModel) connectionsapi.MetricsEndpointScrapeJob {
	return connectionsapi.MetricsEndpointScrapeJob{
		Enabled:                     tfData.Enabled.ValueBool(),
		AuthenticationMethod:        tfData.AuthenticationMethod.ValueString(),
		AuthenticationBearerToken:   tfData.AuthenticationBearerToken.ValueString(),
		AuthenticationBasicUsername: tfData.AuthenticationBasicUsername.ValueString(),
		AuthenticationBasicPassword: tfData.AuthenticationBasicPassword.ValueString(),
		URL:                         tfData.URL.ValueString(),
		ScrapeIntervalSeconds:       tfData.ScrapeIntervalSeconds.ValueInt64(),
	}
}

// convertClientModelToTFModel converts a connectionsapi.MetricsEndpointScrapeJob instance to a metricsEndpointScrapeJobTFModel instance.
// A special converter is needed because the TFModel uses special Terraform types that build upon their underlying Go types for
// supporting Terraform's state management/dependency analysis of the resource and its data.
func convertClientModelToTFModel(stackID, jobName string, scrapeJobData connectionsapi.MetricsEndpointScrapeJob) metricsEndpointScrapeJobTFModel {
	resp := metricsEndpointScrapeJobTFModel{
		ID:                    types.StringValue(resourceMetricsEndpointScrapeJobTerraformID.Make(stackID, jobName)),
		StackID:               types.StringValue(stackID),
		Name:                  types.StringValue(jobName),
		Enabled:               types.BoolValue(scrapeJobData.Enabled),
		AuthenticationMethod:  types.StringValue(scrapeJobData.AuthenticationMethod),
		URL:                   types.StringValue(scrapeJobData.URL),
		ScrapeIntervalSeconds: types.Int64Value(scrapeJobData.ScrapeIntervalSeconds),
	}

	resp.fillOptionalFieldsIfNotEmpty(scrapeJobData)

	return resp
}

func (m *metricsEndpointScrapeJobTFModel) fillOptionalFieldsIfNotEmpty(scrapeJobData connectionsapi.MetricsEndpointScrapeJob) {
	if scrapeJobData.AuthenticationBearerToken != "" {
		m.AuthenticationBearerToken = types.StringValue(scrapeJobData.AuthenticationBearerToken)
	}
	if scrapeJobData.AuthenticationBasicUsername != "" {
		m.AuthenticationBasicUsername = types.StringValue(scrapeJobData.AuthenticationBasicUsername)
	}
	if scrapeJobData.AuthenticationBasicPassword != "" {
		m.AuthenticationBasicPassword = types.StringValue(scrapeJobData.AuthenticationBasicPassword)
	}
}

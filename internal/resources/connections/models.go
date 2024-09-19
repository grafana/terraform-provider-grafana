package connections

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
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

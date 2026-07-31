package oncall

import (
	"context"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// All on-call resources have a single string ID format
var resourceID = common.NewResourceID(common.StringIDField("id"))

type basePluginFrameworkResource struct {
	client *onCallAPI.Client
}

func (r *basePluginFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if client.OnCallClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the OnCall API.",
			"Please ensure that the provider is configured with `url` and `auth` (a Grafana service account token). Alternatively, the deprecated `oncall_url` (and optionally `oncall_access_token`) attributes can be set.",
		)

		return
	}

	if err := client.OnCallClient.EnsureBaseURL(ctx); err != nil {
		resp.Diagnostics.AddError("Failed to configure the Grafana OnCall client", err.Error())
		return
	}
	for _, warning := range client.OnCallClient.Warnings() {
		resp.Diagnostics.AddWarning("Grafana OnCall configuration", warning)
	}

	r.client = client.OnCallClient
}

type basePluginFrameworkDataSource struct {
	client *onCallAPI.Client
}

func (r *basePluginFrameworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	if client.OnCallClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the OnCall API.",
			"Please ensure that the provider is configured with `url` and `auth` (a Grafana service account token). Alternatively, the deprecated `oncall_url` (and optionally `oncall_access_token`) attributes can be set.",
		)

		return
	}

	if err := client.OnCallClient.EnsureBaseURL(ctx); err != nil {
		resp.Diagnostics.AddError("Failed to configure the Grafana OnCall client", err.Error())
		return
	}
	for _, warning := range client.OnCallClient.Warnings() {
		resp.Diagnostics.AddWarning("Grafana OnCall configuration", warning)
	}

	r.client = client.OnCallClient
}

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		client := meta.(*common.Client).OnCallClient
		if client == nil {
			return diag.Errorf("the OnCall client is required for this resource. Configure the provider with `url` and `auth` (a Grafana service account token), or set the deprecated `oncall_url`/`oncall_access_token` attributes")
		}
		if err := client.EnsureBaseURL(ctx); err != nil {
			return diag.Errorf("failed to configure the OnCall client: %s", err)
		}
		diags := f(ctx, d, client)
		for _, warning := range client.Warnings() {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Grafana OnCall configuration",
				Detail:   warning,
			})
		}
		return diags
	}
}

var DataSources = []*common.DataSource{
	dataSourceEscalationChain(),
	dataSourceSchedule(),
	dataSourceSlackChannel(),
	dataSourceOutgoingWebhook(),
	dataSourceUserGroup(),
	dataSourceTeam(),
	dataSourceIntegration(),
	dataSourceUser(),
	dataSourceUsers(),
	dataSourceLabel(),
}

var Resources = []*common.Resource{
	resourceIntegration(),
	resourceRoute(),
	resourceEscalationChain(),
	resourceEscalation(),
	resourceOnCallShift(),
	resourceSchedule(),
	resourceOutgoingWebhook(),
	resourceUserNotificationRule(),
}

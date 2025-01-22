package cloud

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ClientRequestID() string {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return ""
	}
	return "tf-" + uuid
}

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).GrafanaCloudAPI
		if client == nil {
			return diag.Errorf("the Cloud API client is required for this resource. Set the cloud_access_policy_token provider attribute")
		}
		return f(ctx, d, client)
	}
}

func apiError(err error) diag.Diagnostics {
	if err == nil {
		return nil
	}
	detail := err.Error()
	if err, ok := err.(*gcom.GenericOpenAPIError); ok {
		detail += "\n" + string(err.Body())
	}
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
			Detail:   detail,
		},
	}
}

type basePluginFrameworkDataSource struct {
	client *gcom.APIClient
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

	if client.GrafanaCloudAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana Cloud API.",
			"Please ensure that cloud_api_url and cloud_access_policy_token are set in the provider configuration.",
		)

		return
	}

	r.client = client.GrafanaCloudAPI
}

type basePluginFrameworkResource struct {
	client *gcom.APIClient
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

	r.client = client.GrafanaCloudAPI
}

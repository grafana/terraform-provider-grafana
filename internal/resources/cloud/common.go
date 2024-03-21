package cloud

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/go-uuid"
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

type basePluginFrameworkResource struct {
	client *gcom.APIClient
}

func (r *basePluginFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Cloud API client",
			"the Cloud API client is required for this resource. Set the cloud_access_policy_token provider attribute",
		)

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

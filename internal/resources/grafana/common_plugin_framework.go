package grafana

import (
	"context"
	"fmt"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type basePluginFrameworkDataSource struct {
	client *goapi.GrafanaHTTPAPI
	config *goapi.TransportConfig
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

	if client.GrafanaAPI == nil || client.GrafanaAPIConfig == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana API.",
			"Please ensure that url and auth are set in the provider configuration.",
		)

		return
	}

	r.client = client.GrafanaAPI
	r.config = client.GrafanaAPIConfig
}

// clientFromNewOrgResource creates an OpenAPI client from the `org_id` attribute of a resource
// This client is meant to be used in `Create` functions when the ID hasn't already been baked into the resource ID
func (r *basePluginFrameworkDataSource) clientFromNewOrgResource(orgIDStr string) (*goapi.GrafanaHTTPAPI, int64, error) {
	if r.client == nil {
		return nil, 0, fmt.Errorf("client not configured")
	}

	client := r.client.Clone()
	orgID, _ := strconv.ParseInt(orgIDStr, 10, 64)
	if orgID == 0 {
		orgID = client.OrgID()
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, nil
}

type basePluginFrameworkResource struct {
	client *goapi.GrafanaHTTPAPI
	config *goapi.TransportConfig
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

	if client.GrafanaAPI == nil || client.GrafanaAPIConfig == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana API.",
			"Please ensure that url and auth are set in the provider configuration.",
		)

		return
	}

	r.client = client.GrafanaAPI
	r.config = client.GrafanaAPIConfig
}

// clientFromExistingOrgResource creates a client from the ID of an org-scoped resource
// Those IDs are in the <orgID>:<resourceID> format
func (r *basePluginFrameworkResource) clientFromExistingOrgResource(idFormat *common.ResourceID, id string) (*goapi.GrafanaHTTPAPI, int64, []any, error) {
	if r.client == nil {
		return nil, 0, nil, fmt.Errorf("client not configured")
	}

	client := r.client.Clone()
	split, err := idFormat.Split(id)
	if err != nil {
		return nil, 0, nil, err
	}
	var orgID int64
	if len(split) < len(idFormat.Fields()) {
		orgID = client.OrgID()
	} else {
		orgID = split[0].(int64)
		split = split[1:]
		client = client.WithOrgID(orgID)
	}
	return client, orgID, split, nil
}

// clientFromNewOrgResource creates an OpenAPI client from the `org_id` attribute of a resource
// This client is meant to be used in `Create` functions when the ID hasn't already been baked into the resource ID
func (r *basePluginFrameworkResource) clientFromNewOrgResource(orgIDStr string) (*goapi.GrafanaHTTPAPI, int64, error) {
	if r.client == nil {
		return nil, 0, fmt.Errorf("client not configured")
	}

	client := r.client.Clone()
	orgID, _ := strconv.ParseInt(orgIDStr, 10, 64)
	if orgID == 0 {
		orgID = client.OrgID()
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, nil
}

// To be used in non-org-scoped resources
// func (r *basePluginFrameworkResource) globalClient() (*goapi.GrafanaHTTPAPI, error) {
// if r.client == nil {
// 	return nil, 0, nil, fmt.Errorf("client not configured")
// }

// 	client := r.client.Clone().WithOrgID(0)
// 	if r.config.APIKey != "" {
// 		return client, fmt.Errorf("global scope resources cannot be managed with an API key. Use basic auth instead")
// 	}
// 	return client, nil
// }

func pluginFrameworkOrgIDAttribute() frameworkSchema.Attribute {
	return frameworkSchema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "The Organization ID. If not set, the default organization is used for basic authentication, or the one that owns your service account for token authentication.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
			&orgIDAttributePlanModifier{},
		},
	}
}

type orgIDAttributePlanModifier struct{}

func (d *orgIDAttributePlanModifier) Description(ctx context.Context) string {
	return "Ignores the org_id attribute when it is empty, and uses the provider's org_id instead."
}

func (d *orgIDAttributePlanModifier) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (d *orgIDAttributePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var orgID types.String
	diags := req.Plan.GetAttribute(ctx, path.Root("org_id"), &orgID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the org_id is empty, we want to use the provider's org_id
	// We don't want to show any diff
	if (orgID.IsNull() || orgID.ValueString() == "") && !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

type orgScopedAttributePlanModifier struct{}

func (d *orgScopedAttributePlanModifier) Description(ctx context.Context) string {
	return "Ignores the orgID part of a resource ID."
}

func (d *orgScopedAttributePlanModifier) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (d *orgScopedAttributePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Equality should ignore the org ID
	_, first := SplitOrgResourceID(req.StateValue.ValueString())
	_, second := SplitOrgResourceID(resp.PlanValue.ValueString())

	if first != "" && first == second {
		resp.PlanValue = req.StateValue
	}
}

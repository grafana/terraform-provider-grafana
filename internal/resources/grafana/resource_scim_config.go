package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SCIMConfig represents the SCIM configuration structure
type SCIMConfig struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   SCIMConfigMetadata `json:"metadata"`
	Spec       SCIMConfigSpec     `json:"spec"`
}

// SCIMConfigMetadata represents the metadata for SCIM config
type SCIMConfigMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// SCIMConfigSpec represents the SCIM configuration specification
type SCIMConfigSpec struct {
	EnableUserSync            bool `json:"enableUserSync"`
	EnableGroupSync           bool `json:"enableGroupSync"`
	RejectNonProvisionedUsers bool `json:"rejectNonProvisionedUsers"`
}

var (
	_ resource.Resource                = &scimConfigResource{}
	_ resource.ResourceWithConfigure   = &scimConfigResource{}
	_ resource.ResourceWithImportState = &scimConfigResource{}

	resourceSCIMConfigName = "grafana_scim_config"
	resourceSCIMConfigID   = common.NewResourceID(common.OptionalIntIDField("orgID"))
)

func makeResourceSCIMConfig() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceSCIMConfigName,
		resourceSCIMConfigID,
		&scimConfigResource{},
	)
}

type resourceSCIMConfigModel struct {
	ID                        types.String `tfsdk:"id"`
	OrgID                     types.String `tfsdk:"org_id"`
	EnableUserSync            types.Bool   `tfsdk:"enable_user_sync"`
	EnableGroupSync           types.Bool   `tfsdk:"enable_group_sync"`
	RejectNonProvisionedUsers types.Bool   `tfsdk:"reject_non_provisioned_users"`
}

type scimConfigResource struct {
	basePluginFrameworkResource
}

func (r *scimConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceSCIMConfigName
}

func (r *scimConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
**Note:** Available in [Grafana Enterprise](https://grafana.com/docs/grafana/latest/introduction/grafana-enterprise/) and [Grafana Cloud](https://grafana.com/docs/grafana-cloud/).

* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-access/configure-scim-provisioning/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"enable_user_sync": schema.BoolAttribute{
				Required:    true,
				Description: "Whether user synchronization is enabled.",
			},
			"enable_group_sync": schema.BoolAttribute{
				Required:    true,
				Description: "Whether group synchronization is enabled.",
			},
			"reject_non_provisioned_users": schema.BoolAttribute{
				Required:    true,
				Description: "Whether to block non-provisioned user access to Grafana. Cloud Portal users will always be able to access Grafana, regardless of this setting.",
			},
		},
	}
}

func (r *scimConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics = diags
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *scimConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceSCIMConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, "scim-config"))

	diags := r.createOrUpdate(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *scimConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceSCIMConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *scimConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceSCIMConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the ID from state
	var stateData resourceSCIMConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = stateData.ID

	diags := r.createOrUpdate(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *scimConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceSCIMConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, diags := r.namespace()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transportConfig := r.commonClient.GrafanaAPIConfig
	if transportConfig == nil {
		resp.Diagnostics.AddError("Transport configuration not available", "transport configuration not available")
		return
	}

	baseURL := fmt.Sprintf("%s://%s", transportConfig.Schemes[0], transportConfig.Host)
	apiPath, err := url.JoinPath("apis/scim.grafana.app/v0alpha1/namespaces", namespace, "config/default")
	if err != nil {
		resp.Diagnostics.AddError("Failed to construct API path", err.Error())
		return
	}
	requestURL := fmt.Sprintf("%s/%s", baseURL, apiPath)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", requestURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create request", err.Error())
		return
	}

	setAuthHeaders(httpReq, transportConfig)

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete SCIM config", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNotFound {
		resp.Diagnostics.AddError("Failed to delete SCIM config", fmt.Sprintf("unexpected status: %d", httpResp.StatusCode))
	}
}

// namespace determines the API namespace based on whether this is a cloud or on-prem instance.
func (r *scimConfigResource) namespace() (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	switch {
	case r.commonClient.GrafanaStackID > 0:
		return fmt.Sprintf("stacks-%d", r.commonClient.GrafanaStackID), diags
	case r.commonClient.GrafanaOrgID > 0:
		return "default", diags
	default:
		diags.AddError(
			"Cannot determine namespace",
			"expected either Grafana org ID (for local Grafana) or Grafana stack ID (for Grafana Cloud) to be set",
		)
		return "", diags
	}
}

// createOrUpdate sends a PUT request to create or update the SCIM config.
func (r *scimConfigResource) createOrUpdate(ctx context.Context, data *resourceSCIMConfigModel) diag.Diagnostics {
	var diags diag.Diagnostics

	namespace, nsDiags := r.namespace()
	diags.Append(nsDiags...)
	if diags.HasError() {
		return diags
	}

	transportConfig := r.commonClient.GrafanaAPIConfig
	if transportConfig == nil {
		diags.AddError("Transport configuration not available", "transport configuration not available")
		return diags
	}

	scimConfig := SCIMConfig{
		APIVersion: "scim.grafana.app/v0alpha1",
		Kind:       "SCIMConfig",
		Metadata: SCIMConfigMetadata{
			Name:      "default",
			Namespace: namespace,
		},
		Spec: SCIMConfigSpec{
			EnableUserSync:            data.EnableUserSync.ValueBool(),
			EnableGroupSync:           data.EnableGroupSync.ValueBool(),
			RejectNonProvisionedUsers: data.RejectNonProvisionedUsers.ValueBool(),
		},
	}

	jsonData, err := json.Marshal(scimConfig)
	if err != nil {
		diags.AddError("Failed to marshal SCIM config", err.Error())
		return diags
	}

	baseURL := fmt.Sprintf("%s://%s", transportConfig.Schemes[0], transportConfig.Host)
	apiPath, err := url.JoinPath("apis/scim.grafana.app/v0alpha1/namespaces", namespace, "config/default")
	if err != nil {
		diags.AddError("Failed to construct API path", err.Error())
		return diags
	}
	requestURL := fmt.Sprintf("%s/%s", baseURL, apiPath)

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		diags.AddError("Failed to create request", err.Error())
		return diags
	}

	httpReq.Header.Set("Content-Type", "application/json")
	setAuthHeaders(httpReq, transportConfig)

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		diags.AddError("Failed to create or update SCIM config", err.Error())
		return diags
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		diags.AddError("Failed to create or update SCIM config", fmt.Sprintf("unexpected status: %d", httpResp.StatusCode))
	}

	return diags
}

// read fetches the SCIM config from the API and returns a populated model.
func (r *scimConfigResource) read(ctx context.Context, id string) (*resourceSCIMConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	namespace, nsDiags := r.namespace()
	diags.Append(nsDiags...)
	if diags.HasError() {
		return nil, diags
	}

	transportConfig := r.commonClient.GrafanaAPIConfig
	if transportConfig == nil {
		diags.AddError("Transport configuration not available", "transport configuration not available")
		return nil, diags
	}

	baseURL := fmt.Sprintf("%s://%s", transportConfig.Schemes[0], transportConfig.Host)
	apiPath, err := url.JoinPath("apis/scim.grafana.app/v0alpha1/namespaces", namespace, "config/default")
	if err != nil {
		diags.AddError("Failed to construct API path", err.Error())
		return nil, diags
	}
	requestURL := fmt.Sprintf("%s/%s", baseURL, apiPath)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		diags.AddError("Failed to create request", err.Error())
		return nil, diags
	}

	setAuthHeaders(httpReq, transportConfig)

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		diags.AddError("Failed to read SCIM config", err.Error())
		return nil, diags
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		return nil, diags
	}

	if httpResp.StatusCode != http.StatusOK {
		diags.AddError("Failed to read SCIM config", fmt.Sprintf("unexpected status: %d", httpResp.StatusCode))
		return nil, diags
	}

	var scimConfig SCIMConfig
	if err := json.NewDecoder(httpResp.Body).Decode(&scimConfig); err != nil {
		diags.AddError("Failed to decode SCIM config", err.Error())
		return nil, diags
	}

	// Determine orgID from the stored ID, or fall back to the client org ID.
	orgID, _ := SplitOrgResourceID(id)
	if orgID == 0 {
		orgID = r.commonClient.GrafanaOrgID
	}

	data := &resourceSCIMConfigModel{
		ID:                        types.StringValue(MakeOrgResourceID(orgID, "scim-config")),
		OrgID:                     types.StringValue(strconv.FormatInt(orgID, 10)),
		EnableUserSync:            types.BoolValue(scimConfig.Spec.EnableUserSync),
		EnableGroupSync:           types.BoolValue(scimConfig.Spec.EnableGroupSync),
		RejectNonProvisionedUsers: types.BoolValue(scimConfig.Spec.RejectNonProvisionedUsers),
	}

	return data, diags
}

// setAuthHeaders applies the appropriate authentication headers to the request.
func setAuthHeaders(req *http.Request, transportConfig *goapi.TransportConfig) {
	if transportConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+transportConfig.APIKey)
	} else if transportConfig.BasicAuth != nil {
		username := transportConfig.BasicAuth.Username()
		password, _ := transportConfig.BasicAuth.Password()
		req.SetBasicAuth(username, password)
	}
}

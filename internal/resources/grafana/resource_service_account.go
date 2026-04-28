package grafana

import (
	"context"
	"strconv"
	"sync"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

var (
	_ resource.Resource                = &serviceAccountResource{}
	_ resource.ResourceWithConfigure   = &serviceAccountResource{}
	_ resource.ResourceWithImportState = &serviceAccountResource{}

	resourceServiceAccountName = "grafana_service_account"
	resourceServiceAccountID   = orgResourceIDInt("id")
)

// Service accounts have issues with concurrent creation, so we lock creation.
var serviceAccountCreateMutex sync.Mutex

func resourceServiceAccount() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceServiceAccountName,
		resourceServiceAccountID,
		&serviceAccountResource{},
	).
		WithLister(listerFunctionOrgResource(listServiceAccounts)).
		WithPreferredResourceNameField("name")
}

func listServiceAccounts(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	var page int64 = 1
	for {
		params := service_accounts.NewSearchOrgServiceAccountsWithPagingParams().WithPage(&page)
		resp, err := client.ServiceAccounts.SearchOrgServiceAccountsWithPaging(params)
		if err != nil {
			return nil, err
		}

		for _, sa := range resp.Payload.ServiceAccounts {
			ids = append(ids, MakeOrgResourceID(orgID, strconv.FormatInt(sa.ID, 10)))
		}

		if resp.Payload.TotalCount <= int64(len(ids)) {
			break
		}

		page++
	}

	return ids, nil
}

// retrieveServiceAccount fetches a service account by ID. Used by the Framework resource Read and by the datasource.
func retrieveServiceAccount(client *goapi.GrafanaHTTPAPI, id int64) (*models.ServiceAccountDTO, error) {
	resp, err := client.ServiceAccounts.RetrieveServiceAccount(id)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

type serviceAccountResourceModel struct {
	ID         types.String `tfsdk:"id"`
	OrgID      types.String `tfsdk:"org_id"`
	Name       types.String `tfsdk:"name"`
	Role       types.String `tfsdk:"role"`
	IsDisabled types.Bool   `tfsdk:"is_disabled"`
}

type serviceAccountResource struct {
	basePluginFrameworkResource
}

func (r *serviceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceServiceAccountName
}

func (r *serviceAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
**Note:** This resource is available only with Grafana 9.1+.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/serviceaccount/#service-account-api)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Organization ID. If not set, the Org ID defined in the provider block will be used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					&orgIDAttributePlanModifier{},
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service account.",
			},
			"role": schema.StringAttribute{
				Required:    true,
				Description: "The basic role of the service account in the organization.",
				Validators: []validator.String{
					stringvalidator.OneOf("Viewer", "Editor", "Admin", "None"),
				},
			},
			"is_disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The disabled status for the service account. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccountCreateMutex.Lock()
	defer serviceAccountCreateMutex.Unlock()

	client, orgID, err := r.clientFromNewOrgResource(plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	client = client.WithRetries(0, 0) // Disable retries to have our own retry logic

	createReq := models.CreateServiceAccountForm{
		Name:       plan.Name.ValueString(),
		Role:       plan.Role.ValueString(),
		IsDisabled: plan.IsDisabled.ValueBool(),
	}

	var sa *models.ServiceAccountDTO
	retryErr := retry.RetryContext(ctx, 10*time.Second, func() *retry.RetryError {
		params := service_accounts.NewCreateServiceAccountParams().WithBody(&createReq)
		createResp, err := client.ServiceAccounts.CreateServiceAccount(params)
		if err == nil {
			sa = createResp.Payload
			return nil
		}

		if _, ok := err.(*service_accounts.CreateServiceAccountInternalServerError); ok {
			// Sometimes on 500s, the service account is created but the response is not returned.
			// If we just retry, it will conflict because the SA was actually created.
			foundSa, readErr := findServiceAccountByName(client, createReq.Name)
			if readErr != nil {
				return retry.RetryableError(err)
			}
			sa = foundSa
			return nil
		}
		return retry.NonRetryableError(err)
	})
	if retryErr != nil {
		resp.Diagnostics.AddError("Error creating service account", retryErr.Error())
		return
	}

	plan.ID = types.StringValue(MakeOrgResourceID(orgID, strconv.FormatInt(sa.ID, 10)))

	readData, diags := r.read(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, state.ID.ValueString())
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

// serviceAccountClientAndID parses the Terraform composite id and returns an org-scoped client, org ID, and Grafana service account id.
func (r *serviceAccountResource) serviceAccountClientAndID(id string) (*goapi.GrafanaHTTPAPI, int64, int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	client, orgID, split, err := r.clientFromExistingOrgResource(resourceServiceAccountID, id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, 0, 0, diags
	}
	if len(split) == 0 {
		diags.AddError("Invalid resource ID", "Resource ID has no parts")
		return nil, 0, 0, diags
	}
	idInt, ok := split[0].(int64)
	if !ok {
		diags.AddError("Invalid resource ID", "Service account ID is not an integer")
		return nil, 0, 0, diags
	}
	return client, orgID, idInt, diags
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, idInt, diags := r.serviceAccountClientAndID(plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := models.UpdateServiceAccountForm{
		Name:       plan.Name.ValueString(),
		Role:       plan.Role.ValueString(),
		IsDisabled: common.Ref(plan.IsDisabled.ValueBool()),
	}

	params := service_accounts.NewUpdateServiceAccountParams().
		WithBody(&updateReq).
		WithServiceAccountID(idInt)
	if _, err := client.ServiceAccounts.UpdateServiceAccount(params); err != nil {
		resp.Diagnostics.AddError("Error updating service account", err.Error())
		return
	}

	readData, diags := r.read(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, idInt, diags := r.serviceAccountClientAndID(state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := client.ServiceAccounts.DeleteServiceAccount(idInt)
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting service account", err.Error())
		return
	}
}

func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Service account not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *serviceAccountResource) read(ctx context.Context, id string) (*serviceAccountResourceModel, diag.Diagnostics) {
	client, orgID, idInt, diags := r.serviceAccountClientAndID(id)
	if diags.HasError() {
		return nil, diags
	}

	sa, err := retrieveServiceAccount(client, idInt)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Error reading service account", err.Error())
		return nil, diags
	}

	return &serviceAccountResourceModel{
		ID:         types.StringValue(MakeOrgResourceID(orgID, strconv.FormatInt(sa.ID, 10))),
		OrgID:      types.StringValue(strconv.FormatInt(sa.OrgID, 10)),
		Name:       types.StringValue(sa.Name),
		Role:       types.StringValue(sa.Role),
		IsDisabled: types.BoolValue(sa.IsDisabled),
	}, diags
}

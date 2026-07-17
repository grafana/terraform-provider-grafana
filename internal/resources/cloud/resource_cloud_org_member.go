package cloud

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
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
)

var (
	resourceOrgMemberName = "grafana_cloud_org_member"
	resourceOrgMemberID   = common.NewResourceID(common.StringIDField("orgSlugOrID"), common.StringIDField("usernameOrID"))

	// Check interface
	_ resource.ResourceWithImportState = (*orgMemberResource)(nil)
)

func resourceOrgMember() *common.Resource {
	return common.NewResource(
		common.CategoryCloud,
		resourceOrgMemberName,
		resourceOrgMemberID,
		&orgMemberResource{},
	).WithLister(cloudListerFunction(listOrgMembers))
}

type resourceOrgMemberModel struct {
	ID                   types.String `tfsdk:"id"`
	Org                  types.String `tfsdk:"org"`
	User                 types.String `tfsdk:"user"`
	Role                 types.String `tfsdk:"role"`
	ReceiveBillingEmails types.Bool   `tfsdk:"receive_billing_emails"`
}

type orgMemberResource struct {
	basePluginFrameworkResource
}

func (r *orgMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceOrgMemberName
}

func (r *orgMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"org": schema.StringAttribute{
				MarkdownDescription: "The slug or ID of the organization.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "Username or ID of the user to add to the org's members.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "The role to assign to the user in the organization.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("Admin", "Editor", "Viewer", "None"),
				},
			},
			"receive_billing_emails": schema.BoolAttribute{
				MarkdownDescription: "Whether the user should receive billing emails.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
		MarkdownDescription: "Manages the membership of a user in an organization.",
	}
}

func listOrgMembers(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error) {
	var resp *gcom.OrgMemberListResponse
	if err := common.RetryRequest(ctx, "list org members", func() (*http.Response, error) {
		r, httpResp, err := client.OrgsAPI.GetOrgMembers(ctx, data.OrgSlug()).Execute()
		resp = r
		return httpResp, err
	}); err != nil {
		return nil, err
	}

	var ids []string
	for _, member := range resp.Items {
		ids = append(ids, resourceOrgMemberID.Make(data.OrgSlug(), member.UserUsername))
	}

	return ids, nil
}

func (r *orgMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, err := resourceOrgMemberID.Split(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Unable to decode ID")
		return
	}

	data, diags := r.readFromID(ctx, req.ID)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if data == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// orgMemberBilling maps the receive_billing_emails flag to the grafana.com billing value.
func orgMemberBilling(receiveBillingEmails bool) int32 {
	if receiveBillingEmails {
		return 1
	}
	return 0
}

// setOrgMember updates an existing membership's role and billing, retrying transient errors. It
// returns the last HTTP response so callers can detect a 404 (membership removed out-of-band).
func (r *orgMemberResource) setOrgMember(ctx context.Context, org, user string, billing int32, role string) (*http.Response, error) {
	postReq := gcom.NewPostOrgMemberRequest()
	postReq.SetBilling(billing)
	postReq.SetRole(role)
	var httpResp *http.Response
	err := common.RetryRequest(ctx, "update org member", func() (*http.Response, error) {
		_, hr, execErr := r.client.OrgsAPI.PostOrgMember(ctx, org, user).XRequestId(ClientRequestID()).PostOrgMemberRequest(*postReq).Execute()
		httpResp = hr
		return hr, execErr
	})
	return httpResp, err
}

// addOrgMember adds a user to the org with the given role and billing, retrying transient errors.
func (r *orgMemberResource) addOrgMember(ctx context.Context, org, user string, billing int32, role string) error {
	postReq := gcom.NewPostOrgMembersRequest(user)
	postReq.SetBilling(billing)
	postReq.SetRole(role)
	return common.RetryRequest(ctx, "recreate org member", func() (*http.Response, error) {
		_, hr, err := r.client.OrgsAPI.PostOrgMembers(ctx, org).PostOrgMembersRequest(*postReq).XRequestId(ClientRequestID()).Execute()
		return hr, err
	})
}

func (r *orgMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("client not configured", "client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceOrgMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, user := data.Org.ValueString(), data.User.ValueString()
	id := resourceOrgMemberID.Make(org, user)

	// POST
	billing := orgMemberBilling(data.ReceiveBillingEmails.ValueBool())
	postReq := gcom.NewPostOrgMembersRequest(user)
	postReq.SetBilling(billing)
	postReq.SetRole(data.Role.ValueString())

	// Make create idempotent. grafana.com returns 409 when the user is already a member of the
	// org, and a previous attempt may have added the member even though we never observed the
	// response (transient 5xx / dropped connection). In both cases we adopt the existing
	// membership instead of failing. Adoption on its own does not converge, though: the existing
	// membership may carry a different role/billing than the plan, so remember that we adopted
	// and reconcile it with an explicit update below.
	attempt := 0
	adopted := false
	cfg := common.DefaultHTTPRequestRetryConfig()
	cfg.Operation = "create org member"
	cfg.ErrorAnalyzer = func(httpResp *http.Response, err error) error {
		if err == nil {
			return nil
		}
		isConflict := httpResp != nil && httpResp.StatusCode == http.StatusConflict
		if attempt <= 1 && !isConflict {
			return err
		}
		if existing, diags := r.readFromID(ctx, id); !diags.HasError() && existing != nil {
			adopted = true
			return nil
		}
		return err
	}
	if err := common.RetryHTTPRequest(ctx, cfg, func() (*http.Response, error) {
		attempt++
		_, httpResp, err := r.client.OrgsAPI.PostOrgMembers(ctx, org).PostOrgMembersRequest(*postReq).XRequestId(ClientRequestID()).Execute()
		return httpResp, err
	}); err != nil {
		resp.Diagnostics.AddError("Unable to Create Resource", err.Error())
		return
	}

	// If we adopted a pre-existing membership, its role/billing may not match the plan. Update
	// it so create is genuinely idempotent instead of silently leaving a stale role in place.
	if adopted {
		if _, err := r.setOrgMember(ctx, org, user, billing, data.Role.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to reconcile existing org member", err.Error())
			return
		}
	}

	// Read created resource
	readData, diags := r.readFromID(ctx, id)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Unable to read created resource", "Resource not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *orgMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceOrgMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	readData, diags := r.readFromID(ctx, data.ID.ValueString())
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *orgMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("client not configured", "client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceOrgMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, user := data.Org.ValueString(), data.User.ValueString()
	id := resourceOrgMemberID.Make(org, user)

	// POST
	billing := orgMemberBilling(data.ReceiveBillingEmails.ValueBool())
	httpResp, updateErr := r.setOrgMember(ctx, org, user, billing, data.Role.ValueString())
	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		// The membership was removed out-of-band; re-add it so the apply converges instead
		// of failing on a stale update.
		if err := r.addOrgMember(ctx, org, user, billing, data.Role.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to Update Resource", err.Error())
			return
		}
	} else if updateErr != nil {
		resp.Diagnostics.AddError("Unable to Update Resource", updateErr.Error())
		return
	}

	// Read updated resource
	readData, diags := r.readFromID(ctx, id)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Unable to read updated resource", "Resource not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *orgMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("client not configured", "client not configured")
		return
	}

	// Read Terraform prior state data into the model
	var data resourceOrgMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	split, err := resourceOrgMemberID.Split(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to split ID", err.Error())
		return
	}
	org, user := split[0].(string), split[1].(string)

	// DELETE — treat a missing membership as a successful delete so destroying an
	// already-removed member is idempotent.
	cfg := common.DefaultHTTPRequestRetryConfig()
	cfg.Operation = "delete org member"
	cfg.ErrorAnalyzer = common.AcceptNotFound
	if err := common.RetryHTTPRequest(ctx, cfg, func() (*http.Response, error) {
		return r.client.OrgsAPI.DeleteOrgMember(ctx, org, user).XRequestId(ClientRequestID()).Execute()
	}); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Resource", err.Error())
	}
}

func (r *orgMemberResource) readFromID(ctx context.Context, id string) (*resourceOrgMemberModel, diag.Diagnostics) {
	if r.client == nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("client not configured", "client not configured")}
	}

	split, err := resourceOrgMemberID.Split(id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unable to split ID", err.Error())}
	}
	org, user := split[0].(string), split[1].(string)

	// GET
	var memberResp *gcom.FormattedOrgMembership
	var httpResp *http.Response
	err = common.RetryRequest(ctx, "read org member", func() (*http.Response, error) {
		m, resp, execErr := r.client.OrgsAPI.GetOrgMember(ctx, org, user).Execute()
		memberResp, httpResp = m, resp
		return resp, execErr
	})
	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unable to read resource", err.Error())}
	}

	data := &resourceOrgMemberModel{}
	data.ID = types.StringValue(id)
	data.Org = types.StringValue(org)
	data.User = types.StringValue(user)
	data.Role = types.StringValue(memberResp.Role)
	data.ReceiveBillingEmails = types.BoolValue(memberResp.Billing == 1)

	return data, nil
}

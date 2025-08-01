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
	resp, _, err := client.OrgsAPI.GetOrgMembers(ctx, data.OrgSlug()).Execute()
	if err != nil {
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
	id := resourceOrgMemberID.Make(data.Org.ValueString(), data.User.ValueString())

	// POST
	var billing int32 = 0
	if data.ReceiveBillingEmails.ValueBool() {
		billing = 1
	}
	postReq := gcom.NewPostOrgMembersRequest(data.User.ValueString())
	postReq.SetBilling(billing)
	postReq.SetRole(data.Role.ValueString())
	_, _, err := r.client.OrgsAPI.PostOrgMembers(ctx, data.Org.ValueString()).PostOrgMembersRequest(*postReq).XRequestId(ClientRequestID()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Resource", err.Error())
		return
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
	id := resourceOrgMemberID.Make(data.Org.ValueString(), data.User.ValueString())

	// POST
	var billing int32 = 0
	if data.ReceiveBillingEmails.ValueBool() {
		billing = 1
	}
	postReq := gcom.NewPostOrgMemberRequest()
	postReq.SetBilling(billing)
	postReq.SetRole(data.Role.ValueString())
	if _, _, err := r.client.OrgsAPI.PostOrgMember(ctx, data.Org.ValueString(), data.User.ValueString()).XRequestId(ClientRequestID()).PostOrgMemberRequest(*postReq).Execute(); err != nil {
		resp.Diagnostics.AddError("Unable to Update Resource", err.Error())
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

	// DELETE
	if _, err := r.client.OrgsAPI.DeleteOrgMember(ctx, org, user).XRequestId(ClientRequestID()).Execute(); err != nil {
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
	memberResp, httpResp, err := r.client.OrgsAPI.GetOrgMember(ctx, org, user).Execute()
	if httpResp.StatusCode == http.StatusNotFound {
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

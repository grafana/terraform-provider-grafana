package cloud

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
)

func resourceOrgMember() *common.Resource {
	return common.NewPluginFrameworkResource(resourceOrgMemberName, resourceOrgMemberID, &orgMemberResource{})
}

type resourceOrgMemberModel struct {
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

func (r *orgMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceOrgMemberModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *orgMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceOrgMemberModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// GET
	_, httpResp, err := r.client.OrgsAPI.GetOrgMember(ctx, data.Org.ValueString(), data.User.ValueString()).Execute()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Unable to Refresh Resource", err.Error())
		return
	}
}

func (r *orgMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceOrgMemberModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// PUT
	var billing int32 = 0
	if data.ReceiveBillingEmails.ValueBool() {
		billing = 1
	}
	postReq := gcom.NewPostOrgMemberRequest()
	postReq.SetBilling(billing)
	postReq.SetRole(data.Role.ValueString())
	_, _, err := r.client.OrgsAPI.PostOrgMember(ctx, data.Org.ValueString(), data.User.ValueString()).XRequestId(ClientRequestID()).PostOrgMemberRequest(*postReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Resource", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *orgMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceOrgMemberModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// DELETE
	_, err := r.client.OrgsAPI.DeleteOrgMember(ctx, data.Org.ValueString(), data.User.ValueString()).XRequestId(ClientRequestID()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Resource", err.Error())
		return
	}
}

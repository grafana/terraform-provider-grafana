package oncall

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceUserNotificationRuleName = "grafana_oncall_user_notification_rule"

	userNotificationRuleTypeOptions = []string{
		"wait",
		"notify_by_slack",
		"notify_by_msteams",
		"notify_by_sms",
		"notify_by_phone_call",
		"notify_by_telegram",
		"notify_by_email",
		"notify_by_mobile_app",
		"notify_by_mobile_app_critical",
	}
	userNotificationRuleTypeOptionsVerbal = strings.Join(userNotificationRuleTypeOptions, ", ")

	// https://github.com/grafana/oncall/blob/6e0bebaa110a802187254321bd9c3c138fc590b6/engine/apps/base/models/user_notification_policy.py#L123-L129
	userNotificationRuleDurationOptions = []int64{
		60,      // 1 minute
		60 * 5,  // 5 minutes
		60 * 15, // 15 minutes
		60 * 30, // 30 minutes
		60 * 60, // 1 hour
	}
	userNotificationRuleDurationOptionsVerbal = strings.Trim(strings.Join(strings.Fields(fmt.Sprint(userNotificationRuleDurationOptions)), ", "), "[]")

	// Check interface
	_ resource.ResourceWithImportState = (*userNotificationRuleResource)(nil)
)

func resourceUserNotificationRule() *common.Resource {
	return common.NewResource(
		common.CategoryOnCall,
		resourceUserNotificationRuleName,
		resourceID,
		&userNotificationRuleResource{},
	).WithLister(oncallListerFunction(listUserNotificationRules))
}

type resourceUserNotificationRuleModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.String `tfsdk:"user_id"`
	Position  types.Int64  `tfsdk:"position"`
	Duration  types.Int64  `tfsdk:"duration"`
	Important types.Bool   `tfsdk:"important"`
	Type      types.String `tfsdk:"type"`
}

type userNotificationRuleResource struct {
	basePluginFrameworkResource
}

func (r *userNotificationRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceUserNotificationRuleName
}

func (r *userNotificationRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "User ID",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"position": schema.Int64Attribute{
				MarkdownDescription: "Personal notification rules execute one after another starting from position=0. A new escalation policy created with a position of an existing escalation policy will move the old one (and all following) down on the list.",
				Optional:            true,
			},
			"duration": schema.Int64Attribute{
				MarkdownDescription: fmt.Sprintf("A time in seconds to wait (when `type=wait`). Can be %s", userNotificationRuleDurationOptionsVerbal),
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(userNotificationRuleDurationOptions...),
				},
			},
			"important": schema.BoolAttribute{
				MarkdownDescription: "Boolean value which indicates if a rule is “important”",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The type of notification rule. Can be %s. NOTE: `notify_by_msteams` is only available for Grafana Cloud customers.", userNotificationRuleTypeOptionsVerbal),
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(userNotificationRuleTypeOptions...),
				},
			},
		},
		MarkdownDescription: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/personal_notification_rules/)

**Note**: you must be running Grafana OnCall >= v1.8.0 to use this resource.
`,
	}
}

func listUserNotificationRules(client *onCallAPI.Client, listOptions onCallAPI.ListOptions) (ids []string, nextPage *string, err error) {
	resp, _, err := client.UserNotificationRules.ListUserNotificationRules(&onCallAPI.ListUserNotificationRuleOptions{ListOptions: listOptions})
	if err != nil {
		return nil, nil, err
	}
	for _, i := range resp.UserNotificationRules {
		ids = append(ids, i.ID)
	}
	return ids, resp.Next, nil
}

func (r *userNotificationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *userNotificationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceUserNotificationRuleModel
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

func (r *userNotificationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("client not configured", "client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceUserNotificationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createOptions := &onCallAPI.CreateUserNotificationRuleOptions{
		UserId:      data.UserID.ValueString(),
		Type:        data.Type.ValueString(),
		ManualOrder: true,
	}

	if !data.Position.IsNull() {
		p := int(data.Position.ValueInt64())
		createOptions.Position = &p
	}

	if data.Duration.ValueInt64() > 0 {
		d := int(data.Duration.ValueInt64())
		createOptions.Duration = &d
	}

	if data.Important.ValueBool() {
		createOptions.Important = data.Important.ValueBool()
	}

	userNotificationRule, _, err := r.client.UserNotificationRules.CreateUserNotificationRule(createOptions)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Resource", err.Error())
		return
	}

	// Read created resource
	readData, diags := r.readFromID(ctx, userNotificationRule.ID)
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

func (r *userNotificationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("client not configured", "client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceUserNotificationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateOptions := &onCallAPI.UpdateUserNotificationRuleOptions{
		Type:        data.Type.ValueString(),
		ManualOrder: true,
	}

	if !data.Position.IsNull() {
		p := int(data.Position.ValueInt64())
		updateOptions.Position = &p
	}

	if data.Duration.ValueInt64() == 0 {
		updateOptions.Duration = nil
	} else {
		d := int(data.Duration.ValueInt64())
		updateOptions.Duration = &d
	}

	userNotificationRule, _, err := r.client.UserNotificationRules.UpdateUserNotificationRule(data.ID.ValueString(), updateOptions)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Resource", err.Error())
		return
	}

	// Read updated resource
	readData, diags := r.readFromID(ctx, userNotificationRule.ID)
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

func (r *userNotificationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("client not configured", "client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceUserNotificationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// DELETE
	_, err := r.client.UserNotificationRules.DeleteUserNotificationRule(data.ID.ValueString(), &onCallAPI.DeleteUserNotificationRuleOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Resource", err.Error())
	}
}

func (r *userNotificationRuleResource) readFromID(_ context.Context, id string) (*resourceUserNotificationRuleModel, diag.Diagnostics) {
	if r.client == nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("client not configured", "client not configured")}
	}

	// GET
	ruleResp, httpResp, err := r.client.UserNotificationRules.GetUserNotificationRule(id, &onCallAPI.GetUserNotificationRuleOptions{})

	if httpResp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unable to read resource", err.Error())}
	}

	data := &resourceUserNotificationRuleModel{
		ID:        types.StringValue(id),
		UserID:    types.StringValue(ruleResp.UserId),
		Position:  types.Int64Value(int64(ruleResp.Position)),
		Important: types.BoolValue(ruleResp.Important),
		Type:      types.StringValue(ruleResp.Type),
	}

	if ruleResp.Duration == 0 {
		data.Duration = types.Int64Null()
	} else {
		data.Duration = types.Int64Value(int64(ruleResp.Duration))
	}

	return data, nil
}

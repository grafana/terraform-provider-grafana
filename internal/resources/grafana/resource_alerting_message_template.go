package grafana

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

var (
	_ resource.Resource                = &messageTemplateResource{}
	_ resource.ResourceWithConfigure   = &messageTemplateResource{}
	_ resource.ResourceWithImportState = &messageTemplateResource{}
	_ resource.ResourceWithModifyPlan  = &messageTemplateResource{}

	resourceMessageTemplateName = "grafana_message_template"
	resourceMessageTemplateID   = orgResourceIDString("name")
)

func resourceMessageTemplate() *common.Resource {
	return common.NewResource(
		common.CategoryAlerting,
		resourceMessageTemplateName,
		resourceMessageTemplateID,
		&messageTemplateResource{},
	).WithLister(listerFunctionOrgResource(listMessageTemplate))
}

type messageTemplateModel struct {
	ID                types.String `tfsdk:"id"`
	OrgID             types.String `tfsdk:"org_id"`
	Name              types.String `tfsdk:"name"`
	Template          types.String `tfsdk:"template"`
	DisableProvenance types.Bool   `tfsdk:"disable_provenance"`
}

type messageTemplateResource struct {
	basePluginFrameworkResource
}

func (r *messageTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceMessageTemplateName
}

func (r *messageTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages Grafana Alerting notification template groups, including notification templates.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#notification-template-groups)

This resource requires Grafana 9.1.0 or later.
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
				Description: "The name of the notification template group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template": schema.StringAttribute{
				Required:    true,
				Description: "The content of the notification template group.",
			},
			"disable_provenance": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow modifying the message template from other sources than Terraform or the Grafana API. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *messageTemplateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Normalize template to trimmed value so plan matches state when only whitespace differs
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan messageTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Template.IsNull() || plan.Template.IsUnknown() {
		return
	}
	trimmed := strings.TrimSpace(plan.Template.ValueString())
	if trimmed != plan.Template.ValueString() {
		plan.Template = types.StringValue(trimmed)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}
}

func (r *messageTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Message template not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *messageTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan messageTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	name := plan.Name.ValueString()
	content := strings.TrimSpace(plan.Template.ValueString())
	disableProvenance := plan.DisableProvenance.ValueBool()

	var putErr error
	r.commonClient.WithAlertingLock(func() {
		putErr = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
			params := provisioning.NewPutTemplateParams().
				WithName(name).
				WithBody(&models.NotificationTemplateContent{
					Template: content,
				})
			if disableProvenance {
				params.SetXDisableProvenance(&provenanceDisabled)
			}
			if _, err := client.Provisioning.PutTemplate(params); err != nil {
				if err.(runtime.ClientResponseStatus).IsCode(500) {
					return retry.RetryableError(err)
				}
				return retry.NonRetryableError(err)
			}
			return nil
		})
	})
	if putErr != nil {
		resp.Diagnostics.AddError("Failed to create message template", putErr.Error())
		return
	}

	plan.ID = types.StringValue(MakeOrgResourceID(orgID, name))
	readData, diags := r.read(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *messageTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state messageTemplateModel
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

func (r *messageTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan messageTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, split, err := r.clientFromExistingOrgResource(resourceMessageTemplateID, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid resource ID", "Resource ID has no name part")
		return
	}
	name := split[0].(string)
	content := strings.TrimSpace(plan.Template.ValueString())
	disableProvenance := plan.DisableProvenance.ValueBool()

	var putErr error
	r.commonClient.WithAlertingLock(func() {
		putErr = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
			params := provisioning.NewPutTemplateParams().
				WithName(name).
				WithBody(&models.NotificationTemplateContent{
					Template: content,
				})
			if disableProvenance {
				params.SetXDisableProvenance(&provenanceDisabled)
			}
			if _, err := client.Provisioning.PutTemplate(params); err != nil {
				if err.(runtime.ClientResponseStatus).IsCode(500) {
					return retry.RetryableError(err)
				}
				return retry.NonRetryableError(err)
			}
			return nil
		})
	})
	if putErr != nil {
		resp.Diagnostics.AddError("Failed to update message template", putErr.Error())
		return
	}

	readData, diags := r.read(ctx, MakeOrgResourceID(orgID, name))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *messageTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state messageTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceMessageTemplateID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid resource ID", "Resource ID has no name part")
		return
	}
	name := split[0].(string)

	var deleteErr error
	r.commonClient.WithAlertingLock(func() {
		params := provisioning.NewDeleteTemplateParams().WithName(name)
		_, deleteErr = client.Provisioning.DeleteTemplate(params)
	})
	if deleteErr != nil && !common.IsNotFoundError(deleteErr) {
		resp.Diagnostics.AddError("Failed to delete message template", deleteErr.Error())
	}
}

func (r *messageTemplateResource) read(ctx context.Context, id string) (*messageTemplateModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	client, orgID, split, err := r.clientFromExistingOrgResource(resourceMessageTemplateID, id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, diags
	}
	if len(split) == 0 {
		diags.AddError("Invalid resource ID", "Resource ID has no name part")
		return nil, diags
	}
	name := split[0].(string)

	var tmpl *models.NotificationTemplate
	r.commonClient.WithAlertingLock(func() {
		resp, getErr := client.Provisioning.GetTemplate(name)
		if getErr != nil {
			if common.IsNotFoundError(getErr) {
				return
			}
			diags.AddError("Failed to read message template", getErr.Error())
			return
		}
		tmpl = resp.Payload
	})
	if diags.HasError() || tmpl == nil {
		return nil, diags
	}

	return &messageTemplateModel{
		ID:                types.StringValue(MakeOrgResourceID(orgID, tmpl.Name)),
		OrgID:             types.StringValue(strconv.FormatInt(orgID, 10)),
		Name:              types.StringValue(tmpl.Name),
		Template:          types.StringValue(strings.TrimSpace(tmpl.Template)),
		DisableProvenance: types.BoolValue(false), // API does not return provenance
	}, diags
}

func listMessageTemplate(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	if err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.GetTemplates()
		if err != nil {
			if err.(runtime.ClientResponseStatus).IsCode(500) || err.(runtime.ClientResponseStatus).IsCode(403) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		for _, template := range resp.Payload {
			ids = append(ids, MakeOrgResourceID(orgID, template.Name))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return ids, nil
}

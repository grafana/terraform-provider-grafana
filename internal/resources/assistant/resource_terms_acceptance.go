package assistant

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

const (
	resourceTermsAcceptanceName = "grafana_assistant_terms_acceptance"
	termsAcceptanceID           = "terms"
)

var resourceTermsAcceptanceID = common.NewResourceID(common.StringIDField("id"))

type termsAcceptanceResource struct {
	client *assistantapi.Client
}

type termsAcceptanceModel struct {
	ID       types.String `tfsdk:"id"`
	Accepted types.Bool   `tfsdk:"accepted"`
}

func makeResourceTermsAcceptance() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaAssistant,
		resourceTermsAcceptanceName,
		resourceTermsAcceptanceID,
		&termsAcceptanceResource{},
	)
}

func (r *termsAcceptanceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceTermsAcceptanceName
}

func (r *termsAcceptanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Grafana Assistant terms acceptance for the stack configured by the provider. " +
			"Setting `accepted` to `true` constitutes acceptance of the current Assistant terms for the stack. " +
			"The authenticated identity must have the `grafana-assistant-app.settings.terms:write` permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The fixed identifier for the stack's Assistant terms acceptance.",
				Computed:    true,
			},
			"accepted": schema.BoolAttribute{
				Description: "Whether the current Grafana Assistant terms are accepted for the stack. " +
					"Setting this to `false` withdraws acceptance.",
				Required: true,
			},
		},
	}
}

func (r *termsAcceptanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *termsAcceptanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan termsAcceptanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, err := r.setAcceptance(ctx, plan.Accepted.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Assistant terms acceptance", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *termsAcceptanceResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	terms, err := r.client.GetTerms(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Assistant terms acceptance", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, termsAcceptanceToModel(terms))...)
}

func (r *termsAcceptanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan termsAcceptanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, err := r.setAcceptance(ctx, plan.Accepted.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Assistant terms acceptance", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *termsAcceptanceResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if _, err := r.client.SetTermsAcceptance(ctx, false); err != nil {
		resp.Diagnostics.AddError("Failed to withdraw Assistant terms acceptance", err.Error())
	}
}

func (r *termsAcceptanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != termsAcceptanceID {
		resp.Diagnostics.AddError(
			"Invalid Assistant terms acceptance import ID",
			"Use `terms` as the import ID.",
		)
		return
	}

	terms, err := r.client.GetTerms(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Assistant terms acceptance", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, termsAcceptanceToModel(terms))...)
}

func (r *termsAcceptanceResource) setAcceptance(ctx context.Context, accepted bool) (termsAcceptanceModel, error) {
	terms, err := r.client.SetTermsAcceptance(ctx, accepted)
	if err != nil {
		return termsAcceptanceModel{}, err
	}

	return termsAcceptanceToModel(terms), nil
}

func termsAcceptanceToModel(terms assistantapi.Terms) termsAcceptanceModel {
	return termsAcceptanceModel{
		ID:       types.StringValue(termsAcceptanceID),
		Accepted: types.BoolValue(terms.AcceptedTermsAndConditions),
	}
}

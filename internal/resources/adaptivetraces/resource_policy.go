package adaptivetraces

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/adaptivetracesapi"
)

const policyTypeName = "grafana_adaptive_traces_policy"

var policyResourceID = common.NewResourceID(common.StringIDField("id"))

// policyTypes are the types the API lists at GET /adaptive-traces/api/v1/policy_schemas.
var policyTypes = []string{
	"always_sample",
	"and",
	"boolean_attribute",
	"bytes_limiting",
	"composite",
	"diversity",
	"drop",
	"latency",
	"numeric_attribute",
	"ottl_condition",
	"probabilistic",
	"rate_limiting",
	"span_count",
	"status_code",
	"string_attribute",
	"trace_flags",
	"trace_state",
	"volumetric",
}

var (
	_ resource.Resource                = &policyResource{}
	_ resource.ResourceWithConfigure   = &policyResource{}
	_ resource.ResourceWithImportState = &policyResource{}
)

type policyResource struct {
	client *adaptivetracesapi.Client
}

func newPolicyResource() *common.Resource {
	return common.NewResource(
		common.CategoryAdaptiveTraces,
		policyTypeName,
		policyResourceID,
		&policyResource{},
	)
}

func (r *policyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = policyTypeName
}

func (r *policyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Cloud Adaptive Traces sampling policy.\n\n" +
			"Each resource manages one policy, identified by the UUID that the API assigns, and can be imported with it. " +
			"Policies that the API creates itself, such as those of type `anomaly`, are never read, adopted or deleted by this resource.\n\n" +
			"Notes:\n\n" +
			"* Creating the first policy on a stack starts Adaptive Traces filtering, and deleting the last one stops it.\n" +
			"* Only one active `diversity` policy and one active `volumetric` policy are allowed per stack.\n" +
			"* A `drop` policy discards the traces that match all of its sub-policies, where an `and` policy keeps them. " +
			"The API accepts one only while a policy of another type, other than `anomaly`, exists. " +
			"Use `depends_on` to create it after, and destroy it before, the policies it relies on.\n" +
			"* Expiring policies are not supported. A policy that has an expiry cannot be imported, " +
			"and an expiry set outside Terraform is cleared the next time Terraform updates the policy.\n\n" +
			"* [Official documentation](https://grafana.com/docs/grafana-cloud/adaptive-telemetry/adaptive-traces/)\n\n" +
			"Required access policy scopes:\n\n" +
			"* adaptive-traces-policies:read\n" +
			"* adaptive-traces-policies:write\n",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the policy, a UUID assigned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the policy.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: common.AllowedValuesDescription("The type of the policy. It can be changed in place", policyTypes),
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(policyTypes...),
				},
			},
			"body": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "The type-specific body of the policy, as a JSON object string. Use `jsonencode()` to write it. The API validates the body when the policy is created or updated, and its error message is reported when it rejects one.",
				Required:    true,
			},
			"version_created_at": schema.StringAttribute{
				Description: "When the current version of the policy was created, as an RFC 3339 timestamp.",
				Computed:    true,
			},
			"version_created_by": schema.StringAttribute{
				Description: "The user that created the current version of the policy.",
				Computed:    true,
			},
		},
	}
}

func (r *policyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if client.AdaptiveTracesClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Adaptive Traces API.",
			"Please ensure that adaptive_traces_url and adaptive_traces_auth are set in the provider configuration, or set the GRAFANA_ADAPTIVE_TRACES_URL and GRAFANA_ADAPTIVE_TRACES_AUTH environment variables.",
		)
		return
	}

	r.client = client.AdaptiveTracesClient
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreatePolicy(ctx, plan.toPolicy())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create adaptive traces policy", err.Error())
		return
	}

	resp.Diagnostics.Append(plan.setFromPolicy(ctx, created)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetPolicy(ctx, state.ID.ValueString())
	if errors.Is(err, adaptivetracesapi.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read adaptive traces policy", err.Error())
		return
	}

	resp.Diagnostics.Append(state.setFromPolicy(ctx, policy)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdatePolicy(ctx, plan.ID.ValueString(), plan.toPolicy())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update adaptive traces policy", err.Error())
		return
	}

	resp.Diagnostics.Append(plan.setFromPolicy(ctx, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API returns 204 for a policy that is already gone, so a 404 means the request did not reach it (for example,
	// a wrong adaptive_traces_url). Ignoring it would drop a live policy from state.
	if err := r.client.DeletePolicy(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete adaptive traces policy", err.Error())
	}
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := uuid.Parse(req.ID)
	if err != nil || len(req.ID) != len(id.String()) {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected the UUID of a policy, such as 123e4567-e89b-12d3-a456-426614174000, got: %q.", req.ID),
		)
		return
	}

	policy, err := r.client.GetPolicy(ctx, id.String())
	if err != nil {
		if errors.Is(err, adaptivetracesapi.ErrNotFound) {
			resp.Diagnostics.AddError("Failed to import adaptive traces policy", fmt.Sprintf("Policy %s does not exist.", id))
			return
		}
		resp.Diagnostics.AddError("Failed to import adaptive traces policy", err.Error())
		return
	}

	if !slices.Contains(policyTypes, policy.Type) {
		resp.Diagnostics.AddError(
			"Failed to import adaptive traces policy",
			fmt.Sprintf("Policy %s has type %q, which is created by the API and cannot be managed with this resource.", id, policy.Type),
		)
		return
	}
	if policy.ExpiresAt != nil && *policy.ExpiresAt != "" {
		resp.Diagnostics.AddError(
			"Failed to import adaptive traces policy",
			fmt.Sprintf("Policy %s expires at %s. Expiring policies cannot be managed with this resource.", id, *policy.ExpiresAt),
		)
		return
	}

	var state policyModel
	resp.Diagnostics.Append(state.setFromPolicy(ctx, policy)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

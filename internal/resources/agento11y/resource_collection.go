package agento11y

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
)

const resourceCollectionName = "grafana_agento11y_collection"

// The limits match the storage columns: name is a varchar(255), which counts
// characters, and description is a text column, which holds 65535 bytes.
const (
	collectionNameMaxLength        = 255
	collectionDescriptionMaxLength = 65535
)

var resourceCollectionID = common.NewResourceID(common.StringIDField("collection_id"))

// trimmedStringValidator rejects values the API would trim. The resource builds
// its state from the API response, so a value with leading or trailing
// whitespace comes back different and the apply fails with an inconsistent
// result. The check uses strings.TrimSpace, the same function the API uses, so
// the two cannot disagree about which code points count as whitespace.
type trimmedStringValidator struct{}

func (v trimmedStringValidator) Description(_ context.Context) string {
	return "value must not start or end with whitespace"
}

func (v trimmedStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v trimmedStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if strings.TrimSpace(value) != value {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			"Value must not start or end with whitespace. The API trims both ends, which would make the stored value differ from the configuration.",
		)
	}
}

type collectionResource struct {
	client *agento11yapi.Client
}

type collectionModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func makeResourceCollection() *common.Resource {
	return common.NewResource(
		common.CategoryAgentObservability,
		resourceCollectionName,
		resourceCollectionID,
		&collectionResource{},
	)
}

func (r *collectionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceCollectionName
}

func (r *collectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Agent Observability collection: a named group of saved conversations. Rule actions add matching conversations to a collection, so use this resource to reference a collection from `grafana_agento11y_rule_action` without hardcoding a server-generated ID.\n\nTerraform does not manage collection membership. Rule actions and the Agent Observability UI add and remove members.\n\nRequires a Grafana instance with the `grafana-agento11y-app` plugin installed. " + writePermissionDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The server-generated collection identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name of the collection. Names are not unique.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthBetween(1, collectionNameMaxLength),
					trimmedStringValidator{},
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the collection. Remove this attribute to clear the stored description; an empty string is not accepted.",
				Optional:    true,
				Validators: []validator.String{
					// An empty string is rejected rather than stored: the API returns
					// no description for a cleared collection, so state would read
					// back as null and the apply would fail with an inconsistent
					// result. Removing the attribute is the way to clear it.
					stringvalidator.LengthBetween(1, collectionDescriptionMaxLength),
					trimmedStringValidator{},
				},
			},
		},
	}
}

func (r *collectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *collectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan collectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateCollection(ctx, agento11yapi.CollectionCreate{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create agent observability collection", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, collectionToModel(created))...)
}

func (r *collectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state collectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection, err := r.client.GetCollection(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, agento11yapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read agent observability collection", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, collectionToModel(collection))...)
}

func (r *collectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state collectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A null description sends an explicit empty string, which clears the stored
	// value.
	updated, err := r.client.UpdateCollection(ctx, state.ID.ValueString(), agento11yapi.CollectionPatch{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update agent observability collection", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, collectionToModel(updated))...)
}

func (r *collectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state collectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A collection deleted outside Terraform is already in the desired state.
	if err := r.client.DeleteCollection(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, agento11yapi.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to delete agent observability collection", err.Error())
		return
	}
}

func (r *collectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	collection, err := r.client.GetCollection(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import agent observability collection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, collectionToModel(collection))...)
}

func collectionToModel(collection agento11yapi.Collection) collectionModel {
	return collectionModel{
		ID:          types.StringValue(collection.CollectionID),
		Name:        types.StringValue(collection.Name),
		Description: stringValueOrNull(collection.Description),
	}
}

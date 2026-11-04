package incident

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const resourceCustomFieldName = "grafana_incident_custom_field"

// customFieldDomain is the domain of the fields this resource manages. Fields
// in the labels domain are synced with Grafana labels and are not supported.
const customFieldDomain = "incident"

const (
	customFieldTypeString       = "string"
	customFieldTypeSingleSelect = "single-select"
	customFieldTypeMultiSelect  = "multi-select"
	customFieldTypeNumber       = "number"
)

// customFieldTypes are the field types the resource manages. The API also
// accepts bool and date, but the UI does not support them: they break the
// settings and incident pages, and a required bool field blocks resolving
// incidents.
var customFieldTypes = []string{
	customFieldTypeString,
	customFieldTypeSingleSelect,
	customFieldTypeMultiSelect,
	customFieldTypeNumber,
}

var (
	// customFieldSlugRegexp is the API's slug rule (validateKeyName): letters,
	// digits and underscores, starting and ending with a letter.
	customFieldSlugRegexp = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*)?[A-Za-z]$`)
	// customFieldOptionValueRegexp is the API's option value rule
	// (validateOptionValue), which is a non-empty Kubernetes label value.
	customFieldOptionValueRegexp = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$`)
)

var resourceCustomFieldID = common.NewResourceID(common.StringIDField("id"))

var (
	_ resource.Resource                   = &customFieldResource{}
	_ resource.ResourceWithConfigure      = &customFieldResource{}
	_ resource.ResourceWithImportState    = &customFieldResource{}
	_ resource.ResourceWithModifyPlan     = &customFieldResource{}
	_ resource.ResourceWithValidateConfig = &customFieldResource{}
)

type customFieldResource struct {
	client *incident.Client
}

// customFieldModel represents the resource model for an Incident custom field.
type customFieldModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Slug            types.String `tfsdk:"slug"`
	Type            types.String `tfsdk:"type"`
	Description     types.String `tfsdk:"description"`
	Color           types.String `tfsdk:"color"`
	Icon            types.String `tfsdk:"icon"`
	Required        types.Bool   `tfsdk:"required"`
	Archived        types.Bool   `tfsdk:"archived"`
	SelectOptions   types.Set    `tfsdk:"select_option"`
	SelectOptionIDs types.Map    `tfsdk:"select_option_ids"`
}

// customFieldOptionModel is one select_option block.
type customFieldOptionModel struct {
	Value       types.String `tfsdk:"value"`
	Label       types.String `tfsdk:"label"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	Icon        types.String `tfsdk:"icon"`
	Archived    types.Bool   `tfsdk:"archived"`
}

var customFieldOptionType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"value":       types.StringType,
	"label":       types.StringType,
	"description": types.StringType,
	"color":       types.StringType,
	"icon":        types.StringType,
	"archived":    types.BoolType,
}}

func makeResourceCustomField() *common.Resource {
	return common.NewResource(
		common.CategoryIncident,
		resourceCustomFieldName,
		resourceCustomFieldID,
		&customFieldResource{},
	).WithPreferredResourceNameField("slug")
}

func (r *customFieldResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceCustomFieldName
}

func (r *customFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Grafana Incident custom field, which records extra information on every incident, such as the affected team or the customer impact.

The Incident API enforces a few rules that affect Terraform:

* The slug must be unique within the organization. It can be changed without replacing the field.
* The type cannot be changed after the field is created, so changing ` + "`type`" + ` replaces the field.
* A select option is identified by its value. Changing an option's label, description, color or icon updates it in place, but changing its value deletes the option and adds a new one. Option values must be unique within the field.
* **A field or select option that is a value on any incident, open or resolved, cannot be deleted**, archived or not. Deleting it, removing the option, or changing the option's value or the field's type, which deletes and creates it again, fails. To stop managing a field with Terraform without deleting it, use a ` + "`removed`" + ` block with ` + "`lifecycle { destroy = false }`" + `.
* To retire a field or a select option that incidents use, archive it instead: set ` + "`archived = true`" + `. Archived fields and options can't be used on new incidents, but incidents that already use them keep them.
* Default values are not managed by this resource, since the Incident API does not expose them. A default set in the UI is kept.

Only fields in the ` + "`incident`" + ` domain are supported. Fields synced with Grafana labels and system-managed fields cannot be imported.

* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/incident/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/incident/api/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The UUID of the custom field.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The display name of the custom field, for example `Affected team`.",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the custom field, used to search for it and to reference it as a metric. Up to 63 letters, digits and underscores, starting and ending with a letter. It must be unique within the organization.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(customFieldSlugRegexp, "must consist of letters, digits and underscores, and must start and end with a letter"),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the custom field. One of `string`, `single-select`, `multi-select` or `number`. Changing it forces a new custom field to be created, which fails if any incident has a value for the field.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(customFieldTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of what the custom field records.",
				Optional:    true,
			},
			"color": schema.StringAttribute{
				Description: "The color of the custom field, as a CSS color such as `#FF5286`.",
				Optional:    true,
			},
			"icon": schema.StringAttribute{
				Description: "The name of the icon shown for the custom field.",
				Optional:    true,
			},
			"required": schema.BoolAttribute{
				Description: "Whether the custom field must be filled in on every incident. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"archived": schema.BoolAttribute{
				Description: "Whether the custom field is archived. Archived fields can't be used on new incidents, but incidents that already use them keep their values. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"select_option_ids": schema.MapAttribute{
				Description: "The UUIDs of the select options, keyed by the option value.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"select_option": schema.SetNestedBlock{
				Description: "An option of a `single-select` or `multi-select` field. Options are identified by their value, which must be unique within the field.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Description: "The value of the option, which identifies it. Changing it deletes the option and adds a new one, which fails if any incident has the option selected. Up to 63 letters, digits, `-`, `_` and `.`, starting and ending with a letter or digit.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtMost(63),
								stringvalidator.RegexMatches(customFieldOptionValueRegexp, "must consist of letters, digits, '-', '_' and '.', and must start and end with a letter or digit"),
							},
						},
						"label": schema.StringAttribute{
							Description: "The display label of the option.",
							Required:    true,
						},
						"description": schema.StringAttribute{
							Description: "A description of when the option applies.",
							Optional:    true,
						},
						"color": schema.StringAttribute{
							Description: "The color of the option, as a CSS color such as `#FF5286`.",
							Optional:    true,
						},
						"icon": schema.StringAttribute{
							Description: "The name of the icon shown for the option.",
							Optional:    true,
						},
						"archived": schema.BoolAttribute{
							Description: "Whether the option is archived. Archived options can't be selected on new incidents, but incidents that already use them keep them. Defaults to `false`.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func (r *customFieldResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *customFieldResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config customFieldModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.SelectOptions.IsUnknown() || len(config.SelectOptions.Elements()) == 0 {
		return
	}
	if !config.Type.IsUnknown() && !isSelectFieldType(config.Type.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("select_option"),
			"Select options require a select field",
			fmt.Sprintf("A %s field cannot have select options. Only single-select and multi-select fields can.", config.Type.ValueString()),
		)
	}

	// Options are identified by value, and the set only refuses blocks that are
	// identical in every attribute.
	var options []customFieldOptionModel
	resp.Diagnostics.Append(config.SelectOptions.ElementsAs(ctx, &options, false)...)
	seen := map[string]bool{}
	for _, option := range options {
		if option.Value.IsUnknown() {
			continue
		}
		value := option.Value.ValueString()
		if seen[value] {
			resp.Diagnostics.AddAttributeError(path.Root("select_option"),
				"Duplicate select option value",
				fmt.Sprintf("More than one select option has the value %q. Option values must be unique within the field.", value),
			)
		}
		seen[value] = true
	}
}

// ModifyPlan plans select_option_ids, which stay known as long as no option is
// added, and warns before a change that deletes the field or options, since the
// API refuses it, at apply time, when incidents use them.
func (r *customFieldResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan customFieldModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior *customFieldModel
	if !req.State.Raw.IsNull() {
		prior = &customFieldModel{}
		resp.Diagnostics.Append(req.State.Get(ctx, prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	replaced := prior != nil && !plan.Type.IsUnknown() && !plan.Type.Equal(prior.Type)
	if replaced {
		resp.Diagnostics.AddAttributeWarning(path.Root("type"),
			"Changing the type deletes the custom field",
			fmt.Sprintf("The type of custom field %q cannot be changed, so it will be deleted and created again. It will not be deleted if any incident, open or resolved, has a value for it, and the apply will fail. Archive the field instead to retire it.",
				prior.Slug.ValueString()),
		)
	}

	planned, known := optionValues(plan.SelectOptions)
	if !known {
		return
	}

	if prior != nil && !replaced {
		priorValues, _ := optionValues(prior.SelectOptions)
		if removed := slices.DeleteFunc(priorValues, func(value string) bool {
			return slices.Contains(planned, value)
		}); len(removed) > 0 {
			resp.Diagnostics.AddAttributeWarning(path.Root("select_option"),
				"Removing select options deletes them",
				fmt.Sprintf("The select options %s will be deleted from custom field %q. Changing an option's value deletes the option too. An option will not be deleted if any incident, open or resolved, has it selected, and the apply will fail. Archive the option instead to retire it.",
					strings.Join(quoteAll(removed), ", "), prior.Slug.ValueString()),
			)
		}
	}

	// A planned option without a known UUID is a new one, which only the API
	// can give an ID. Otherwise the IDs are the prior ones, minus the options
	// the plan removes.
	priorIDs := map[string]attr.Value{}
	if prior != nil && !replaced {
		priorIDs = prior.SelectOptionIDs.Elements()
	}
	ids := make(map[string]attr.Value, len(planned))
	for _, value := range planned {
		id, ok := priorIDs[value]
		if !ok {
			return
		}
		ids[value] = id
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("select_option_ids"), types.MapValueMust(types.StringType, ids))...)
}

func (r *customFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customFieldModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options, diags := plan.toSelectOptions(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The UUID is chosen here rather than by the API, so that a create that
	// fails after the API stored the field, or that the client retries after a
	// lost response, can be recovered by reading the field back instead of
	// leaving it untracked.
	fieldUUID := uuid.NewString()
	fields := incident.NewFieldsService(r.client)
	_, err := fields.AddField(ctx, incident.AddFieldRequest{
		Field: incident.CustomMetadataField{
			UUID:          fieldUUID,
			Name:          plan.Name.ValueString(),
			Slug:          plan.Slug.ValueString(),
			Type:          plan.Type.ValueString(),
			Description:   plan.Description.ValueString(),
			Color:         plan.Color.ValueString(),
			Icon:          plan.Icon.ValueString(),
			Required:      plan.Required.ValueBool(),
			DomainName:    customFieldDomain,
			Selectoptions: options,
		},
	})
	if err != nil {
		if _, getErr := fields.GetField(ctx, incident.GetFieldRequest{FieldUUID: fieldUUID, IncludeArchived: true}); getErr != nil {
			resp.Diagnostics.AddError("Failed to create incident custom field", err.Error()+r.conflictHint(ctx, plan, fieldUUID, err))
			return
		}
	}

	// AddField ignores the field's archived flag, so it is set separately.
	created, err := r.setFieldArchived(ctx, fieldUUID, plan.Archived.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to archive incident custom field", err.Error())
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fieldUUID)...)
		return
	}

	state, diags := customFieldToModel(ctx, *created, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *customFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customFieldModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := incident.NewFieldsService(r.client).GetField(ctx, incident.GetFieldRequest{FieldUUID: state.ID.ValueString(), IncludeArchived: true})
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read incident custom field", err.Error())
		return
	}

	newState, diags := customFieldToModel(ctx, got.Field, state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *customFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customFieldModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fieldUUID := state.ID.ValueString()

	options, diags := plan.toSelectOptions(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.updateField(ctx, fieldUUID, plan, options); err != nil {
		if incident.HasErrorCode(err, incident.ErrFieldVersionConflict) {
			resp.Diagnostics.AddError("Incident custom field changed during the update",
				fmt.Sprintf("Custom field %q was changed by someone else during each of %d attempts to update it, so the update was refused. Run terraform apply again.", state.Slug.ValueString(), customFieldUpdateAttempts),
			)
			return
		}
		resp.Diagnostics.AddError("Failed to update incident custom field", err.Error()+r.conflictHint(ctx, plan, fieldUUID, err))
		return
	}

	// UpdateField does not write the field's archived flag, and its response
	// leaves archived options out, so the field is read back afterwards.
	updated, err := r.setFieldArchived(ctx, fieldUUID, plan.Archived.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to archive incident custom field", err.Error())
		return
	}

	newState, diags := customFieldToModel(ctx, *updated, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// customFieldUpdateAttempts is how many times updateField sends the update
// before it gives up on version conflicts.
const customFieldUpdateAttempts = 3

// updateField writes the plan to the field. UpdateField refuses a request whose
// version is not the stored one, and every option change bumps it, so the
// version is read right before each attempt rather than kept in state.
//
// The configuration is the desired state, so an edit made after the plan is
// overwritten, as it is when it lands before the read. A version conflict
// means an edit landed between the read and the write, so the field is read
// again and the same update sent with the new version, rather than failing an
// apply whose outcome would be the same one second later.
func (r *customFieldResource) updateField(ctx context.Context, fieldUUID string, plan customFieldModel, options []incident.CustomMetadataFieldSelectOption) error {
	fields := incident.NewFieldsService(r.client)
	var err error
	for range customFieldUpdateAttempts {
		var current *incident.GetFieldResponse
		current, err = fields.GetField(ctx, incident.GetFieldRequest{FieldUUID: fieldUUID, IncludeArchived: true})
		if err != nil {
			return err
		}

		// UpdateField replaces the whole field, select options included, so
		// the request carries every managed field. Options are sent without
		// UUIDs: the API matches them to the stored options by value, and
		// deletes the options the request leaves out, archived or not. The
		// domain and immutability are not managed, so they keep their stored
		// values.
		_, err = fields.UpdateField(ctx, incident.UpdateFieldRequest{
			FieldUUID:     fieldUUID,
			Name:          plan.Name.ValueString(),
			Slug:          plan.Slug.ValueString(),
			Color:         plan.Color.ValueString(),
			Icon:          plan.Icon.ValueString(),
			Description:   plan.Description.ValueString(),
			Required:      plan.Required.ValueBool(),
			DomainName:    current.Field.DomainName,
			Immutable:     current.Field.Immutable,
			Selectoptions: options,
			Version:       current.Field.Version,
		})
		if !incident.HasErrorCode(err, incident.ErrFieldVersionConflict) {
			return err
		}
	}
	return err
}

// setFieldArchived archives or unarchives the field when archived differs from
// what the API holds, and returns the field as the API then
// holds it, archived options included.
func (r *customFieldResource) setFieldArchived(ctx context.Context, fieldUUID string, archived bool) (*incident.CustomMetadataField, error) {
	fields := incident.NewFieldsService(r.client)
	got, err := fields.GetField(ctx, incident.GetFieldRequest{FieldUUID: fieldUUID, IncludeArchived: true})
	if err != nil {
		return nil, err
	}
	if got.Field.Archived == archived {
		return &got.Field, nil
	}

	if archived {
		_, err = fields.ArchiveField(ctx, incident.ArchiveFieldRequest{FieldUUID: fieldUUID})
	} else {
		_, err = fields.UnarchiveField(ctx, incident.UnarchiveFieldRequest{FieldUUID: fieldUUID})
	}
	if err != nil {
		return nil, err
	}

	got, err = fields.GetField(ctx, incident.GetFieldRequest{FieldUUID: fieldUUID, IncludeArchived: true})
	if err != nil {
		return nil, err
	}
	return &got.Field, nil
}

func (r *customFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customFieldModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := incident.NewFieldsService(r.client).DeleteField(ctx, incident.DeleteFieldRequest{FieldUUID: state.ID.ValueString()})
	if err == nil || isNotFound(err) {
		// A field that is already gone is not a failure: Terraform's contract
		// is that Delete succeeds when the object no longer exists.
		return
	}

	if incident.HasErrorCode(err, incident.ErrFieldSystemManaged) {
		resp.Diagnostics.AddError(
			"Incident custom field is system-managed and cannot be deleted",
			fmt.Sprintf("Custom field %q is generated by Grafana and shared across services. To stop managing it with Terraform, use a removed block with lifecycle { destroy = false }, or terraform state rm.", state.Slug.ValueString()),
		)
		return
	}
	resp.Diagnostics.AddError("Failed to delete incident custom field", err.Error())
}

func (r *customFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	got, err := incident.NewFieldsService(r.client).GetField(ctx, incident.GetFieldRequest{FieldUUID: req.ID, IncludeArchived: true})
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Incident custom field not found", fmt.Sprintf("No incident custom field with UUID %q exists.", req.ID))
			return
		}
		resp.Diagnostics.AddError("Failed to import incident custom field", err.Error())
		return
	}

	field := got.Field
	switch {
	case field.DomainName != customFieldDomain:
		resp.Diagnostics.AddError("Incident custom field cannot be imported",
			fmt.Sprintf("Custom field %q is in the %q domain. Only fields in the %q domain can be managed with this resource.", field.Slug, field.DomainName, customFieldDomain),
		)
		return
	case isSystemManagedField(field):
		resp.Diagnostics.AddError("Incident custom field cannot be imported",
			fmt.Sprintf("Custom field %q is generated by Grafana and shared across services, so it cannot be changed or deleted through the Incident API.", field.Slug),
		)
		return
	case !slices.Contains(customFieldTypes, field.Type):
		resp.Diagnostics.AddError("Incident custom field cannot be imported",
			fmt.Sprintf("Custom field %q is a %s field. Only %s fields can be managed with this resource.", field.Slug, field.Type, strings.Join(customFieldTypes, ", ")),
		)
		return
	}

	// Import has no configuration to go on, so every empty optional string is
	// imported as null.
	state, diags := customFieldToModel(ctx, field, customFieldModel{
		Description:   types.StringNull(),
		Color:         types.StringNull(),
		Icon:          types.StringNull(),
		SelectOptions: types.SetNull(customFieldOptionType),
	})
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// conflictHint explains a refused create or update. The API's slug conflict
// does not name the field that holds the slug, so it is looked up. It returns
// "" for any other error, or when the fields cannot be listed.
func (r *customFieldResource) conflictHint(ctx context.Context, plan customFieldModel, fieldUUID string, err error) string {
	if !incident.HasErrorCode(err, incident.ErrFieldSlugConflict) {
		return ""
	}

	fields, listErr := incident.NewFieldsService(r.client).GetFields(ctx, incident.GetFieldsRequest{DomainName: customFieldDomain})
	if listErr != nil {
		return ""
	}
	for _, other := range fields.Fields {
		if other.UUID == fieldUUID || other.Slug != plan.Slug.ValueString() {
			continue
		}
		return fmt.Sprintf("\n\nSlug %q is held by custom field %q (%s). Slugs must be unique within the organization. "+
			"Choose a free slug, or delete the other field first.",
			other.Slug, other.Name, other.UUID)
	}
	return ""
}

// toSelectOptions builds the API select options from the plan, sorted by value
// so that requests are stable. They carry no UUIDs; see Update.
func (m customFieldModel) toSelectOptions(ctx context.Context) ([]incident.CustomMetadataFieldSelectOption, diag.Diagnostics) {
	if m.SelectOptions.IsNull() || m.SelectOptions.IsUnknown() {
		return []incident.CustomMetadataFieldSelectOption{}, nil
	}
	var planned []customFieldOptionModel
	diags := m.SelectOptions.ElementsAs(ctx, &planned, false)
	if diags.HasError() {
		return nil, diags
	}

	options := make([]incident.CustomMetadataFieldSelectOption, 0, len(planned))
	for _, option := range planned {
		options = append(options, incident.CustomMetadataFieldSelectOption{
			Value:       option.Value.ValueString(),
			Label:       option.Label.ValueString(),
			Description: option.Description.ValueString(),
			Color:       option.Color.ValueString(),
			Icon:        option.Icon.ValueString(),
			Archived:    option.Archived.ValueBool(),
		})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Value < options[j].Value })
	return options, diags
}

// customFieldToModel maps an API field onto Terraform state. The field must be
// read with IncludeArchived, so that archived options are in it. The version
// and the source and external ID are deliberately not modelled.
//
// configured is the plan or prior state. The API stores an unset optional
// string as "", so it decides whether an empty value is "" or null, the same
// way severityToModel does. Options are matched to it by value.
func customFieldToModel(ctx context.Context, field incident.CustomMetadataField, configured customFieldModel) (customFieldModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var configuredOptions []customFieldOptionModel
	if !configured.SelectOptions.IsNull() && !configured.SelectOptions.IsUnknown() {
		diags.Append(configured.SelectOptions.ElementsAs(ctx, &configuredOptions, false)...)
	}
	configuredByValue := make(map[string]customFieldOptionModel, len(configuredOptions))
	for _, option := range configuredOptions {
		configuredByValue[option.Value.ValueString()] = option
	}

	elements := make([]customFieldOptionModel, 0, len(field.Selectoptions))
	for _, option := range field.Selectoptions {
		// An option the configuration does not have is new to Terraform, so
		// its empty strings are null, as on import.
		was, ok := configuredByValue[option.Value]
		if !ok {
			was = customFieldOptionModel{Description: types.StringNull(), Color: types.StringNull(), Icon: types.StringNull()}
		}
		elements = append(elements, customFieldOptionModel{
			Value:       types.StringValue(option.Value),
			Label:       types.StringValue(option.Label),
			Description: optionalString(option.Description, was.Description),
			Color:       optionalString(option.Color, was.Color),
			Icon:        optionalString(option.Icon, was.Icon),
			Archived:    types.BoolValue(option.Archived),
		})
	}
	options, d := types.SetValueFrom(ctx, customFieldOptionType, elements)
	diags.Append(d...)

	ids := make(map[string]attr.Value, len(field.Selectoptions))
	for _, option := range field.Selectoptions {
		ids[option.Value] = types.StringValue(option.UUID)
	}

	return customFieldModel{
		ID:              types.StringValue(field.UUID),
		Name:            types.StringValue(field.Name),
		Slug:            types.StringValue(field.Slug),
		Type:            types.StringValue(field.Type),
		Description:     optionalString(field.Description, configured.Description),
		Color:           optionalString(field.Color, configured.Color),
		Icon:            optionalString(field.Icon, configured.Icon),
		Required:        types.BoolValue(field.Required),
		Archived:        types.BoolValue(field.Archived),
		SelectOptions:   options,
		SelectOptionIDs: types.MapValueMust(types.StringType, ids),
	}, diags
}

// isSelectFieldType reports whether fields of the type have select options.
func isSelectFieldType(fieldType string) bool {
	return fieldType == customFieldTypeSingleSelect || fieldType == customFieldTypeMultiSelect
}

// isSystemManagedField reports whether the API refuses to change or delete a
// field, because Grafana generates it and shares it across services. It
// mirrors isFieldSystemManaged in the Incident API.
func isSystemManagedField(field incident.CustomMetadataField) bool {
	return field.Immutable && field.DomainName != "labels"
}

// optionValues returns the sorted values of a select_option set. known is false
// when the set, or any value in it, is unknown.
func optionValues(options types.Set) (values []string, known bool) {
	if options.IsUnknown() {
		return nil, false
	}
	for _, element := range options.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsUnknown() {
			return nil, false
		}
		value, ok := object.Attributes()["value"].(types.String)
		if !ok || value.IsUnknown() {
			return nil, false
		}
		values = append(values, value.ValueString())
	}
	sort.Strings(values)
	return values, true
}

func quoteAll(values []string) []string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = fmt.Sprintf("%q", value)
	}
	return quoted
}

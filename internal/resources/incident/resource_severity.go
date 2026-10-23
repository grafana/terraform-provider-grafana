package incident

import (
	"context"
	"fmt"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const resourceSeverityName = "grafana_incident_severity"

// pendingSeverityLevel is the level of the Pending severity. Every organization
// is seeded with it, incidents fall back to it, and the API refuses to archive
// or delete it.
const pendingSeverityLevel = -1

var resourceSeverityID = common.NewResourceID(common.StringIDField("id"))

var (
	_ resource.Resource                = &severityResource{}
	_ resource.ResourceWithConfigure   = &severityResource{}
	_ resource.ResourceWithImportState = &severityResource{}
	_ resource.ResourceWithModifyPlan  = &severityResource{}
)

type severityResource struct {
	client *incident.Client
}

// severityModel represents the resource model for an Incident Severity.
type severityModel struct {
	ID          types.String `tfsdk:"id"`
	Label       types.String `tfsdk:"label"`
	Level       types.Int64  `tfsdk:"level"`
	Description types.String `tfsdk:"description"`
	Icon        types.String `tfsdk:"icon"`
	DarkColor   types.String `tfsdk:"dark_color"`
	LightColor  types.String `tfsdk:"light_color"`
	Archived    types.Bool   `tfsdk:"archived"`
}

func makeResourceSeverity() *common.Resource {
	return common.NewResource(
		common.CategoryIncident,
		resourceSeverityName,
		resourceSeverityID,
		&severityResource{},
	).WithPreferredResourceNameField("label")
}

func (r *severityResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceSeverityName
}

func (r *severityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Grafana Incident severity, which expresses the impact of an incident.

The Incident API enforces a few rules that affect Terraform:

* Severities can be created at levels ` + "`1`" + ` to ` + "`5`" + `, and labels and levels must be unique within the organization, **including archived severities**. Besides Pending, an organization can therefore hold at most 5 severities, archived or not. A new organization starts with Pending, Critical, Major and Minor, which leaves two levels free. When a new severity's level or label is already taken, the error names the severity that holds it.
* An archived severity keeps its label and level, so neither can be reused until it is deleted. Archiving a severity does not free its level for another one.
* The level cannot be changed after the severity is created, so changing ` + "`level`" + ` replaces the severity. The plan is refused when another severity, archived or not, already holds the new level or label, since the replacement would otherwise delete the severity and then fail to create it. ` + "`create_before_destroy`" + ` cannot be used, because the new severity would conflict with the old one's level or label.
* A severity that any incident uses, open or closed, cannot be deleted, and that includes the delete a replacement makes. Set ` + "`archived = true`" + ` to retire it instead, which keeps the incident history intact. To stop managing it without deleting it, use a ` + "`removed`" + ` block with ` + "`lifecycle { destroy = false }`" + `.

### The Pending severity

Every organization has a system-managed **Pending** severity at level ` + "`-1`" + `, which incidents get when they are declared without one. It can be imported and its label, description, icon and colors changed, but it cannot be archived, its level cannot change, and a second one cannot be created. Destroying it only removes it from the Terraform state; the severity itself is left in place.

* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/incident/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/incident/api/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the severity.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"label": schema.StringAttribute{
				Description: "The display label of the severity, for example `Critical`. Incidents reference the severity by this label. It must be unique within the organization, including archived severities.",
				Required:    true,
			},
			"level": schema.Int64Attribute{
				Description: "The level of the severity, from `1` to `5`; lower is more severe. It must be unique within the organization, including archived severities. Level `-1` belongs to the system-managed Pending severity, which can be imported but not created. Changing it forces a new severity to be created.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Any(
						int64validator.OneOf(pendingSeverityLevel),
						int64validator.Between(1, 5),
					),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of when the severity applies.",
				Optional:    true,
			},
			"icon": schema.StringAttribute{
				Description: "The name of the icon shown for the severity in the UI and in Slack. One of `angle-double-down`, `angle-double-up`, `bolt`, `bug`, `exclamation`, `exclamation-triangle`, `fire`, `question-circle` or `shield`.",
				Optional:    true,
			},
			"dark_color": schema.StringAttribute{
				Description: "The color of the severity in dark mode, as a CSS color such as `#FF5286` or `rgb(255, 82, 134)`.",
				Optional:    true,
			},
			"light_color": schema.StringAttribute{
				Description: "The color of the severity in light mode, as a CSS color such as `#FF5286` or `rgb(255, 82, 134)`.",
				Optional:    true,
			},
			"archived": schema.BoolAttribute{
				Description: "Whether the severity is archived. Archived severities cannot be assigned to new incidents, but remain on historical ones. The Pending severity cannot be archived. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *severityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

// ModifyPlan rejects, at plan time, the changes to the Pending severity that the
// API would only refuse halfway through an apply.
func (r *severityResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan severityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior *severityModel
	if !req.State.Raw.IsNull() {
		prior = &severityModel{}
		resp.Diagnostics.Append(req.State.Get(ctx, prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	planIsPending := !plan.Level.IsUnknown() && plan.Level.ValueInt64() == pendingSeverityLevel
	priorIsPending := prior != nil && prior.Level.ValueInt64() == pendingSeverityLevel

	switch {
	case priorIsPending && !plan.Level.IsUnknown() && !planIsPending:
		// A replace would only drop Pending from state and create a new
		// severity alongside it, which is never what the config means.
		resp.Diagnostics.AddAttributeError(path.Root("level"),
			"The Pending severity's level cannot change",
			"The Pending severity is system-managed and always has level -1.",
		)
	case !priorIsPending && planIsPending:
		resp.Diagnostics.AddAttributeError(path.Root("level"),
			"A severity with level -1 cannot be created",
			"Level -1 belongs to the Pending severity, which every organization already has. Import it to manage it with Terraform instead of creating it.",
		)
	case planIsPending && !plan.Archived.IsUnknown() && plan.Archived.ValueBool():
		resp.Diagnostics.AddAttributeError(path.Root("archived"),
			"The Pending severity cannot be archived",
			"The Pending severity is system-managed: incidents fall back to it, so it must stay active.",
		)
	}

	if resp.Diagnostics.HasError() || prior == nil || plan.Level.IsUnknown() || plan.Level.Equal(prior.Level) {
		return
	}
	r.checkReplacementConflicts(ctx, prior.ID.ValueString(), plan, &resp.Diagnostics)
}

// checkReplacementConflicts refuses a level change when another severity,
// archived or not, already holds the new level or label. A level change
// replaces the severity, and Terraform deletes the old one before creating the
// new one, so a conflict the API only reports on create would delete the
// severity and leave nothing in its place.
func (r *severityResource) checkReplacementConflicts(ctx context.Context, severityID string, plan severityModel, diags *diag.Diagnostics) {
	if r.client == nil {
		return
	}
	severities, err := incident.NewSeveritiesService(r.client).GetOrgSeverities(ctx, incident.GetOrgSeveritiesRequest{})
	if err != nil {
		diags.AddError("Failed to list incident severities",
			"Changing the level replaces the severity, so the new level and label are checked for conflicts before the severity is deleted: "+err.Error(),
		)
		return
	}

	for _, other := range severities.Severities {
		if other.SeverityID == severityID {
			continue
		}
		if int64(other.Level) == plan.Level.ValueInt64() {
			diags.AddAttributeError(path.Root("level"),
				"Incident severity level is already in use",
				fmt.Sprintf("Level %d is held by severity %q%s. Changing the level replaces the severity, so the apply would delete it and then fail to create it at the new level. "+
					"Choose a free level, or delete severity %q first.",
					other.Level, other.DisplayLabel, archivedSuffix(other), other.DisplayLabel),
			)
		}
		if !plan.Label.IsUnknown() && other.DisplayLabel == plan.Label.ValueString() {
			diags.AddAttributeError(path.Root("label"),
				"Incident severity label is already in use",
				fmt.Sprintf("Label %q is held by the severity at level %d%s. Changing the level replaces the severity, so the apply would delete it and then fail to create it with this label. "+
					"Choose a free label, or delete the other severity first.",
					other.DisplayLabel, other.Level, archivedSuffix(other)),
			)
		}
	}
}

// conflictHolder names the severity that holds the level or label a refused
// create asked for, since the API's conflict error does not, and an archived
// one is hidden in the UI. It returns "" for any other error, or when the
// severities cannot be listed.
//
// This is checked at apply rather than plan time: a replacement plans its
// create without prior state, so at plan time a new severity cannot be told
// apart from the replacement of one that still holds the same label.
func (r *severityResource) conflictHolder(ctx context.Context, plan severityModel, err error) string {
	levelConflict := incident.HasErrorCode(err, incident.ErrSeverityLevelConflict)
	labelConflict := incident.HasErrorCode(err, incident.ErrSeverityLabelConflict)
	if !levelConflict && !labelConflict {
		return ""
	}
	severities, listErr := incident.NewSeveritiesService(r.client).GetOrgSeverities(ctx, incident.GetOrgSeveritiesRequest{})
	if listErr != nil {
		return ""
	}

	for _, other := range severities.Severities {
		if levelConflict && int64(other.Level) == plan.Level.ValueInt64() {
			return fmt.Sprintf("\n\nLevel %d is held by severity %q%s. Levels must be unique within the organization, including archived severities. "+
				"Choose a free level, or delete severity %q first.",
				other.Level, other.DisplayLabel, archivedSuffix(other), other.DisplayLabel)
		}
		if labelConflict && other.DisplayLabel == plan.Label.ValueString() {
			return fmt.Sprintf("\n\nLabel %q is held by the severity at level %d%s. Labels must be unique within the organization, including archived severities. "+
				"Choose a free label, or delete the other severity first.",
				other.DisplayLabel, other.Level, archivedSuffix(other))
		}
	}
	return ""
}

// archivedSuffix notes that a conflicting severity is archived, since archived
// severities are hidden in the UI but still hold their level and label.
func archivedSuffix(severity incident.Severity) string {
	if severity.ArchivedAt != "" {
		return ", which is archived"
	}
	return ""
}

func (r *severityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan severityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	severities := incident.NewSeveritiesService(r.client)
	created, err := severities.CreateOrgSeverity(ctx, incident.CreateOrgSeverityRequest{
		Severity: plan.toSeverity(""),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create incident severity", err.Error()+r.conflictHolder(ctx, plan, err))
		return
	}

	state := severityToModel(created.Severity, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() || !plan.Archived.ValueBool() {
		return
	}

	// CreateOrgSeverity has no archived input, so an archived severity is
	// created active and archived afterwards. The state set above records the
	// severity before that, so a refused archive leaves it tracked rather than
	// orphaned. Terraform marks it tainted, so the next apply replaces it.
	if _, err := severities.ArchiveOrgSeverity(ctx, incident.ArchiveOrgSeverityRequest{SeverityID: created.Severity.SeverityID}); err != nil {
		resp.Diagnostics.AddError("Failed to archive incident severity", err.Error())
		return
	}
	state.Archived = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *severityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state severityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := incident.NewSeveritiesService(r.client).GetSeverityByID(ctx, incident.GetSeverityByIDRequest{
		SeverityID: state.ID.ValueString(),
	})
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read incident severity", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, severityToModel(got.Severity, state))...)
}

func (r *severityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state severityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	severityID := state.ID.ValueString()

	// An archived transition goes through ArchiveOrgSeverity and
	// UnarchiveOrgSeverity, since UpdateOrgSeverity never writes archivedAt.
	// Doing it before UpdateOrgSeverity means a refused unarchive leaves the
	// severity untouched.
	if !plan.Archived.Equal(state.Archived) {
		if err := r.setArchived(ctx, severityID, plan.Archived.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Failed to change whether the incident severity is archived", err.Error())
			return
		}

		// Record the archive before UpdateOrgSeverity runs, so that if it fails
		// the next plan does not retry an archive the API already applied. The
		// other fields keep their prior values, since nothing has written them
		// yet.
		state.Archived = plan.Archived
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// UpdateOrgSeverity writes only the fields that differ from the stored
	// severity and never the level or archivedAt, so the request carries every
	// managed field and leaves archivedAt empty.
	updated, err := incident.NewSeveritiesService(r.client).UpdateOrgSeverity(ctx, incident.UpdateOrgSeverityRequest{
		Severity: plan.toSeverity(severityID),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update incident severity", err.Error())
		return
	}

	// UpdateOrgSeverity never writes archivedAt, so its response is not relied
	// on for it: the archive above, if any, already set it to the plan's value.
	newState := severityToModel(updated.Severity, plan)
	newState.Archived = plan.Archived
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *severityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state severityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Pending severity cannot be deleted, so destroying it only stops
	// Terraform managing it. It is checked here rather than left to the API
	// because an API that predates the system-managed guard would delete it,
	// and that breaks every later severity delete in the organization.
	if state.Level.ValueInt64() == pendingSeverityLevel {
		resp.Diagnostics.AddWarning(
			"The Pending severity was not deleted",
			fmt.Sprintf("Severity %q is the system-managed Pending severity, which cannot be deleted. It has been removed from the Terraform state and left in place.", state.Label.ValueString()),
		)
		return
	}

	_, err := incident.NewSeveritiesService(r.client).DeleteOrgSeverity(ctx, incident.DeleteOrgSeverityRequest{
		SeverityID: state.ID.ValueString(),
	})
	if err == nil || isNotFound(err) {
		// A severity that is already gone is not a failure: Terraform's
		// contract is that Delete succeeds when the object no longer exists.
		return
	}

	if incident.HasErrorCode(err, incident.ErrSeverityInUse) {
		resp.Diagnostics.AddError(
			"Incident severity is in use and cannot be deleted",
			fmt.Sprintf("Severity %q is used by %s, and a severity that incidents use cannot be deleted. "+
				"To retire it while keeping the incident history, set archived = true instead. "+
				"To stop managing it with Terraform without deleting it, use a removed block with lifecycle { destroy = false }, or terraform state rm.",
				state.Label.ValueString(), incidentsUsingSeverity(err)),
		)
		return
	}
	resp.Diagnostics.AddError("Failed to delete incident severity", err.Error())
}

func (r *severityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	got, err := incident.NewSeveritiesService(r.client).GetSeverityByID(ctx, incident.GetSeverityByIDRequest{SeverityID: req.ID})
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Incident severity not found", fmt.Sprintf("No incident severity with ID %q exists.", req.ID))
			return
		}
		resp.Diagnostics.AddError("Failed to import incident severity", err.Error())
		return
	}

	// Import has no configuration to go on, so every empty optional string is
	// imported as null.
	resp.Diagnostics.Append(resp.State.Set(ctx, severityToModel(got.Severity, severityModel{
		Description: types.StringNull(),
		Icon:        types.StringNull(),
		DarkColor:   types.StringNull(),
		LightColor:  types.StringNull(),
	}))...)
}

// setArchived archives or unarchives a severity through the dedicated
// endpoints.
func (r *severityResource) setArchived(ctx context.Context, severityID string, archived bool) error {
	severities := incident.NewSeveritiesService(r.client)
	if archived {
		_, err := severities.ArchiveOrgSeverity(ctx, incident.ArchiveOrgSeverityRequest{SeverityID: severityID})
		return err
	}
	_, err := severities.UnarchiveOrgSeverity(ctx, incident.UnarchiveOrgSeverityRequest{SeverityID: severityID})
	return err
}

// toSeverity builds the API severity from the plan. The server-owned fields,
// including archivedAt, are left empty: the API ignores them on update.
func (m severityModel) toSeverity(severityID string) incident.Severity {
	return incident.Severity{
		SeverityID:   severityID,
		DisplayLabel: m.Label.ValueString(),
		Level:        int(m.Level.ValueInt64()),
		Description:  m.Description.ValueString(),
		IconName:     m.Icon.ValueString(),
		DarkColor:    m.DarkColor.ValueString(),
		LightColor:   m.LightColor.ValueString(),
	}
}

// severityToModel maps an API severity onto Terraform state. The timestamps are
// deliberately not modelled, and archived is derived from whether archivedAt
// is set.
//
// configured is the plan or prior state. The API stores an unset optional
// string as "", so it decides whether an empty value is "" or null, the same
// way roleToModel does for the role description.
func severityToModel(severity incident.Severity, configured severityModel) severityModel {
	return severityModel{
		ID:          types.StringValue(severity.SeverityID),
		Label:       types.StringValue(severity.DisplayLabel),
		Level:       types.Int64Value(int64(severity.Level)),
		Description: optionalString(severity.Description, configured.Description),
		Icon:        optionalString(severity.IconName, configured.Icon),
		DarkColor:   optionalString(severity.DarkColor, configured.DarkColor),
		LightColor:  optionalString(severity.LightColor, configured.LightColor),
		Archived:    types.BoolValue(severity.ArchivedAt != ""),
	}
}

// incidentsUsingSeverity describes how many incidents use a severity, from the
// incidents_count the in-use error carries, falling back to a vaguer phrase
// when the count is missing.
func incidentsUsingSeverity(err error) string {
	if apiErr, ok := asAPIError(err); ok {
		// JSON numbers decode into an interface{} as float64.
		if count, ok := apiErr.Context["incidents_count"].(float64); ok {
			if count == 1 {
				return "1 incident"
			}
			return fmt.Sprintf("%d incidents", int(count))
		}
	}
	return "one or more incidents"
}

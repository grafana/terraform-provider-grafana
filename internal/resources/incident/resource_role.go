package incident

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const resourceRoleName = "grafana_incident_role"

var resourceRoleID = common.NewResourceID(common.IntIDField("id"))

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

type roleResource struct {
	client *incident.Client
}

// roleModel represents the resource model for an Incident Role.
type roleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Important   types.Bool   `tfsdk:"important"`
	Mandatory   types.Bool   `tfsdk:"mandatory"`
	Archived    types.Bool   `tfsdk:"archived"`
}

func makeResourceRole() *common.Resource {
	return common.NewResource(
		common.CategoryIncident,
		resourceRoleName,
		resourceRoleID,
		&roleResource{},
	).WithPreferredResourceNameField("name")
}

func (r *roleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceRoleName
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Grafana Incident role that can be assigned to people during an incident.

The Incident API enforces two limits that affect Terraform:

* Creating or unarchiving a role fails when the organization already has **5 active** (non-archived) roles.
* Deleting or archiving a role fails when the organization has **2 or fewer active** roles.

Terraform applies independent resources in parallel, so a configuration that frees an active slot and fills it in the same apply may fail depending on which call reaches the API first. Order the two with ` + "`depends_on`" + `, putting the dependency on whichever role frees the slot:

* At the 5-active limit, the role being created or unarchived depends on the role being archived or deleted.
* At the 2-active limit, the role being archived or deleted depends on the role being created or unarchived.

* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/incident/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/incident/api/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the role, for example `Commander`.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of what the role is responsible for.",
				Optional:    true,
			},
			"important": schema.BoolAttribute{
				Description: "Whether assigning this role is recorded as an important activity on the incident. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"mandatory": schema.BoolAttribute{
				Description: "Whether this role must be assigned on every incident. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"archived": schema.BoolAttribute{
				Description: "Whether the role is archived. Archived roles cannot be assigned on new incidents, but remain on historical ones. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := incident.NewRolesService(r.client).CreateRole(ctx, incident.CreateRoleRequest{
		Role: incident.Role{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Important:   plan.Important.ValueBool(),
			Mandatory:   plan.Mandatory.ValueBool(),
			Archived:    plan.Archived.ValueBool(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create incident role", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, roleToModel(created.Role))...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid incident role ID", err.Error())
		return
	}

	role, err := r.findRole(ctx, roleID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read incident role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, roleToModel(*role))...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid incident role ID", err.Error())
		return
	}

	roles := incident.NewRolesService(r.client)

	// An archived transition has to go through ArchiveRole/UnarchiveRole. Those
	// endpoints enforce the active-role bounds, whereas UpdateRole writes the
	// archived column with no checks at all and would let Terraform push the
	// organization past a limit the API means to hold. Doing it before
	// UpdateRole means a rejected bound leaves the role untouched.
	if !plan.Archived.Equal(state.Archived) {
		if err := r.setArchived(ctx, roleID, plan.Archived.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Failed to change whether the incident role is archived", err.Error())
			return
		}
	}

	// UpdateRole replaces the whole role, so every managed field has to be sent
	// even when only one of them changed. It rewrites the archived column too,
	// but to the value the guarded call above already set.
	updated, err := roles.UpdateRole(ctx, incident.UpdateRoleRequest{
		Role: incident.Role{
			RoleID:      roleID,
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Important:   plan.Important.ValueBool(),
			Mandatory:   plan.Mandatory.ValueBool(),
			Archived:    plan.Archived.ValueBool(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update incident role", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, roleToModel(updated.Role))...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid incident role ID", err.Error())
		return
	}

	if _, err := incident.NewRolesService(r.client).DeleteRole(ctx, incident.DeleteRoleRequest{RoleID: roleID}); err != nil {
		// A role that is already gone is not a failure. Terraform's contract is
		// that Delete succeeds when the object no longer exists, which happens
		// when the role was removed outside Terraform or when the apply runs
		// with -refresh=false.
		var apiErr *incident.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Failed to delete incident role", err.Error())
		return
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	roleID, err := strconv.Atoi(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid incident role ID",
			fmt.Sprintf("Expected a numeric role ID, got %q: %s", req.ID, err.Error()),
		)
		return
	}

	role, err := r.findRole(ctx, roleID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import incident role", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Incident role not found", fmt.Sprintf("No incident role with ID %d exists.", roleID))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, roleToModel(*role))...)
}

// setArchived archives or unarchives a role through the dedicated endpoints,
// which enforce the active-role bounds. The API refuses to archive when the
// organization has 2 or fewer active roles, and to unarchive when it already
// has 5.
func (r *roleResource) setArchived(ctx context.Context, roleID int, archived bool) error {
	roles := incident.NewRolesService(r.client)
	if archived {
		_, err := roles.ArchiveRole(ctx, incident.ArchiveRoleRequest{RoleID: roleID})
		return err
	}
	_, err := roles.UnarchiveRole(ctx, incident.UnarchiveRoleRequest{RoleID: roleID})
	return err
}

// findRole looks a role up by ID. The Incident API has no get-by-ID method for
// roles, so this lists them and filters. A nil role with a nil error means the
// role does not exist.
func (r *roleResource) findRole(ctx context.Context, roleID int) (*incident.Role, error) {
	roles, err := incident.NewRolesService(r.client).GetRoles(ctx, incident.GetRolesRequest{})
	if err != nil {
		return nil, err
	}
	for _, role := range roles.Roles {
		if role.RoleID == roleID {
			return &role, nil
		}
	}
	return nil, nil
}

// roleToModel maps an API role onto Terraform state. CreatedAt and UpdatedAt
// are deliberately not modelled: the API populates them only on reads, not on
// create and update responses, which would make them permanently inconsistent.
func roleToModel(role incident.Role) roleModel {
	description := types.StringNull()
	if role.Description != "" {
		description = types.StringValue(role.Description)
	}

	return roleModel{
		ID:          types.StringValue(strconv.Itoa(role.RoleID)),
		Name:        types.StringValue(role.Name),
		Description: description,
		Important:   types.BoolValue(role.Important),
		Mandatory:   types.BoolValue(role.Mandatory),
		Archived:    types.BoolValue(role.Archived),
	}
}

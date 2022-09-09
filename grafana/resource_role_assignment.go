package grafana

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceRoleAssignment() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 9.1+.
* [Official documentation](https://grafana.com/docs/grafana/latest/enterprise/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/access_control/)
`,
		CreateContext: UpdateRoleAssignments,
		UpdateContext: UpdateRoleAssignments,
		ReadContext:   ReadRoleAssignments,
		DeleteContext: UpdateRoleAssignments,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"role_uid": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Grafana RBAC role UID.",
			},
			"users": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Role assignments to users.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    false,
							Description: "User ID.",
						},
						"global": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							ForceNew:    false,
							Description: "States whether the assignment is available across all organizations or not.",
						},
					},
				},
			},
			"teams": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Role assignments to teams.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    false,
							Description: "Team ID.",
						},
					},
				},
			},
			"service_accounts": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Role assignments to service accounts.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    false,
							Description: "Service account ID.",
						},
					},
				},
			},
		},
	}
}

func ReadRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO check how ot handle repeated requests
	// TODO improve errors from the backend side

	client := meta.(*client).gapi
	uid := d.Get("role_uid").(string)
	assignments, err := client.GetRoleAssignments(uid)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := setRoleAssignments(assignments, d); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(uid)
	return nil
}

func UpdateRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChange("users") && !d.HasChange("teams") && !d.HasChange("service_accounts") {
		return nil
	}

	client := meta.(*client).gapi

	uid := d.Get("role_uid").(string)
	users := collectRoleAssignmentsToFn(d.Get("users"))
	teams := collectRoleAssignmentsToFn(d.Get("teams"))
	serviceAccounts := collectRoleAssignmentsToFn(d.Get("service_accounts"))

	ra := gapi.RoleAssignments{
		RoleUID:         uid,
		Users:           users,
		Teams:           teams,
		ServiceAccounts: serviceAccounts,
	}
	// TODO check why it keeps hammering request after a failed one
	assignments, err := client.UpdateRoleAssignments(ra)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := setRoleAssignments(assignments, d); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setRoleAssignments(assignments *gapi.RoleAssignments, d *schema.ResourceData) error {
	// resolve users
	users := make([]interface{}, 0)
	for _, user := range assignments.Users {
		u := map[string]interface{}{
			"id":     user.ID,
			"global": user.Global,
		}
		users = append(users, u)
	}
	if err := d.Set("users", users); err != nil {
		return err
	}

	// resolve teams
	teams := make([]interface{}, 0)
	for _, team := range assignments.Teams {
		t := map[string]interface{}{
			"id": team.ID,
		}
		teams = append(teams, t)
	}
	if err := d.Set("teams", teams); err != nil {
		return err
	}

	// resolve service accounts
	serviceAccounts := make([]interface{}, 0)
	for _, sa := range assignments.ServiceAccounts {
		sa := map[string]interface{}{
			"id": sa.ID,
		}
		serviceAccounts = append(serviceAccounts, sa)
	}
	if err := d.Set("service_accounts", serviceAccounts); err != nil {
		return err
	}

	return nil
}

func collectRoleAssignmentsToFn(r interface{}) []gapi.RoleAssignment {
	output := make([]gapi.RoleAssignment, 0)
	for _, r := range r.(*schema.Set).List() {
		el := r.(map[string]interface{})
		roleAssignment := gapi.RoleAssignment{ID: el["id"].(int)}
		if el["global"] != nil {
			roleAssignment.Global = el["global"].(bool)
		}
		output = append(output, roleAssignment)
	}
	return output
}

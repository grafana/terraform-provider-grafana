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
							ForceNew:    true,
							Description: "User ID.",
						},
						"global": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							ForceNew:    true,
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
							ForceNew:    true,
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
							ForceNew:    true,
							Description: "Service account ID.",
						},
					},
				},
			},
		},
	}
}

func ReadRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	uid := d.Get("role_uid").(string)
	assignments, err := client.GetRoleAssignments(uid)

	if err != nil {
		return diag.FromErr(err)
	}

	// resolve users
	users := make([]interface{}, 0)
	for _, user := range assignments.Users {
		u := map[string]interface{}{
			"id":     user.ID,
			"global": user.Global,
		}
		users = append(users, u)
	}

	if err = d.Set("users", users); err != nil {
		return diag.FromErr(err)
	}

	// resolve teams
	teams := make([]interface{}, 0)
	for _, teamId := range assignments.Teams {
		t := map[string]interface{}{
			"id": teamId,
		}
		teams = append(teams, t)
	}

	if err = d.Set("teams", teams); err != nil {
		return diag.FromErr(err)
	}

	// resolve service accounts
	serviceAccounts := make([]interface{}, 0)
	for _, saId := range assignments.ServiceAccount {
		sa := map[string]interface{}{
			"id": saId,
		}
		serviceAccounts = append(serviceAccounts, sa)
	}

	if err = d.Set("service_accounts", serviceAccounts); err != nil {
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
	users := collectRoleAssignmentsToUsers(d)
	teams := collectRoleAssignmentsToFn(d.Get("teams"))
	serviceAccounts := collectRoleAssignmentsToFn(d.Get("service_accounts"))

	ra := gapi.RoleAssignments{
		UID:            uid,
		Users:          users,
		Teams:          teams,
		ServiceAccount: serviceAccounts,
	}
	err := client.UpdateRoleAssignments(ra)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func collectRoleAssignmentsToUsers(d *schema.ResourceData) []gapi.UserRoleAssignment {
	users := d.Get("users")
	output := make([]gapi.UserRoleAssignment, 0)
	for _, r := range users.(*schema.Set).List() {
		user := r.(map[string]interface{})
		id := user["id"].(int)
		global := user["global"].(bool)
		output = append(output, gapi.UserRoleAssignment{
			ID:     id,
			Global: global,
		})
	}
	return output
}

func collectRoleAssignmentsToFn(r interface{}) []int {
	output := make([]int, 0)
	for _, r := range r.(*schema.Set).List() {
		el := r.(map[string]interface{})
		id := el["id"].(int)
		output = append(output, id)
	}
	return output
}

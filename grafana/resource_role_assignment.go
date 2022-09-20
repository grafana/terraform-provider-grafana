package grafana

import (
	"context"
	"fmt"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceRoleAssignment() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 9.2+.
* [Official documentatigrafana_role_assignmenton](https://grafana.com/docs/grafana/latest/enterprise/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/access_control/)
`,
		CreateContext: UpdateRoleAssignments,
		UpdateContext: UpdateRoleAssignments,
		ReadContext:   ReadRoleAssignments,
		DeleteContext: UpdateRoleAssignments,
		// Import either by UID
		Importer: &schema.ResourceImporter{
			StateContext: func(c context.Context, rd *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				rd.Set("role_uid", rd.Id())
				return []*schema.ResourceData{rd}, nil
			},
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
				ForceNew:    false,
				Description: "IDs of users that the role should be assigned to.",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"teams": {
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    false,
				Description: "IDs of teams that the role should be assigned to.",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"service_accounts": {
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    false,
				Description: "IDs of service accounts that the role should be assigned to.",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
		},
	}
}

func ReadRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO test what happens when roles are set for ie teams and then the team id list is removed - do they get cleared correctly or is the previous state used?
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
	users, err := collectRoleAssignmentsToFn(d.Get("users"))
	if err != nil {
		return diag.Errorf("invalid user IDs specifiedL %v", err)
	}
	teams, err := collectRoleAssignmentsToFn(d.Get("teams"))
	if err != nil {
		return diag.Errorf("invalid team IDs specifiedL %v", err)
	}
	serviceAccounts, err := collectRoleAssignmentsToFn(d.Get("service_accounts"))
	if err != nil {
		return diag.Errorf("invalid service account IDs specifiedL %v", err)
	}

	ra := &gapi.RoleAssignments{
		RoleUID:         uid,
		Users:           users,
		Teams:           teams,
		ServiceAccounts: serviceAccounts,
	}
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
	if err := d.Set("users", assignments.Users); err != nil {
		return err
	}
	if err := d.Set("teams", assignments.Teams); err != nil {
		return err
	}
	if err := d.Set("service_accounts", assignments.ServiceAccounts); err != nil {
		return err
	}

	return nil
}

func collectRoleAssignmentsToFn(r interface{}) ([]int, error) {
	output := make([]int, 0)
	for _, rID := range r.(*schema.Set).List() {
		id, ok := rID.(int)
		if !ok {
			return []int{}, fmt.Errorf("%s is not a valid id", rID)
		}
		output = append(output, id)
	}
	return output, nil
}

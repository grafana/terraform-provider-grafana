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
**Note:** This resource is available only with Grafana Enterprise 9.2+.
* [Official documentation](https://grafana.com/docs/grafana/latest/enterprise/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/access_control/)
`,
		CreateContext: UpdateRoleAssignments,
		UpdateContext: UpdateRoleAssignments,
		ReadContext:   ReadRoleAssignments,
		DeleteContext: UpdateRoleAssignments,
		// Import either by UID
		Importer: &schema.ResourceImporter{
			StateContext: func(c context.Context, rd *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				//_, err := strconv.ParseInt(rd.Id(), 10, 64)
				//if err != nil {
				//	// If the ID is not a number, then it may be a UID
				//	client := meta.(*client).gapi
				//	folder, err := client.FolderByUID(rd.Id())
				//	if err != nil {
				//		return nil, fmt.Errorf("failed to find folder by ID or UID '%s': %w", rd.Id(), err)
				//	}
				//	rd.SetId(strconv.FormatInt(folder.ID, 10))
				//}
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
				Description: "Role assignments to users.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    false,
							Description: "User ID.",
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
	for _, userID := range assignments.Users {
		u := map[string]interface{}{
			"id": userID,
		}
		users = append(users, u)
	}
	if err := d.Set("users", users); err != nil {
		return err
	}

	// resolve teams
	teams := make([]interface{}, 0)
	for _, teamID := range assignments.Teams {
		t := map[string]interface{}{
			"id": teamID,
		}
		teams = append(teams, t)
	}
	if err := d.Set("teams", teams); err != nil {
		return err
	}

	// resolve service accounts
	serviceAccounts := make([]interface{}, 0)
	for _, saID := range assignments.ServiceAccounts {
		sa := map[string]interface{}{
			"id": saID,
		}
		serviceAccounts = append(serviceAccounts, sa)
	}
	if err := d.Set("service_accounts", serviceAccounts); err != nil {
		return err
	}

	return nil
}

func collectRoleAssignmentsToFn(r interface{}) []int {
	output := make([]int, 0)
	for _, r := range r.(*schema.Set).List() {
		el := r.(map[string]interface{})
		output = append(output, el["id"].(int))
	}
	return output
}

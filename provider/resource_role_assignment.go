package provider

import (
	"context"
	"fmt"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceRoleAssignment() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 9.2+.
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/access_control/)
`,
		CreateContext: UpdateRoleAssignments,
		UpdateContext: UpdateRoleAssignments,
		ReadContext:   ReadRoleAssignments,
		DeleteContext: DeleteRoleAssignments,
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
	client := meta.(*common.Client).GrafanaAPI
	uid := d.Id()
	assignments, err := client.GetRoleAssignments(uid)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := setRoleAssignments(assignments, d); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func UpdateRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.IsNewResource() && !d.HasChange("users") && !d.HasChange("teams") && !d.HasChange("service_accounts") {
		return nil
	}

	client := meta.(*common.Client).GrafanaAPI

	uid := d.Get("role_uid").(string)
	users, err := collectRoleAssignmentsToFn(d.Get("users"))
	if err != nil {
		return diag.Errorf("invalid user IDs specified %v", err)
	}
	teams, err := collectRoleAssignmentsToFn(d.Get("teams"))
	if err != nil {
		return diag.Errorf("invalid team IDs specified %v", err)
	}
	serviceAccounts, err := collectRoleAssignmentsToFn(d.Get("service_accounts"))
	if err != nil {
		return diag.Errorf("invalid service account IDs specified %v", err)
	}

	ra := &gapi.RoleAssignments{
		RoleUID:         uid,
		Users:           users,
		Teams:           teams,
		ServiceAccounts: serviceAccounts,
	}
	if _, err := client.UpdateRoleAssignments(ra); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(uid)
	return ReadRoleAssignments(ctx, d, meta)
}

func DeleteRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	uid := d.Get("role_uid").(string)
	ra := &gapi.RoleAssignments{
		RoleUID:         uid,
		Users:           []int{},
		Teams:           []int{},
		ServiceAccounts: []int{},
	}

	if _, err := client.UpdateRoleAssignments(ra); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func setRoleAssignments(assignments *gapi.RoleAssignments, d *schema.ResourceData) error {
	d.SetId(assignments.RoleUID)
	if err := d.Set("role_uid", assignments.RoleUID); err != nil {
		return err
	}
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

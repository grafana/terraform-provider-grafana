package grafana

import (
	"context"
	"fmt"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
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
			"org_id": orgIDAttribute(),
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
				// Ignore the org ID of the team when hashing. It works with or without it.
				Set: func(i interface{}) int {
					_, teamID := SplitOrgResourceID(i.(string))
					return schema.HashString(teamID)
				},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"service_accounts": {
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    false,
				Description: "IDs of service accounts that the role should be assigned to.",
				// Ignore the org ID of the team when hashing. It works with or without it.
				Set: func(i interface{}) int {
					_, saID := SplitOrgResourceID(i.(string))
					return schema.HashString(saID)
				},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func ReadRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := ClientFromExistingOrgResource(meta, d.Id())
	assignments, err := client.GetRoleAssignments(uid)
	if err, shouldReturn := common.CheckReadError("role assignments", d, err); shouldReturn {
		return err
	}

	return diag.FromErr(setRoleAssignments(assignments, d))
}

func UpdateRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.IsNewResource() && !d.HasChange("users") && !d.HasChange("teams") && !d.HasChange("service_accounts") {
		return nil
	}

	client, orgID := ClientFromNewOrgResource(meta, d)
	uid := d.Get("role_uid").(string)
	users, err := collectRoleAssignmentsToFn(d.Get("users"))
	if err != nil {
		return diag.Errorf("invalid user IDs specified %v", err)
	}
	teamsStrings := d.Get("teams").(*schema.Set).List()
	teams := make([]int, len(teamsStrings))
	for i, t := range teamsStrings {
		_, teamIDStr := SplitOrgResourceID(t.(string))
		teams[i], _ = strconv.Atoi(teamIDStr)
	}
	serviceAccountsStrings := d.Get("service_accounts").(*schema.Set).List()
	serviceAccounts := make([]int, len(serviceAccountsStrings))
	for i, t := range serviceAccountsStrings {
		_, saIDStr := SplitOrgResourceID(t.(string))
		serviceAccounts[i], _ = strconv.Atoi(saIDStr)
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

	d.SetId(MakeOrgResourceID(orgID, uid))
	return ReadRoleAssignments(ctx, d, meta)
}

func DeleteRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := ClientFromExistingOrgResource(meta, d.Id())

	ra := &gapi.RoleAssignments{
		RoleUID:         uid,
		Users:           []int{},
		Teams:           []int{},
		ServiceAccounts: []int{},
	}

	_, err := client.UpdateRoleAssignments(ra)
	return diag.FromErr(err)
}

func setRoleAssignments(assignments *gapi.RoleAssignments, d *schema.ResourceData) error {
	if err := d.Set("role_uid", assignments.RoleUID); err != nil {
		return err
	}
	if err := d.Set("users", assignments.Users); err != nil {
		return err
	}
	teams := make([]string, len(assignments.Teams))
	for i, t := range assignments.Teams {
		teams[i] = strconv.Itoa(t)
	}
	if err := d.Set("teams", teams); err != nil {
		return err
	}
	serviceAccounts := make([]string, len(assignments.ServiceAccounts))
	for i, sa := range assignments.ServiceAccounts {
		serviceAccounts[i] = strconv.Itoa(sa)
	}
	if err := d.Set("service_accounts", serviceAccounts); err != nil {
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

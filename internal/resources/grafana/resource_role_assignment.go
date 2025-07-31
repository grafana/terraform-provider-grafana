package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRoleAssignment() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages the entire set of assignments for a role. Assignments that aren't specified when applying this resource will be removed.
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
					_, saID := SplitServiceAccountID(i.(string))
					return schema.HashString(saID)
				},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
		"grafana_role_assignment",
		orgResourceIDString("roleUID"),
		schema,
	)
}

func ReadRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	resp, err := client.AccessControl.GetRoleAssignments(uid)
	if err, shouldReturn := common.CheckReadError("role assignments", d, err); shouldReturn {
		return err
	}

	return diag.FromErr(setRoleAssignments(resp.Payload, d))
}

func UpdateRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.IsNewResource() && !d.HasChange("users") && !d.HasChange("teams") && !d.HasChange("service_accounts") {
		return nil
	}

	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	uid := d.Get("role_uid").(string)

	ra := models.SetRoleAssignmentsCommand{
		Users:           collectRoleAssignents(d.Get("users"), false),
		Teams:           collectRoleAssignents(d.Get("teams"), true),
		ServiceAccounts: collectRoleAssignents(d.Get("service_accounts"), true),
	}
	if _, err := client.AccessControl.SetRoleAssignments(uid, &ra); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, uid))
	return ReadRoleAssignments(ctx, d, meta)
}

func DeleteRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())

	_, err := client.AccessControl.SetRoleAssignments(uid, &models.SetRoleAssignmentsCommand{
		ServiceAccounts: []int64{},
		Teams:           []int64{},
		Users:           []int64{},
	})
	return diag.FromErr(err)
}

func setRoleAssignments(assignments *models.RoleAssignmentsDTO, d *schema.ResourceData) error {
	if err := d.Set("role_uid", assignments.RoleUID); err != nil {
		return err
	}
	if err := d.Set("users", assignments.Users); err != nil {
		return err
	}
	teams := make([]string, len(assignments.Teams))
	for i, t := range assignments.Teams {
		teams[i] = strconv.FormatInt(t, 10)
	}
	if err := d.Set("teams", teams); err != nil {
		return err
	}
	serviceAccounts := make([]string, len(assignments.ServiceAccounts))
	for i, sa := range assignments.ServiceAccounts {
		serviceAccounts[i] = strconv.FormatInt(sa, 10)
	}
	if err := d.Set("service_accounts", serviceAccounts); err != nil {
		return err
	}

	return nil
}

func collectRoleAssignents(r interface{}, orgScoped bool) []int64 {
	var output []int64
	for _, rID := range r.(*schema.Set).List() {
		var id int64
		if orgScoped {
			_, idStr := SplitServiceAccountID(rID.(string))
			id, _ = strconv.ParseInt(idStr, 10, 64)
		} else {
			if idInt, ok := rID.(int); ok {
				id = int64(idInt)
			}
		}
		output = append(output, id)
	}
	return output
}

package grafana

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceBuiltInRoleAssignment() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 8.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/enterprise/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/access_control/)
`,
		CreateContext: CreateBuiltInRoleAssignment,
		UpdateContext: UpdateBuiltInRoleAssignments,
		ReadContext:   ReadBuiltInRole,
		DeleteContext: DeleteBuiltInRole,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// Built-in roles are all organization roles and Grafana Admin
			"builtin_role": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"Grafana Admin", "Admin", "Editor", "Viewer"}, false),
			},
			"roles": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uid": {
							Type:     schema.TypeString,
							Required: true,
						},
						"global": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func CreateBuiltInRoleAssignment(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("builtin_role").(string)
	if dg := updateAssignments(d, meta); dg != nil {
		return dg
	}
	d.SetId(name)
	return nil
}

func updateAssignments(d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	stateRoles, configRoles, err := collectRoles(d)
	if err != nil {
		return diag.FromErr(err)
	}
	//compile the list of differences between current state and config
	changes := roleChanges(stateRoles, configRoles)
	brName := d.Get("builtin_role").(string)
	//now we can make the corresponding updates so current state matches config
	if err := createOrRemove(meta, brName, changes); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func ReadBuiltInRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)
	brName := d.Id()
	builtInRoles, err := client.GetBuiltInRoleAssignments()

	if err != nil {
		return diag.FromErr(err)
	}

	brRole := builtInRoles[brName]
	if builtInRoles[brName] == nil {
		log.Printf("[WARN] removing built-in role %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}

	roles := make([]interface{}, 0)
	for _, br := range brRole {
		rm := map[string]interface{}{
			"uid":    br.UID,
			"global": br.Global,
		}
		roles = append(roles, rm)
	}

	if err = d.Set("roles", roles); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("builtin_role", brName); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(brName)
	return nil
}

func UpdateBuiltInRoleAssignments(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChange("roles") {
		return nil
	}

	if dg := updateAssignments(d, meta); dg != nil {
		return dg
	}

	return nil
}

func DeleteBuiltInRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	for _, r := range d.Get("roles").(*schema.Set).List() {
		role := r.(map[string]interface{})
		bra := gapi.BuiltInRoleAssignment{
			RoleUID:     role["uid"].(string),
			BuiltinRole: d.Id(),
			Global:      role["global"].(bool),
		}
		err := client.DeleteBuiltInRoleAssignment(bra)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId("")
	return nil
}

type RoleChange struct {
	Type   ChangeRoleType
	UID    string
	Global bool
}

type ChangeRoleType int8

const (
	AddRole ChangeRoleType = iota
	RemoveRole
)

func roleChanges(rolesInState, rolesInConfig map[string]bool) []RoleChange {
	var changes []RoleChange
	for uid, g := range rolesInConfig {
		if _, ok := rolesInState[uid]; !ok {
			changes = append(changes, RoleChange{Type: AddRole, UID: uid, Global: g})
		}
	}
	for uid, g := range rolesInState {
		if _, ok := rolesInConfig[uid]; !ok {
			changes = append(changes, RoleChange{Type: RemoveRole, UID: uid, Global: g})
		}
	}
	return changes
}

func collectRoles(d *schema.ResourceData) (map[string]bool, map[string]bool, error) {

	errFn := func(uid string) error {
		return errors.New(fmt.Sprintf("Error: Role '%s' cannot be specified multiple times.", uid))
	}

	rolesFn := func(roles interface{}) (map[string]bool, error) {
		output := make(map[string]bool)
		for _, r := range roles.(*schema.Set).List() {
			role := r.(map[string]interface{})
			uid := role["uid"].(string)
			if _, ok := output[uid]; ok {
				return nil, errFn(uid)
			}
			output[uid] = role["global"].(bool)
		}
		return output, nil
	}

	state, config := d.GetChange("roles")
	rolesInState, err := rolesFn(state)
	if err != nil {
		return nil, nil, err
	}
	rolesInConfig, err := rolesFn(config)
	if err != nil {
		return nil, nil, err
	}

	return rolesInState, rolesInConfig, nil
}

func createOrRemove(meta interface{}, name string, changes []RoleChange) error {
	client := meta.(*gapi.Client)
	var err error
	for _, c := range changes {
		br := gapi.BuiltInRoleAssignment{BuiltinRole: name, RoleUID: c.UID, Global: c.Global}
		switch c.Type {
		case AddRole:
			_, err = client.NewBuiltInRoleAssignment(br)
		case RemoveRole:
			err = client.DeleteBuiltInRoleAssignment(br)
		}
		if err != nil {
			return errors.New(fmt.Sprintf("Error with %s %v", name, err))
		}
	}
	return nil
}

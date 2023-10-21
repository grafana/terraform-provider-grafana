package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceRole() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 8.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/access_control/)
`,
		CreateContext: CreateRole,
		UpdateContext: UpdateRole,
		ReadContext:   ReadRole,
		DeleteContext: DeleteRole,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Description: "Unique identifier of the role. Used for assignments.",
			},
			"version": {
				Type:         schema.TypeInt,
				Description:  "Version of the role. A role is updated only on version increase. This field or `auto_increment_version` should be set.",
				Optional:     true,
				ExactlyOneOf: []string{"version", "auto_increment_version"},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return new == "0" || old == new // new will be 0 when switching from manually versioned to auto_increment_version
				},
			},
			"auto_increment_version": {
				Type:         schema.TypeBool,
				Description:  "Whether the role version should be incremented automatically on updates (and set to 1 on creation). This field or `version` should be set.",
				Optional:     true,
				ExactlyOneOf: []string{"version", "auto_increment_version"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the role",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the role.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Display name of the role. Available with Grafana 8.5+.",
			},
			"group": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Group of the role. Available with Grafana 8.5+.",
			},
			"hidden": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Boolean to state whether the role should be visible in the Grafana UI or not. Available with Grafana 8.5+.",
			},
			"global": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Boolean to state whether the role is available across all organizations or not.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Specific set of actions granted by the role.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Specific action users granted with the role will be allowed to perform (for example: `users:read`)",
						},
						"scope": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Scope to restrict the action to a set of resources (for example: `users:*` or `roles:customrole1`)",
						},
					},
				},
			},
		},
	}
}

func CreateRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromNewOrgResource(meta, d)

	var version int
	if d.Get("auto_increment_version").(bool) {
		version = 1
	} else {
		version = d.Get("version").(int)
	}

	role := gapi.Role{
		UID:         d.Get("uid").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Version:     int64(version),
		Global:      d.Get("global").(bool),
		DisplayName: d.Get("display_name").(string),
		Group:       d.Get("group").(string),
		Hidden:      d.Get("hidden").(bool),
		Permissions: permissions(d),
	}
	r, err := client.NewRole(role)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, r.UID))
	return ReadRole(ctx, d, meta)
}

func permissions(d *schema.ResourceData) []gapi.Permission {
	p, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}

	perms := make([]gapi.Permission, 0)
	for _, permission := range p.(*schema.Set).List() {
		p := permission.(map[string]interface{})
		perms = append(perms, gapi.Permission{
			Action: p["action"].(string),
			Scope:  p["scope"].(string),
		})
	}

	return perms
}

func ReadRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := ClientFromExistingOrgResource(meta, d.Id())
	return readRoleFromUID(client, uid, d)
}

func readRoleFromUID(client *gapi.Client, uid string, d *schema.ResourceData) diag.Diagnostics {
	r, err := client.GetRole(uid)
	if err, shouldReturn := common.CheckReadError("role", d, err); shouldReturn {
		return err
	}

	err = d.Set("version", r.Version)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("name", r.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("uid", r.UID)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("description", r.Description)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("display_name", r.DisplayName)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("group", r.Group)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("global", r.Global)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("hidden", r.Hidden)
	if err != nil {
		return diag.FromErr(err)
	}
	perms := make([]interface{}, 0)
	for _, p := range r.Permissions {
		pMap := map[string]interface{}{
			"action": p.Action,
			"scope":  p.Scope,
		}
		perms = append(perms, pMap)
	}
	err = d.Set("permissions", perms)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func UpdateRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := ClientFromExistingOrgResource(meta, d.Id())

	if d.HasChange("version") || d.HasChange("name") || d.HasChange("description") || d.HasChange("permissions") ||
		d.HasChange("display_name") || d.HasChange("group") || d.HasChange("hidden") {
		version := d.Get("version").(int)
		if d.Get("auto_increment_version").(bool) {
			version += 1
		}

		r := gapi.Role{
			UID:         uid,
			Name:        d.Get("name").(string),
			Global:      d.Get("global").(bool),
			Description: d.Get("description").(string),
			DisplayName: d.Get("display_name").(string),
			Group:       d.Get("group").(string),
			Hidden:      d.Get("hidden").(bool),
			Version:     int64(version),
			Permissions: permissions(d),
		}
		if err := client.UpdateRole(r); err != nil {
			return diag.FromErr(err)
		}
	}

	return ReadRole(ctx, d, meta)
}

func DeleteRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := ClientFromExistingOrgResource(meta, d.Id())
	g := d.Get("global").(bool)
	return diag.FromErr(client.DeleteRole(uid, g))
}

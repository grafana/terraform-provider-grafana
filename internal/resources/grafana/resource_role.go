package grafana

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func resourceRole() *common.Resource {
	schema := &schema.Resource{
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
							ValidateFunc: func(i interface{}, k string) (warnings []string, errors []error) {
								action := i.(string)
								if strings.HasPrefix(action, "grafana-oncall-app.") {
									warnings = append(warnings, "'grafana-oncall-app' permissions are deprecated. Permissions from 'grafana-oncall-app' should be migrated to the corresponding 'grafana-irm-app' permissions.")
								}
								return warnings, nil
							},
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

	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
		"grafana_role",
		orgResourceIDString("uid"),
		schema,
	)
}

func CreateRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	if d.Get("global").(bool) {
		orgID = 0
		client = client.WithOrgID(orgID)
	}

	var version int
	if d.Get("auto_increment_version").(bool) {
		version = 1
	} else {
		version = d.Get("version").(int)
	}

	role := models.CreateRoleForm{
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

	resp, err := client.AccessControl.CreateRole(&role)
	if err != nil {
		return diag.FromErr(err)
	}
	r := resp.Payload
	d.SetId(MakeOrgResourceID(orgID, r.UID))
	return ReadRole(ctx, d, meta)
}

func permissions(d *schema.ResourceData) []*models.Permission {
	p, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}

	perms := make([]*models.Permission, 0)
	for _, permission := range p.(*schema.Set).List() {
		p := permission.(map[string]interface{})
		perms = append(perms, &models.Permission{
			Action: p["action"].(string),
			Scope:  p["scope"].(string),
		})
	}

	return perms
}

func ReadRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	if d.Get("global").(bool) {
		var orgID int64 = 0
		client = client.WithOrgID(orgID)
	}
	return readRoleFromUID(client, uid, d)
}

func readRoleFromUID(client *goapi.GrafanaHTTPAPI, uid string, d *schema.ResourceData) diag.Diagnostics {
	resp, err := client.AccessControl.GetRole(uid)
	if err, shouldReturn := common.CheckReadError("role", d, err); shouldReturn {
		return err
	}
	r := resp.Payload

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
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	if d.Get("global").(bool) {
		var orgID int64 = 0
		client = client.WithOrgID(orgID)
	}

	if d.HasChange("version") || d.HasChange("name") || d.HasChange("description") || d.HasChange("permissions") ||
		d.HasChange("display_name") || d.HasChange("group") || d.HasChange("hidden") {
		version := d.Get("version").(int)
		if d.Get("auto_increment_version").(bool) {
			version += 1
		}

		description := d.Get("description").(string)
		displayName := d.Get("display_name").(string)
		group := d.Get("group").(string)

		r := models.UpdateRoleCommand{
			Name:        d.Get("name").(string),
			Global:      d.Get("global").(bool),
			Description: &description,
			DisplayName: &displayName,
			Group:       &group,
			Hidden:      d.Get("hidden").(bool),
			Version:     int64(version),
			Permissions: permissions(d),
		}
		if _, err := client.AccessControl.UpdateRole(uid, &r); err != nil {
			return diag.FromErr(err)
		}
	}

	return ReadRole(ctx, d, meta)
}

func DeleteRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	global := d.Get("global").(bool)
	if global {
		var orgID int64 = 0
		client = client.WithOrgID(orgID)
	}
	_, err := client.AccessControl.DeleteRole(access_control.NewDeleteRoleParams().WithRoleUID(uid).WithGlobal(&global), nil)
	diag, _ := common.CheckReadError("role", d, err)
	return diag
}

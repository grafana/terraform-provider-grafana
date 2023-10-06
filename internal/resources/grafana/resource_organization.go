package grafana

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type OrgUser struct {
	ID    int64
	Email string
	Role  string
}

type UserChange struct {
	Type ChangeType
	User OrgUser
}

type ChangeType int8

const (
	Add ChangeType = iota
	Update
	Remove
)

func ResourceOrganization() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/organization-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/org/)

This resource represents an instance-scoped resource and uses Grafana's admin APIs.
It does not work with API tokens or service accounts which are org-scoped.
You must use basic auth.
`,

		CreateContext: CreateOrganization,
		ReadContext:   ReadOrganization,
		UpdateContext: UpdateOrganization,
		DeleteContext: DeleteOrganization,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The display name for the Grafana organization created.",
			},
			"admin_user": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "admin",
				Description: `
The login name of the configured default admin user for the Grafana
installation. If unset, this value defaults to admin, the Grafana default.
Grafana adds the default admin user to all organizations automatically upon
creation, and this parameter keeps Terraform from removing it from
organizations.
`,
			},
			"create_users": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				Description: `
Whether or not to create Grafana users specified in the organization's
membership if they don't already exist in Grafana. If unspecified, this
parameter defaults to true, creating placeholder users with the name, login,
and email set to the email of the user, and a random password. Setting this
option to false will cause an error to be thrown for any users that do not
already exist in Grafana.
`,
			},
			"org_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The organization id assigned to this organization by Grafana.",
			},
			"admins": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: `
A list of email addresses corresponding to users who should be given admin
access to the organization. Note: users specified here must already exist in
Grafana unless 'create_users' is set to true.
`,
			},
			"editors": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: `
A list of email addresses corresponding to users who should be given editor
access to the organization. Note: users specified here must already exist in
Grafana unless 'create_users' is set to true.
`,
			},
			"viewers": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: `
A list of email addresses corresponding to users who should be given viewer
access to the organization. Note: users specified here must already exist in
Grafana unless 'create_users' is set to true.
`,
			},
			"nones": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: `
A list of email addresses corresponding to users who has no basic role assigned.
Note: users specified here must already exist in Grafana, unless 'create_users' is
set to true.
`,
			},
		},
	}
}

func CreateOrganization(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	name := d.Get("name").(string)
	orgID, err := client.NewOrg(name)
	if err != nil && err.Error() == "409 Conflict" {
		return diag.Errorf("Error: A Grafana Organization with the name '%s' already exists.", name)
	}
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(orgID, 10))
	if err = UpdateUsers(d, meta); err != nil {
		return diag.FromErr(err)
	}

	return ReadOrganization(ctx, d, meta)
}

func ReadOrganization(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	orgID, _ := strconv.ParseInt(d.Id(), 10, 64)
	resp, err := client.Org(orgID)
	if err, shouldReturn := common.CheckReadError("organization", d, err); shouldReturn {
		return err
	}
	d.Set("org_id", resp.ID)
	d.Set("name", resp.Name)
	if err := ReadUsers(d, meta); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func UpdateOrganization(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	orgID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if d.HasChange("name") {
		name := d.Get("name").(string)
		err := client.UpdateOrg(orgID, name)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if err := UpdateUsers(d, meta); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func DeleteOrganization(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	orgID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if err := client.DeleteOrg(orgID); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ReadUsers(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*common.Client).GrafanaAPI
	orgID, _ := strconv.ParseInt(d.Id(), 10, 64)
	orgUsers, err := client.OrgUsers(orgID)
	if err != nil {
		return err
	}
	roleMap := map[string][]string{"Admin": nil, "Editor": nil, "Viewer": nil, "None": nil}
	grafAdmin := d.Get("admin_user")
	for _, orgUser := range orgUsers {
		if orgUser.Login != grafAdmin {
			roleMap[orgUser.Role] = append(roleMap[orgUser.Role], orgUser.Email)
		}
	}
	for k, v := range roleMap {
		d.Set(fmt.Sprintf("%ss", strings.ToLower(k)), v)
	}
	return nil
}

func UpdateUsers(d *schema.ResourceData, meta interface{}) error {
	stateUsers, configUsers, err := collectUsers(d)
	if err != nil {
		return err
	}
	changes := changes(stateUsers, configUsers)
	orgID, _ := strconv.ParseInt(d.Id(), 10, 64)
	changes, err = addIdsToChanges(d, meta, changes)
	if err != nil {
		return err
	}
	return applyChanges(meta, orgID, changes)
}

func collectUsers(d *schema.ResourceData) (map[string]OrgUser, map[string]OrgUser, error) {
	roles := []string{"admins", "editors", "viewers", "nones"}
	stateUsers, configUsers := make(map[string]OrgUser), make(map[string]OrgUser)
	for _, role := range roles {
		caser := cases.Title(language.English)
		roleName := caser.String(role[:len(role)-1])
		// Get the lists of users read in from Grafana state (old) and configured (new)
		state, config := d.GetChange(role)
		for _, u := range state.(*schema.Set).List() {
			email := u.(string)
			// Sanity check that a user isn't specified twice within an organization
			if _, ok := stateUsers[email]; ok {
				return nil, nil, fmt.Errorf("error: User '%s' cannot be specified multiple times", email)
			}
			stateUsers[email] = OrgUser{0, email, roleName}
		}
		for _, u := range config.(*schema.Set).List() {
			email := u.(string)
			// Sanity check that a user isn't specified twice within an organization
			if _, ok := configUsers[email]; ok {
				return nil, nil, fmt.Errorf("error: User '%s' cannot be specified multiple times", email)
			}
			configUsers[email] = OrgUser{0, email, roleName}
		}
	}
	return stateUsers, configUsers, nil
}

func changes(stateUsers, configUsers map[string]OrgUser) []UserChange {
	var changes []UserChange
	for _, user := range configUsers {
		sUser, ok := stateUsers[user.Email]
		if !ok {
			// User doesn't exist in Grafana's state for the organization, should be added.
			changes = append(changes, UserChange{Add, user})
			continue
		}
		if sUser.Role != user.Role {
			// Update the user as they're configured with a different role than
			// what is in Grafana's state.
			changes = append(changes, UserChange{Update, user})
		}
	}
	for _, user := range stateUsers {
		if _, ok := configUsers[user.Email]; !ok {
			// User exists in Grafana's state for the organization, but isn't
			// present in the organization configuration, should be removed.
			changes = append(changes, UserChange{Remove, user})
		}
	}
	return changes
}

func addIdsToChanges(d *schema.ResourceData, meta interface{}, changes []UserChange) ([]UserChange, error) {
	client := meta.(*common.Client).GrafanaAPI
	gUserMap := make(map[string]int64)
	gUsers, err := client.Users()
	if err != nil {
		return nil, err
	}
	for _, u := range gUsers {
		gUserMap[u.Email] = u.ID
	}
	var output []UserChange
	create := d.Get("create_users").(bool)
	for _, change := range changes {
		id, ok := gUserMap[change.User.Email]
		if !ok && change.Type == Remove {
			log.Printf("[WARN] can't remove user %s from organization %s because it no longer exists in grafana", change.User.Email, d.Id())
			continue
		}
		if !ok && !create {
			return nil, fmt.Errorf("error adding user %s. User does not exist in Grafana", change.User.Email)
		}
		if !ok && create {
			id, err = createUser(meta, change.User.Email)
			if err != nil {
				return nil, err
			}
		}
		change.User.ID = id
		output = append(output, change)
	}
	return output, nil
}

func createUser(meta interface{}, user string) (int64, error) {
	client := meta.(*common.Client).GrafanaAPI
	id, n := int64(0), 64
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return id, err
	}
	pass := string(bytes[:n])
	u := gapi.User{
		Name:     user,
		Login:    user,
		Email:    user,
		Password: pass,
	}
	id, err = client.CreateUser(u)
	if err != nil {
		return id, err
	}
	return id, err
}

func applyChanges(meta interface{}, orgID int64, changes []UserChange) error {
	var err error
	client := meta.(*common.Client).GrafanaAPI
	for _, change := range changes {
		u := change.User
		switch change.Type {
		case Add:
			err = client.AddOrgUser(orgID, u.Email, u.Role)
		case Update:
			err = client.UpdateOrgUser(orgID, u.ID, u.Role)
		case Remove:
			err = client.RemoveOrgUser(orgID, u.ID)
		}
		if err != nil && !strings.HasPrefix(err.Error(), "status: 409") {
			return err
		}
	}
	return nil
}

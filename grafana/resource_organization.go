package grafana

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	gapi "github.com/nytm/go-grafana-api"
)

type OrgUser struct {
	Id    int64
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
		Create: CreateOrganization,
		Read:   ReadOrganization,
		Update: UpdateOrganization,
		Delete: DeleteOrganization,
		Exists: ExistsOrganization,
		Importer: &schema.ResourceImporter{
			State: ImportOrganization,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"admin_user": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "admin",
			},
			"create_users": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"org_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"admins": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"editors": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"viewers": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func CreateOrganization(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	name := d.Get("name").(string)
	orgId, err := client.NewOrg(name)
	if err != nil && err.Error() == "409 Conflict" {
		return errors.New(fmt.Sprintf("Error: A Grafana Organization with the name '%s' already exists.", name))
	}
	if err != nil {
		return err
	}
	d.SetId(strconv.FormatInt(orgId, 10))
	return UpdateUsers(d, meta)
}

func ReadOrganization(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	resp, err := client.Org(orgId)
	if err != nil && err.Error() == "404 Not Found" {
		log.Printf("[WARN] removing organization %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	d.Set("name", resp.Name)
	if err := ReadUsers(d, meta); err != nil {
		return err
	}
	return nil
}

func UpdateOrganization(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	if d.HasChange("name") {
		name := d.Get("name").(string)
		err := client.UpdateOrg(orgId, name)
		if err != nil {
			return err
		}
	}
	return UpdateUsers(d, meta)
}

func DeleteOrganization(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	return client.DeleteOrg(orgId)
}

func ExistsOrganization(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*gapi.Client)
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.Org(orgId)
	if err != nil && err.Error() == "404 Not Found" {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, err
}

func ImportOrganization(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	exists, err := ExistsOrganization(d, meta)
	if err != nil || !exists {
		return nil, errors.New(fmt.Sprintf("Error: Unable to import Grafana Organization: %s.", err))
	}
	d.Set("admin_user", "admin")
	d.Set("create_users", "true")
	err = ReadOrganization(d, meta)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func ReadUsers(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	orgUsers, err := client.OrgUsers(orgId)
	if err != nil {
		return err
	}
	roleMap := map[string][]string{"Admin": nil, "Editor": nil, "Viewer": nil}
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
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	changes, err = addIdsToChanges(d, meta, changes)
	if err != nil {
		return err
	}
	return applyChanges(meta, orgId, changes)
}

func collectUsers(d *schema.ResourceData) (map[string]OrgUser, map[string]OrgUser, error) {
	roles := []string{"admins", "editors", "viewers"}
	stateUsers, configUsers := make(map[string]OrgUser), make(map[string]OrgUser)
	for _, role := range roles {
		roleName := strings.Title(role[:len(role)-1])
		// Get the lists of users read in from Grafana state (old) and configured (new)
		state, config := d.GetChange(role)
		for _, u := range state.([]interface{}) {
			email := u.(string)
			// Sanity check that a user isn't specified twice within an organization
			if _, ok := stateUsers[email]; ok {
				return nil, nil, errors.New(fmt.Sprintf("Error: User '%s' cannot be specified multiple times.", email))
			}
			stateUsers[email] = OrgUser{0, email, roleName}
		}
		for _, u := range config.([]interface{}) {
			email := u.(string)
			// Sanity check that a user isn't specified twice within an organization
			if _, ok := configUsers[email]; ok {
				return nil, nil, errors.New(fmt.Sprintf("Error: User '%s' cannot be specified multiple times.", email))
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
	client := meta.(*gapi.Client)
	gUserMap := make(map[string]int64)
	gUsers, err := client.Users()
	if err != nil {
		return nil, err
	}
	for _, u := range gUsers {
		gUserMap[u.Email] = u.Id
	}
	var output []UserChange
	create := d.Get("create_users").(bool)
	for _, change := range changes {
		id, ok := gUserMap[change.User.Email]
		if !ok && !create {
			return nil, errors.New(fmt.Sprintf("Error adding user %s. User does not exist in Grafana.", change.User.Email))
		}
		if !ok && create {
			id, err = createUser(meta, change.User.Email)
			if err != nil {
				return nil, err
			}
		}
		change.User.Id = id
		output = append(output, change)
	}
	return output, nil
}

func createUser(meta interface{}, user string) (int64, error) {
	client := meta.(*gapi.Client)
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

func applyChanges(meta interface{}, orgId int64, changes []UserChange) error {
	var err error
	client := meta.(*gapi.Client)
	for _, change := range changes {
		u := change.User
		switch change.Type {
		case Add:
			err = client.AddOrgUser(orgId, u.Email, u.Role)
		case Update:
			err = client.UpdateOrgUser(orgId, u.Id, u.Role)
		case Remove:
			err = client.RemoveOrgUser(orgId, u.Id)
		}
		if err != nil && err.Error() != "409 Conflict" {
			return err
		}
	}
	return nil
}

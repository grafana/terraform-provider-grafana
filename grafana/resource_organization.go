package grafana

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	gapi "github.com/nytm/go-grafana-api"
	"log"
	"strconv"
	"strings"
)

type OrgUser struct {
	Id    int64
	Email string
	Role  string
}

const UserAdd = 10
const UserUpdate = 20
const UserRemove = 30

type UserChange struct {
	Type int8
	User OrgUser
}

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
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"admin_user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "admin",
			},
			"create_users": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"org_id": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"admins": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"editors": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"viewers": &schema.Schema{
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
		log.Printf("[DEBUG] creating Grafana organization %s", name)
		return err
	}
	d.SetId(strconv.FormatInt(orgId, 10))
	return UpdateUsers(d, meta)
}

func ReadOrganization(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	resp, err := client.Org(orgId)
	if err != nil {
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] removing organization %s from state because it no longer exists in grafana", d.Id())
			d.SetId("")
			return nil
		}
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
		oldName, newName := d.GetChange("name")
		log.Printf("[DEBUG] org name has been updated from %s to %s", oldName.(string), newName.(string))
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
		return nil, errors.New("Error Importing Grafana Organization")
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
	oldUsers, newUsers := collectUsers(d)
	changes := changes(oldUsers, newUsers)
	orgId, _ := strconv.ParseInt(d.Id(), 10, 64)
	changes, err := addIds(d, meta, changes)
	if err != nil {
		return err
	}
	return applyChanges(meta, orgId, changes)
}

func collectUsers(d *schema.ResourceData) (map[string]OrgUser, map[string]OrgUser) {
	roles := []string{"admins", "editors", "viewers"}
	oldUsers, newUsers := make(map[string]OrgUser), make(map[string]OrgUser)
	for _, role := range roles {
		roleName := strings.Title(role[:len(role)-1])
		old, new := d.GetChange(role)
		for _, u := range old.([]interface{}) {
			oldUsers[u.(string)] = OrgUser{0, u.(string), roleName}
		}
		for _, u := range new.([]interface{}) {
			newUsers[u.(string)] = OrgUser{0, u.(string), roleName}
		}
	}
	return oldUsers, newUsers
}

func changes(oldUsers, newUsers map[string]OrgUser) map[string]UserChange {
	changes := make(map[string]UserChange)
	for _, user := range newUsers {
		oUser, ok := oldUsers[user.Email]
		if !ok {
			changes[user.Email] = UserChange{UserAdd, user}
			continue
		}
		if oUser.Role != user.Role {
			changes[user.Email] = UserChange{UserUpdate, user}
		}
	}
	for _, user := range oldUsers {
		if _, ok := newUsers[user.Email]; !ok {
			changes[user.Email] = UserChange{UserRemove, user}
		}
	}
	return changes
}

func addIds(d *schema.ResourceData, meta interface{}, changes map[string]UserChange) (map[string]UserChange, error) {
	client := meta.(*gapi.Client)
	gUserMap := make(map[string]int64)
	gUsers, err := client.Users()
	if err != nil {
		return nil, err
	}
	for _, u := range gUsers {
		gUserMap[u.Email] = u.Id
	}
	output := make(map[string]UserChange)
	create := d.Get("create_users").(bool)
	for _, change := range changes {
		id, ok := gUserMap[change.User.Email]
		if !ok && !create {
			return nil, errors.New(fmt.Sprintf("Error adding user %s. User does not exist in Grafana.", change.User.Email))
		}
		if !ok && create {
			log.Printf("[DEBUG] Creating user '%s'. User is not known to Grafana.", change.User.Email)
			user, err := createUser(meta, change.User.Email)
			if err != nil {
				return nil, err
			}
			id = user
		}
		change.User.Id = id
		output[change.User.Email] = change
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
	log.Printf("[DEBUG] creating user %s with random password", user)
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

func applyChanges(meta interface{}, orgId int64, changes map[string]UserChange) error {
	var err error
	client := meta.(*gapi.Client)
	for _, change := range changes {
		u := change.User
		switch change.Type {
		case UserAdd:
			err = client.AddOrgUser(orgId, u.Email, u.Role)
		case UserUpdate:
			err = client.UpdateOrgUser(orgId, u.Id, u.Role)
		case UserRemove:
			err = client.RemoveOrgUser(orgId, u.Id)
		}
		if err != nil && err.Error() != "409 Conflict" {
			return err
		}
	}
	return nil
}

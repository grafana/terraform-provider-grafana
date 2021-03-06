package grafana

import (
	"errors"
	"fmt"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceUser() *schema.Resource {
	return &schema.Resource{
		Create: CreateUser,
		Read:   ReadUser,
		Update: UpdateUser,
		Delete: DeleteUser,
		Exists: ExistsUser,
		Importer: &schema.ResourceImporter{
			State: ImportUser,
		},
		Schema: map[string]*schema.Schema{
			"email": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"login": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"is_admin": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func CreateUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	user := gapi.User{
		Email:    d.Get("email").(string),
		Name:     d.Get("name").(string),
		Login:    d.Get("login").(string),
		Password: d.Get("password").(string),
	}
	id, err := client.CreateUser(user)
	if err != nil {
		return err
	}
	if d.HasChange("is_admin") {
		err = client.UpdateUserPermissions(id, d.Get("is_admin").(bool))
		if err != nil {
			return err
		}
	}
	d.SetId(strconv.FormatInt(id, 10))
	return ReadUser(d, meta)
}

func ReadUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}
	user, err := client.User(id)
	if err != nil {
		return err
	}
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("login", user.Login)
	d.Set("is_admin", user.IsAdmin)
	return nil
}

func UpdateUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}
	u := gapi.User{
		ID:    id,
		Email: d.Get("email").(string),
		Name:  d.Get("name").(string),
		Login: d.Get("login").(string),
	}
	err = client.UserUpdate(u)
	if err != nil {
		return err
	}
	if d.HasChange("password") {
		err = client.UpdateUserPassword(id, d.Get("password").(string))
		if err != nil {
			return err
		}
	}
	if d.HasChange("is_admin") {
		err = client.UpdateUserPermissions(id, d.Get("is_admin").(bool))
		if err != nil {
			return err
		}
	}
	return ReadUser(d, meta)
}

func DeleteUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}
	return client.DeleteUser(id)
}

func ExistsUser(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*gapi.Client)
	userId, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.User(userId)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ImportUser(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	exists, err := ExistsUser(d, meta)
	if err != nil || !exists {
		return nil, errors.New(fmt.Sprintf("Error: Unable to import Grafana User: %s.", err))
	}
	err = ReadUser(d, meta)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

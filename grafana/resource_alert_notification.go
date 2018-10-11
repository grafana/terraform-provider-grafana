package grafana

import (
	"fmt"
	"log"
	"strconv"

	gapi "github.com/goraxe/go-grafana-api"
	"github.com/hashicorp/terraform/helper/schema"
)

func ResourceAlertNotification() *schema.Resource {
	return &schema.Resource{
		Create: CreateAlertNotification,
		Update: UpdateAlertNotification,
		Delete: DeleteAlertNotification,
		Read:   ReadAlertNotification,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"is_default": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"settings": {
				Type:      schema.TypeMap,
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func CreateAlertNotification(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	alertNotification, err := makeAlertNotification(d)
	if err != nil {
		return err
	}

	id, err := client.NewAlertNotification(alertNotification)
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(id, 10))

	return ReadAlertNotification(d, meta)
}

func UpdateAlertNotification(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	alertNotification, err := makeAlertNotification(d)
	if err != nil {
		return err
	}

	return client.UpdateAlertNotification(alertNotification)
}

func ReadAlertNotification(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid id: %#v", idStr)
	}

	alertNotification, err := client.AlertNotification(id)
	if err != nil {
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] removing datasource %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("id", alertNotification.Id)
	d.Set("is_default", alertNotification.IsDefault)
	d.Set("name", alertNotification.Name)
	d.Set("type", alertNotification.Type)
	d.Set("settings", alertNotification.Settings)

	return nil
}

func DeleteAlertNotification(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid id: %#v", idStr)
	}

	return client.DeleteAlertNotification(id)
}

func makeAlertNotification(d *schema.ResourceData) (*gapi.AlertNotification, error) {
	idStr := d.Id()
	var id int64
	var err error
	if idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	return &gapi.AlertNotification{
		Id:        id,
		Name:      d.Get("name").(string),
		Type:      d.Get("type").(string),
		IsDefault: d.Get("is_default").(bool),
		Settings:  d.Get("settings").(interface{}),
	}, err
}

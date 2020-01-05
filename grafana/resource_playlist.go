package grafana

import (
	"errors"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	gapi "github.com/nytm/go-grafana-api"
)

func ResourcePlaylist() *schema.Resource {
	return &schema.Resource{
		Create: CreatePlaylist,
		Read:   ReadPlaylist,
		Update: UpdatePlaylist,
		Delete: DeletePlaylist,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"interval": { // #todo validation func
				Type:     schema.TypeString,
				Required: true,
			},

			"item": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "dashboard_by_id",
							ValidateFunc: validatePlaylistItemType,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"title": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func CreatePlaylist(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	playlist := gapi.Playlist{}
	playlist.Name = d.Get("name").(string)
	playlist.Interval = d.Get("interval").(string)

	items := d.Get("item").([]interface{})

	for i := 0; i < len(items); i++ {
		item := items[i].(map[string]interface{})

		playlist.Items = append(playlist.Items, gapi.PlaylistItem{
			Type:  item["type"].(string),
			Value: item["value"].(string),
			Order: i + 1,
			Title: item["title"].(string),
		})
	}

	id, err := client.NewPlaylist(playlist)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(id))

	return ReadPlaylist(d, meta)
}

func ReadPlaylist(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	playlist, err := client.Playlist(id)
	if err != nil {
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] removing playlist %s from state because it no longer exists in grafana", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.SetId(strconv.Itoa(playlist.Id))
	d.Set("name", playlist.Name)
	d.Set("interval", playlist.Interval)
	d.Set("item", flattenItem(playlist))

	return nil
}

func flattenItem(playlist *gapi.Playlist) []interface{} {
	var list []interface{}

	for i := 0; i < len(playlist.Items); i++ {
		m := make(map[string]interface{})
		m["type"] = playlist.Items[i].Type
		m["value"] = playlist.Items[i].Value
		m["title"] = playlist.Items[i].Title

		list = append(list, m)
	}

	return list
}

func UpdatePlaylist(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	playlist := gapi.Playlist{
		Id:       id,
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
	}

	items := d.Get("item").([]interface{})

	for i := 0; i < len(items); i++ {
		item := items[i].(map[string]interface{})

		playlist.Items = append(playlist.Items, gapi.PlaylistItem{
			Type:  item["type"].(string),
			Value: item["value"].(string),
			Order: i + 1,
			Title: item["title"].(string),
		})
	}

	err = client.UpdatePlaylist(playlist)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(id))

	return ReadPlaylist(d, meta)
}

func DeletePlaylist(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	return client.DeletePlaylist(id)
}

func validatePlaylistItemType(val interface{}, k string) ([]string, []error) {
	if val.(string) != "dashboard_by_id" && val.(string) != "dashboard_by_tag" {
		return nil, []error{errors.New(`Given playlist item type is not supported or unknown. Supported values: "dashboard_by_id", "dashboard_by_tag"`)}
	}

	return nil, nil
}

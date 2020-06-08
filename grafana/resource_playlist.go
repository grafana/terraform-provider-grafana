package grafana

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	gapi "github.com/nytm/go-grafana-api"
)

func ResourcePlaylist() *schema.Resource {
	return &schema.Resource{
		Create: resourcePlaylistCreate,
		Read:   resourcePlaylistRead,
		Update: resourcePlaylistUpdate,
		Delete: resourcePlaylistDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"interval": {
				Type:     schema.TypeString,
				Required: true,
			},
			"item": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"order": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"title": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"value": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourcePlaylistCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	playlist := gapi.Playlist{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	log.Printf("[DEBUG] Creating Playlist %s", playlist.Name)
	id, err := client.NewPlaylist(playlist)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(id))

	return resourcePlaylistRead(d, meta)
}

func resourcePlaylistRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Reading Playlist %s", d.Id())
	resp, err := client.Playlist(id)
	if err != nil {
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] removing playlist %s from state because it no longer exists in grafana", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", resp.Name)
	d.Set("interval", resp.Interval)
	if err := d.Set("item", flattenPlaylistItems(resp.Items)); err != nil {
		return fmt.Errorf("unable to set item: %s", err)
	}
	log.Printf("[INFO] SETTING ITEMS AS %v", d.Get("item").(*schema.Set))
	return nil

}

func resourcePlaylistUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	playlist := gapi.Playlist{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	log.Printf("[DEBUG] Updating Playlist %s", playlist.Name)
	err := client.UpdatePlaylist(playlist)
	if err != nil {
		return err
	}

	return resourcePlaylistRead(d, meta)
}

func resourcePlaylistDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Deleting Playlist %s", d.Id())
	err = client.DeletePlaylist(id)
	return err
}

func expandPlaylistItems(items []interface{}) []gapi.PlaylistItem {
	playlistItems := make([]gapi.PlaylistItem, 0)
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		p := gapi.PlaylistItem{
			Order: itemMap["order"].(int),
			Title: itemMap["title"].(string),
		}
		if v, ok := itemMap["type"].(string); ok {
			p.Type = v
		}
		if v, ok := itemMap["value"].(string); ok {
			p.Value = v
		}
		playlistItems = append(playlistItems, p)
	}
	return playlistItems
}

func flattenPlaylistItems(items []gapi.PlaylistItem) []interface{} {
	playlistItems := make([]interface{}, 0)
	for _, item := range items {
		p := map[string]interface{}{
			"type":  item.Type,
			"value": item.Value,
			"order": item.Order,
			"title": item.Title,
		}
		playlistItems = append(playlistItems, p)
	}
	return playlistItems
}

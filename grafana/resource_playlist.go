package grafana

import (
	"context"
	"log"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourcePlaylist() *schema.Resource {
	return &schema.Resource{
		CreateContext: CreatePlaylist,
		ReadContext:   ReadPlaylist,
		UpdateContext: UpdatePlaylist,
		DeleteContext: DeletePlaylist,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/playlist/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/playlist/)
`,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the playlist.",
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
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
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
			"org_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreatePlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	playlist := gapi.Playlist{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	id, err := client.NewPlaylist(playlist)

	if err != nil {
		return diag.Errorf("error creating Playlist: %v", err)
	}

	d.SetId(id)

	return ReadPlaylist(ctx, d, meta)
}

func ReadPlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	resp, err := client.Playlist(d.Id())

	// In Grafana 9.0+, if the playlist doesn't exist, the API returns an empty playlist but not a 404
	if (err != nil && strings.HasPrefix(err.Error(), "status: 404")) || (resp.ID == 0 && resp.UID == "") {
		log.Printf("[WARN] removing playlist %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	} else if err != nil {
		return diag.Errorf("error reading Playlist (%s): %v", d.Id(), err)
	}

	d.Set("name", resp.Name)
	d.Set("interval", resp.Interval)
	if err := d.Set("item", flattenPlaylistItems(resp.Items)); err != nil {
		return diag.Errorf("error setting item: %v", err)
	}

	return nil
}

func UpdatePlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	playlist := gapi.Playlist{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	// Support both Grafana 9.0+ and older versions (UID is used in 9.0+)
	if idInt, err := strconv.Atoi(d.Id()); err == nil {
		playlist.ID = idInt
	} else {
		playlist.UID = d.Id()
	}

	err := client.UpdatePlaylist(playlist)
	if err != nil {
		return diag.Errorf("error updating Playlist (%s): %v", d.Id(), err)
	}

	return ReadPlaylist(ctx, d, meta)
}

func DeletePlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	if err := client.DeletePlaylist(d.Id()); err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			return nil
		}
		return diag.Errorf("error deleting Playlist (%s): %v", d.Id(), err)
	}

	return nil
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

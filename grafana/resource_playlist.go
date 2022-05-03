package grafana

import (
	"context"
	"fmt"
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
		return diag.FromErr(fmt.Errorf("error creating Playlist: %w", err))
	}

	d.SetId(strconv.Itoa(id))

	return ReadPlaylist(ctx, d, meta)
}

func ReadPlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading Playlist (%s): %w", d.Id(), err))
	}

	resp, err := client.Playlist(id)

	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing playlist %d from state because it no longer exists in grafana", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("error reading Playlist (%s): %w", d.Id(), err))
	}

	d.Set("name", resp.Name)
	d.Set("interval", resp.Interval)
	if err := d.Set("item", flattenPlaylistItems(resp.Items)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting item: %v", err))
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

	err := client.UpdatePlaylist(playlist)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating Playlist (%s): %w", d.Id(), err))
	}

	return ReadPlaylist(ctx, d, meta)
}

func DeletePlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	id, err := strconv.Atoi(d.Id())

	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting Playlist (%s): %w", d.Id(), err))
	}

	if err := client.DeletePlaylist(id); err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			return nil
		}
		return diag.FromErr(fmt.Errorf("error deleting Playlist (%s): %w", d.Id(), err))
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

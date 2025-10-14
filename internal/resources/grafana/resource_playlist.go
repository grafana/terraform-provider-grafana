package grafana

import (
	"context"
	"errors"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
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
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/create-manage-playlists/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/playlist/)
`,

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
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

func CreatePlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromNewOrgResource(meta, d)

	playlist := gapi.Playlist{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	id, err := client.NewPlaylist(playlist)

	if err != nil {
		return diag.Errorf("error creating Playlist: %v", err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))

	return ReadPlaylist(ctx, d, meta)
}

func ReadPlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, id := ClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Playlist(id)
	// In Grafana 9.0+, if the playlist doesn't exist, the API returns an empty playlist but not a notfound error
	if resp != nil && resp.ID == 0 && resp.UID == "" {
		err = errors.New(common.NotFoundError)
	}
	if err, shouldReturn := common.CheckReadError("playlist", d, err); shouldReturn {
		return err
	}

	d.SetId(MakeOrgResourceID(orgID, id))
	d.Set("name", resp.Name)
	d.Set("interval", resp.Interval)
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	if err := d.Set("item", flattenPlaylistItems(resp.Items)); err != nil {
		return diag.Errorf("error setting item: %v", err)
	}

	return nil
}

func UpdatePlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, id := ClientFromExistingOrgResource(meta, d.Id())

	playlist := gapi.Playlist{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	// Support both Grafana 9.0+ and older versions (UID is used in 9.0+)
	if idInt, err := strconv.Atoi(id); err == nil {
		playlist.ID = idInt
	} else {
		playlist.UID = id
	}

	err := client.UpdatePlaylist(playlist)
	if err != nil {
		return diag.Errorf("error updating Playlist (%s): %v", id, err)
	}

	return ReadPlaylist(ctx, d, meta)
}

func DeletePlaylist(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, id := ClientFromExistingOrgResource(meta, d.Id())
	err := client.DeletePlaylist(id)
	diag, _ := common.CheckReadError("playlist", d, err)
	return diag
}

func expandPlaylistItems(items []interface{}) []gapi.PlaylistItem {
	playlistItems := make([]gapi.PlaylistItem, 0)
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		p := gapi.PlaylistItem{
			Order: itemMap["order"].(int),
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
	for i, item := range items {
		if item.Order == 0 {
			item.Order = i + 1
		}
		p := map[string]interface{}{
			"type":  item.Type,
			"value": item.Value,
			"order": item.Order,
		}
		playlistItems = append(playlistItems, p)
	}
	return playlistItems
}

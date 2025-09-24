package grafana

import (
	"context"
	"errors"
	"sort"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/playlists"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePlaylist() *common.Resource {
	schema := &schema.Resource{
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

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_playlist",
		orgResourceIDString("uid"),
		schema,
	).
		WithLister(listerFunctionOrgResource(listPlaylists)).
		WithPreferredResourceNameField("name")
}

func listPlaylists(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	resp, err := client.Playlists.SearchPlaylists(playlists.NewSearchPlaylistsParams())
	if err != nil {
		return nil, err
	}

	for _, playlist := range resp.Payload {
		ids = append(ids, MakeOrgResourceID(orgID, playlist.UID))
	}

	return ids, nil
}

func CreatePlaylist(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	playlist := models.CreatePlaylistCommand{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	resp, err := client.Playlists.CreatePlaylist(&playlist)

	if err != nil {
		return diag.Errorf("error creating Playlist: %v", err)
	}

	id := resp.Payload.UID
	if id == "" {
		id = strconv.FormatInt(resp.Payload.ID, 10)
	}
	d.SetId(MakeOrgResourceID(orgID, id))

	return ReadPlaylist(ctx, d, meta)
}

func ReadPlaylist(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID, id := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Playlists.GetPlaylist(id)
	// In Grafana 9.0+, if the playlist doesn't exist, the API returns an empty playlist but not a notfound error
	if resp != nil && resp.GetPayload().ID == 0 && resp.GetPayload().UID == "" {
		err = errors.New(common.NotFoundError)
	}
	if err, shouldReturn := common.CheckReadError("playlist", d, err); shouldReturn {
		return err
	}

	playlist := resp.Payload
	itemsResp, err := client.Playlists.GetPlaylistItems(id)
	if err != nil {
		return diag.Errorf("error getting playlist items: %v", err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))
	d.Set("name", playlist.Name)
	d.Set("interval", playlist.Interval)
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	if err := d.Set("item", flattenPlaylistItems(itemsResp.Payload)); err != nil {
		return diag.Errorf("error setting item: %v", err)
	}

	return nil
}

func UpdatePlaylist(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, id := OAPIClientFromExistingOrgResource(meta, d.Id())

	playlist := models.UpdatePlaylistCommand{
		Name:     d.Get("name").(string),
		Interval: d.Get("interval").(string),
		Items:    expandPlaylistItems(d.Get("item").(*schema.Set).List()),
	}

	_, err := client.Playlists.UpdatePlaylist(id, &playlist)
	if err != nil {
		return diag.Errorf("error updating Playlist (%s): %v", id, err)
	}

	return ReadPlaylist(ctx, d, meta)
}

func DeletePlaylist(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, id := OAPIClientFromExistingOrgResource(meta, d.Id())
	_, err := client.Playlists.DeletePlaylist(id)
	diag, _ := common.CheckReadError("playlist", d, err)
	return diag
}

func expandPlaylistItems(items []any) []*models.PlaylistItem {
	playlistItems := make([]*models.PlaylistItem, 0)
	for _, item := range items {
		itemMap := item.(map[string]any)
		p := &models.PlaylistItem{
			Order: int64(itemMap["order"].(int)),
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
	sort.Slice(playlistItems, func(i, j int) bool {
		return playlistItems[i].Order < playlistItems[j].Order
	})
	return playlistItems
}

func flattenPlaylistItems(items []*models.PlaylistItem) []any {
	playlistItems := make([]any, 0)
	for i, item := range items {
		if item.Order == 0 {
			item.Order = int64(i + 1)
		}
		p := map[string]any{
			"type":  item.Type,
			"value": item.Value,
			"order": item.Order,
			"title": item.Title,
		}
		playlistItems = append(playlistItems, p)
	}
	return playlistItems
}

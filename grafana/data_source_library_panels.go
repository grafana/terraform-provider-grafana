package grafana

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceLibraryPanels() *schema.Resource {
	return &schema.Resource{
		Description: `
Datasource for retrieving all library panels.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/)
`,
		ReadContext: dataSourceLibraryPanelsRead,
		Schema: map[string]*schema.Schema{
			"folder_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Folder IDs to search in for library panels.",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"panels": {
				Description: "List of library panels found.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of this library panel.",
							Computed:    true,
						},
						"uid": {
							Type:        schema.TypeString,
							Description: "The unique identifier (UID) of this library panel.",
							Computed:    true,
						},
						"folder_id": {
							Type:        schema.TypeInt,
							Description: "The ID of the folder this library was found in.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceLibraryPanelsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	folderIDs := d.Get("folder_ids").([]interface{})

	results, err := client.LibraryPanels()
	if err != nil {
		diag.FromErr(err)
	}

	params := url.Values{
		"library_panels": {"all"},
	}
	folderIDsMap := make(map[string]bool)
	for f := range folderIDs {
		params.Add("folder_ids", fmt.Sprint(f))
		folderIDsMap[fmt.Sprint(f)] = true
	}

	panels := []map[string]interface{}{}
	for _, panel := range results {
		// filter panel by folder, if specified
		isWanted := true
		if len(folderIDs) > 0 {
			if _, ok := folderIDsMap[fmt.Sprint(panel.Folder)]; !ok {
				isWanted = false
			}
		}

		if isWanted {
			panels = append(panels, map[string]interface{}{
				"folder_id": panel.Folder,
				"name":      panel.Name,
				"uid":       panel.UID,
			})
		}
	}

	d.SetId(hashDashboardSearchParameters(params))
	if err := d.Set("panels", panels); err != nil {
		diag.Errorf("failed to set 'panels' attribute: %s", err)
	}

	return nil
}

func hashDashboardSearchParameters(params map[string][]string) string {
	// hash a sorted slice of all string parameters and corresponding values
	hashOut := sha256.New()

	var paramsList []string
	for key, vals := range params {
		paramsList = append(paramsList, key)
		paramsList = append(paramsList, vals...)
	}

	sort.Strings(paramsList)
	hashIn := strings.Join(paramsList[:], "")
	hashOut.Write([]byte(hashIn))
	return fmt.Sprintf("%x", hashOut.Sum(nil))[0:23]
}

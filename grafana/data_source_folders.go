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

func DatasourceFolders() *schema.Resource {
	return &schema.Resource{
		Description: `
Datasource for retrieving all Grafana folders.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/dashboard_folders/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/)
`,
		ReadContext: dataSourceReadFolders,
		Schema: map[string]*schema.Schema{
			"limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1000,
				Description: "Maximum number of folders to return.",
			},
			"folders": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"uid": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
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
	hashIn := strings.Join(paramsList, "")
	hashOut.Write([]byte(hashIn))
	return fmt.Sprintf("%x", hashOut.Sum(nil))[0:23]
}

func dataSourceReadFolders(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	var diags diag.Diagnostics
	params := url.Values{
		"limit": {fmt.Sprint(d.Get("limit"))},
		"type":  {"folder-db"},
	}

	d.SetId(hashDashboardSearchParameters(params))

	results, err := client.FolderDashboardSearch(params)
	if err != nil {
		return diag.FromErr(err)
	}

	folders := make([]map[string]interface{}, len(results))
	for i, result := range results {
		folders[i] = map[string]interface{}{
			"title": result.Title,
			"uid":   result.UID,
			"id":    result.ID,
		}
	}

	if err := d.Set("folders", folders); err != nil {
		return diag.Errorf("error setting folders attribute: %s", err)
	}

	return diags
}

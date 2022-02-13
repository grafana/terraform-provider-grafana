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

func DatasourceDashboards() *schema.Resource {
	return &schema.Resource{
		Description: `
Datasource for retrieving all dashboards. Specify list of folder IDs to search in for dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/)
`,
		ReadContext: dataSourceReadDashboards,
		Schema: map[string]*schema.Schema{
			"folder_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Numerical IDs of Grafana folders containing dashboards. Specify to filter for dashboards by folder (eg. `[0]` for General folder), or leave blank to get all dashboards in all folders.",
				Elem:        &schema.Schema{Type: schema.TypeInt},
			},
			"limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     5000,
				Description: "Maximum number of dashboard search results to return.",
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of string Grafana dashboard tags to search for, eg. `[\"prod\"]`. Used only as search input, i.e., attribute value will remain unchanged.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"dashboards": {
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
						"folder_title": {
							Type:     schema.TypeString,
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

func dataSourceReadDashboards(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	var diags diag.Diagnostics
	params := url.Values{
		"limit": {fmt.Sprint(d.Get("limit"))},
		"type":  {"dash-db"},
	}

	// add tags and folder IDs from attributes to dashboard search parameters
	if list, ok := d.GetOk("folder_ids"); ok {
		for _, elem := range list.([]interface{}) {
			params.Add("folderIds", fmt.Sprint(elem))
		}
	}

	if list, ok := d.GetOk("tags"); ok {
		for _, elem := range list.([]interface{}) {
			params.Add("tag", fmt.Sprint(elem))
		}
	}

	d.SetId(hashDashboardSearchParameters(params))

	results, err := client.FolderDashboardSearch(params)
	if err != nil {
		return diag.FromErr(err)
	}

	dashboards := make([]map[string]interface{}, len(results))
	for i, result := range results {
		dashboards[i] = map[string]interface{}{
			"title":        result.Title,
			"uid":          result.UID,
			"folder_title": result.FolderTitle,
		}
	}

	if err := d.Set("dashboards", dashboards); err != nil {
		return diag.Errorf("error setting dashboards attribute: %s", err)
	}

	return diags
}

package dashboardhelpers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type TablePanelData struct {
	Type       string                   `json:"type"`
	Title      string                   `json:"title"`
	Datasource Datasource               `json:"datasource"`
	Targets    []map[string]interface{} `json:"targets"`
}

type Datasource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

func DatasourceTablePanel() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,
		ReadContext: dataSourceTablePanelRead,
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Title of the table panel",
			},
			// TODO: maybe not needed, and can be looked up by UID
			"datasource_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Type of the datasource.",
			},
			"datasource_uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "UID of the datasource.",
			},
			"target_json": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Targets JSON for this panel.",
			},
			"config_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Computed JSON config for this panel.",
			},
		},
	}
}

func dataSourceTablePanelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	target := []map[string]interface{}{}

	err := json.Unmarshal([]byte(d.Get("target_json").(string)), &target)
	if err != nil {
		return diag.FromErr(err)
	}
	data := TablePanelData{
		Type:    "table",
		Title:   d.Get("title").(string),
		Targets: target,
		Datasource: Datasource{
			Type: d.Get("datasource_type").(string),
			UID:  d.Get("datasource_uid").(string),
		},
	}

	JSONConfig, err := json.Marshal(data)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	d.Set("config_json", string(JSONConfig))

	return nil
}

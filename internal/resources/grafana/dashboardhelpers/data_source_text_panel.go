package dashboardhelpers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type TextPanelData struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	Mode        string `json:"mode"`
	Transparent bool   `json:"transparent"`
}

func DatasourceTextPanel() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,
		ReadContext: dataSourceTextPanelRead,
		Schema: map[string]*schema.Schema{
			"content": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Content of the text panel",
			},
			"mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "markdown",
				Description: "Mode of the text panel: 'html' or 'markdown'.",
			},
			"transparent": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Panel background transparency.",
			},
			"config_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Computed JSON config for this panel.",
			},
		},
	}
}

func dataSourceTextPanelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data := TextPanelData{
		Type:        "text",
		Content:     d.Get("content").(string),
		Mode:        d.Get("mode").(string),
		Transparent: d.Get("transparent").(bool),
	}

	JSONConfig, err := json.Marshal(data)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	d.Set("config_json", string(JSONConfig))

	return nil
}

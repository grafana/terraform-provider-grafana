package grafana

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// var (
// 	StoreDashboardSHA256 bool
// )

func ResourcePanel() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages Grafana panels.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,

		CreateContext: CreatePanel,
		ReadContext:   ReadPanel,
		UpdateContext: UpdatePanel,
		DeleteContext: DeletePanel,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// TODO: this should be replaces with a schema of the panel, that probably should be serialized and put into config_json
			"temp_json": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The panel JSON.",
			},
			"config_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The panel JSON.",
			},
		},
	}
}

func CreatePanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: this should be replaces with a schema of the panel, that probably should be serialized and put into config_json
	tmp := d.Get("temp_json")
	err := d.Set("config_json", tmp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

func ReadPanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func UpdatePanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return ReadDashboard(ctx, d, meta)
}

func DeletePanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}

// UnmarshalPanelConfigJSON is a convenience func for unmarshalling
// `config_json` field.
func UnmarshalPanelConfigJSON(configJSON string) (map[string]interface{}, error) {
	panelJSON := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &panelJSON)
	if err != nil {
		return nil, err
	}
	return panelJSON, nil
}

func NormalizePanelConfigJSON(config interface{}) string {
	var panelJSON map[string]interface{}
	switch c := config.(type) {
	case map[string]interface{}:
		panelJSON = c
	case string:
		var err error
		panelJSON, err = UnmarshalPanelConfigJSON(c)
		if err != nil {
			return c
		}
	}

	delete(panelJSON, "id")

	j, _ := json.Marshal(panelJSON)

	return string(j)
}

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
			"config_json": {
				Type:      schema.TypeString,
				Required:  true,
				StateFunc: NormalizeDashboardConfigJSON,
				// ValidateFunc: validateDashboardConfigJSON,
				Description: "The panel JSON.",
			},
		},
	}
}

func CreatePanel(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return ReadDashboard(ctx, d, meta)
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

package grafana

import (
	"context"
	"fmt"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourcePublicDashboard() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages Grafana public dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/dashboard-public/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard_public/)
`,

		CreateContext: CreatePublicDashboard,
		ReadContext:   ReadPublicDashboard,
		UpdateContext: UpdatePublicDashboard,
		DeleteContext: DeletePublicDashboard,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				Description: "The unique identifier of a public dashboard. " +
					"It's automatically generated if not provided when creating a public dashboard. ",
			},
			"dashboard_uid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique identifier of the original dashboard.",
			},
			"access_token": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				Description: "A public unique identifier of a public dashboard. This is used to construct its URL. " +
					"It's automatically generated if not provided when creating a public dashboard. ",
			},
			"time_selection_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to `true` to enable the time picker in the public dashboard. The default value is `false`.",
			},
			"is_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to `true` to enable the public dashboard. The default value is `false`.",
			},
			"annotations_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to `true` to show annotations. The default value is `false`.",
			},
			"share": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Set the share mode. The default value is `public`.",
			},
		},
	}
}

func CreatePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	dashboardUID := d.Get("dashboard_uid").(string)

	publicDashboard := makePublicDashboard(d)
	resp, err := client.NewPublicDashboard(dashboardUID, publicDashboard)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s:%s", resp.DashboardUID, resp.UID))
	return ReadPublicDashboard(ctx, d, meta)
}
func UpdatePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	dashboardUID, publicDashboardUID := SplitPublicDashboardID(d.Id())

	publicDashboard := makePublicDashboard(d)
	dashboard, err := client.UpdatePublicDashboard(dashboardUID, publicDashboardUID, publicDashboard)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s:%s", dashboard.DashboardUID, dashboard.UID))
	return ReadPublicDashboard(ctx, d, meta)
}

func DeletePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	dashboardUID, publicDashboardUID := SplitPublicDashboardID(d.Id())
	return diag.FromErr(client.DeletePublicDashboard(dashboardUID, publicDashboardUID))
}

func makePublicDashboard(d *schema.ResourceData) gapi.PublicDashboardPayload {
	return gapi.PublicDashboardPayload{
		UID:                  d.Get("uid").(string),
		AccessToken:          d.Get("access_token").(string),
		TimeSelectionEnabled: d.Get("time_selection_enabled").(bool),
		IsEnabled:            d.Get("is_enabled").(bool),
		AnnotationsEnabled:   d.Get("annotations_enabled").(bool),
		Share:                d.Get("share").(string),
	}
}

func ReadPublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	dashboardUID, _ := SplitPublicDashboardID(d.Id())
	dashboard, err := client.PublicDashboardbyUID(dashboardUID)
	if err, shouldReturn := common.CheckReadError("dashboard", d, err); shouldReturn {
		return err
	}

	d.SetId(fmt.Sprintf("%s:%s", dashboard.DashboardUID, dashboard.UID))

	d.Set("uid", dashboard.UID)
	d.Set("dashboard_uid", dashboard.DashboardUID)
	d.Set("access_token", dashboard.AccessToken)
	d.Set("time_selection_enabled", dashboard.TimeSelectionEnabled)
	d.Set("is_enabled", dashboard.IsEnabled)
	d.Set("annotations_enabled", dashboard.AnnotationsEnabled)
	d.Set("share", dashboard.Share)

	return nil
}

func SplitPublicDashboardID(id string) (string, string) {
	d, pd, _ := strings.Cut(id, ":")
	return d, pd
}

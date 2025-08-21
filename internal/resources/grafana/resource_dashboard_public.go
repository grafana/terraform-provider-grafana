package grafana

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/client/dashboard_public"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var resourcePublicDashboardID = common.NewResourceID(
	common.OptionalIntIDField("orgID"),
	common.StringIDField("dashboardUID"),
	common.StringIDField("publicDashboardUID"),
)

func resourcePublicDashboard() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages Grafana public dashboards.

**Note:** This resource is available only with Grafana 10.2+.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/share-dashboards-panels/shared-dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/next/developers/http_api/dashboard_public/)
`,

		CreateContext: CreatePublicDashboard,
		ReadContext:   ReadPublicDashboard,
		UpdateContext: UpdatePublicDashboard,
		DeleteContext: DeletePublicDashboard,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
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

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_dashboard_public",
		resourcePublicDashboardID,
		schema,
	)
}

func CreatePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	dashboardUID := d.Get("dashboard_uid").(string)

	publicDashboardPayload := makePublicDashboard(d)
	resp, err := client.DashboardPublic.CreatePublicDashboard(dashboardUID, publicDashboardPayload)
	if err != nil {
		return diag.FromErr(err)
	}
	pd := resp.Payload

	d.SetId(resourcePublicDashboardID.Make(orgID, pd.DashboardUID, pd.UID))
	return ReadPublicDashboard(ctx, d, meta)
}
func UpdatePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, compositeID := OAPIClientFromExistingOrgResource(meta, d.Id())
	dashboardUID, publicDashboardUID, _ := strings.Cut(compositeID, ":")

	publicDashboard := makePublicDashboard(d)
	params := dashboard_public.NewUpdatePublicDashboardParams().
		WithDashboardUID(dashboardUID).
		WithUID(publicDashboardUID).
		WithBody(publicDashboard)
	resp, err := client.DashboardPublic.UpdatePublicDashboard(params)
	if err != nil {
		return diag.FromErr(err)
	}
	pd := resp.Payload

	d.SetId(fmt.Sprintf("%d:%s:%s", orgID, pd.DashboardUID, pd.UID))
	return ReadPublicDashboard(ctx, d, meta)
}

func DeletePublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, compositeID := OAPIClientFromExistingOrgResource(meta, d.Id())
	dashboardUID, publicDashboardUID, _ := strings.Cut(compositeID, ":")
	_, err := client.DashboardPublic.DeletePublicDashboard(publicDashboardUID, dashboardUID)

	return diag.FromErr(err)
}

func makePublicDashboard(d *schema.ResourceData) *models.PublicDashboardDTO {
	return &models.PublicDashboardDTO{
		UID:                  d.Get("uid").(string),
		AccessToken:          d.Get("access_token").(string),
		TimeSelectionEnabled: d.Get("time_selection_enabled").(bool),
		IsEnabled:            d.Get("is_enabled").(bool),
		AnnotationsEnabled:   d.Get("annotations_enabled").(bool),
		Share:                models.ShareType(d.Get("share").(string)),
	}
}

func ReadPublicDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, compositeID := OAPIClientFromExistingOrgResource(meta, d.Id())
	dashboardUID, _, _ := strings.Cut(compositeID, ":")

	resp, err := client.DashboardPublic.GetPublicDashboard(dashboardUID)
	if err, shouldReturn := common.CheckReadError("dashboard", d, err); shouldReturn {
		return err
	}
	pd := resp.Payload

	d.Set("org_id", strconv.FormatInt(orgID, 10))

	d.Set("uid", pd.UID)
	d.Set("dashboard_uid", pd.DashboardUID)
	d.Set("access_token", pd.AccessToken)
	d.Set("time_selection_enabled", pd.TimeSelectionEnabled)
	d.Set("is_enabled", pd.IsEnabled)
	d.Set("annotations_enabled", pd.AnnotationsEnabled)
	d.Set("share", pd.Share)

	d.SetId(fmt.Sprintf("%d:%s:%s", orgID, pd.DashboardUID, pd.UID))

	return nil
}

package grafana

import (
	"context"
	"strconv"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/annotations"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAnnotation() *common.Resource {
	schema := &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/annotate-visualizations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/annotations/)
`,

		CreateContext: CreateAnnotation,
		UpdateContext: UpdateAnnotation,
		DeleteContext: DeleteAnnotation,
		ReadContext:   ReadAnnotation,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"text": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The text to associate with the annotation.",
			},

			"time": {
				Description:  "The RFC 3339-formatted time string indicating the annotation's time.",
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},

			"time_end": {
				Description:  "The RFC 3339-formatted time string indicating the annotation's end time.",
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},

			"dashboard_uid": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The UID of the dashboard on which to create the annotation.",
			},

			"panel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "The ID of the dashboard panel on which to create the annotation.",
			},

			"tags": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The tags to associate with the annotation.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_annotation",
		orgResourceIDInt("id"),
		schema,
	).WithLister(listerFunctionOrgResource(listAnnotations))
}

func listAnnotations(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	resp, err := client.Annotations.GetAnnotations(annotations.NewGetAnnotationsParams())
	if err != nil {
		return nil, err
	}

	for _, annotation := range resp.Payload {
		ids = append(ids, MakeOrgResourceID(orgID, annotation.ID))
	}

	return ids, nil
}

func CreateAnnotation(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	annotation, err := makeAnnotation(d)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.Annotations.PostAnnotation(annotation)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, *resp.GetPayload().ID))

	return ReadAnnotation(ctx, d, meta)
}

func UpdateAnnotation(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	postAnnotation, err := makeAnnotation(d)
	if err != nil {
		return diag.FromErr(err)
	}
	// Convert to update payload
	annotation := models.UpdateAnnotationsCmd{
		Tags:    postAnnotation.Tags,
		Text:    *postAnnotation.Text,
		Time:    postAnnotation.Time,
		TimeEnd: postAnnotation.TimeEnd,
	}

	_, err = client.Annotations.UpdateAnnotation(idStr, &annotation)
	return diag.FromErr(err)
}

func ReadAnnotation(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Annotations.GetAnnotationByID(idStr)
	if err, shouldReturn := common.CheckReadError("Annotation", d, err); shouldReturn {
		return err
	}
	annotation := resp.GetPayload()

	if annotation.DashboardID > 0 && annotation.DashboardUID == "" {
		// Have to list annotations here because the dashboard_uid is not fetched when using GetAnnotationByID
		// Also, the GetDashboardByID API is deprecated and removed.
		// TODO: Fix the API. The dashboard UID is not returned in the response.
		listParams := annotations.NewGetAnnotationsParams().
			WithDashboardID(&annotation.DashboardID).
			WithFrom(&annotation.Time).
			WithTo(&annotation.TimeEnd)

		listResp, err := client.Annotations.GetAnnotations(listParams)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, a := range listResp.Payload {
			if strconv.FormatInt(a.ID, 10) == idStr {
				annotation.DashboardUID = a.DashboardUID
				break
			}
		}
	}

	t := time.UnixMilli(annotation.Time)
	tEnd := time.UnixMilli(annotation.TimeEnd)

	// Convert tags to set; use null when empty so state matches plan (optional attribute unset).
	if len(annotation.Tags) == 0 {
		d.Set("tags", nil)
	} else {
		d.Set("tags", annotation.Tags)
	}

	if annotation.DashboardUID != "" {
		d.Set("dashboard_uid", annotation.DashboardUID)
	} else {
		d.Set("dashboard_uid", nil)
	}
	if annotation.PanelID != 0 {
		d.Set("panel_id", annotation.PanelID)
	} else {
		d.Set("panel_id", nil)
	}

	d.Set("text", annotation.Text)
	d.Set("time", t.Format(time.RFC3339))
	d.Set("time_end", tEnd.Format(time.RFC3339))
	d.Set("org_id", strconv.FormatInt(orgID, 10))

	return nil
}

func DeleteAnnotation(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	_, err := client.Annotations.DeleteAnnotationByID(idStr)
	diag, _ := common.CheckReadError("annotation", d, err)
	return diag
}

func makeAnnotation(d *schema.ResourceData) (*models.PostAnnotationsCmd, error) {
	var err error

	text := d.Get("text").(string)
	a := &models.PostAnnotationsCmd{
		Text:         &text,
		PanelID:      int64(d.Get("panel_id").(int)),
		DashboardUID: d.Get("dashboard_uid").(string),
		Tags:         common.SetToStringSlice(d.Get("tags").(*schema.Set)),
	}

	start := d.Get("time").(string)
	if start != "" {
		t, err := millisSinceEpoch(start)
		if err != nil {
			return a, err
		}
		a.Time = t
	}

	timeEnd := d.Get("time_end").(string)
	if timeEnd != "" {
		tEnd, err := millisSinceEpoch(timeEnd)
		if err != nil {
			return a, err
		}
		a.TimeEnd = tEnd
	}

	return a, err
}

func millisSinceEpoch(timeStr string) (int64, error) {
	t, err := time.Parse(
		time.RFC3339,
		timeStr,
	)
	if err != nil {
		return 0, err
	}

	return t.UnixNano() / int64(time.Millisecond), nil
}

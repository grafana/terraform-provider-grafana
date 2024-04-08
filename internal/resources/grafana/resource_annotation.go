package grafana

import (
	"context"
	"strconv"
	"time"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
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

			"dashboard_id": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				Deprecated:    "Use dashboard_uid instead.",
				Description:   "The ID of the dashboard on which to create the annotation. Deprecated: Use dashboard_uid instead.",
				ConflictsWith: []string{"dashboard_uid"},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, ok := d.GetOk("dashboard_uid")
					return ok
				},
			},

			"dashboard_uid": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "The ID of the dashboard on which to create the annotation.",
				ConflictsWith: []string{"dashboard_id"},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, ok := d.GetOk("dashboard_id")
					return ok
				},
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
		"grafana_annotation",
		orgResourceIDInt("id"),
		schema,
	)
}

func CreateAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func UpdateAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func ReadAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Annotations.GetAnnotationByID(idStr)
	if err, shouldReturn := common.CheckReadError("Annotation", d, err); shouldReturn {
		return err
	}
	annotation := resp.GetPayload()

	t := time.UnixMilli(annotation.Time)
	tEnd := time.UnixMilli(annotation.TimeEnd)

	d.Set("text", annotation.Text)
	d.Set("dashboard_id", annotation.DashboardID)
	d.Set("dashboard_uid", annotation.DashboardUID)
	d.Set("panel_id", annotation.PanelID)
	d.Set("tags", annotation.Tags)
	d.Set("time", t.Format(time.RFC3339))
	d.Set("time_end", tEnd.Format(time.RFC3339))
	d.Set("org_id", strconv.FormatInt(orgID, 10))

	return nil
}

func DeleteAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		DashboardID:  int64(d.Get("dashboard_id").(int)),
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

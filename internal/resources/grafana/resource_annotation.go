package grafana

import (
	"context"
	"log"
	"net/url"
	"strconv"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceAnnotation() *schema.Resource {
	return &schema.Resource{

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
}

func CreateAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromNewOrgResource(meta, d)

	annotation, err := makeAnnotation(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := client.NewAnnotation(annotation)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))

	return ReadAnnotation(ctx, d, meta)
}

func UpdateAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	annotation, err := makeAnnotation(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("invalid Grafana annotation ID: %#v", idStr)
	}

	_, err = client.UpdateAnnotation(id, annotation)
	return diag.FromErr(err)
}

func ReadAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, idStr := ClientFromExistingOrgResource(meta, d.Id())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("invalid Grafana annotation ID: %#v", idStr)
	}
	params := url.Values{
		"type":        []string{"annotation"},
		"dashboardId": []string{strconv.FormatInt(int64(d.Get("dashboard_id").(int)), 10)},
		"panelId":     []string{strconv.FormatInt(int64(d.Get("panel_id").(int)), 10)},
		"limit":       []string{"100"},
	}
	if v, ok := d.GetOk("dashboard_uid"); ok {
		params.Set("dashboardUid", v.(string))
		params.Del("dashboardId")
	}
	annotations, err := client.Annotations(params)
	if err != nil {
		return diag.FromErr(err)
	}

	var annotation gapi.Annotation
	for _, a := range annotations {
		if a.ID == id {
			annotation = a
			break
		}
	}

	if annotation.ID <= 0 {
		log.Printf("[WARN] removing annotation %v from state because it no longer exists in grafana", idStr)
		d.SetId("")
		return nil
	}

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
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("invalid Grafana annotation ID: %#v", idStr)
	}

	if _, err = client.DeleteAnnotation(id); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func makeAnnotation(_ context.Context, d *schema.ResourceData) (*gapi.Annotation, error) {
	idStr := d.Id()
	var id int64
	var err error
	if idStr != "" {
		_, idStr = SplitOrgResourceID(idStr)
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	a := &gapi.Annotation{
		ID:           id,
		Text:         d.Get("text").(string),
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

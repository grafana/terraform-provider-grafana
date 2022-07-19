package grafana

import (
	"context"
	"net/url"
	"strconv"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceAnnotation() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/annotations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/annotations/)
`,

		CreateContext: CreateAnnotation,
		UpdateContext: UpdateAnnotation,
		DeleteContext: DeleteAnnotation,
		ReadContext:   ReadAnnotation,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
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
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The ID of the dashboard on which to create the annotation.",
			},

			"panel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
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
	client := meta.(*client).gapi

	annotation, err := makeAnnotation(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := client.NewAnnotation(annotation)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(id, 10))

	return ReadAnnotation(ctx, d, meta)
}

func UpdateAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	annotation, err := makeAnnotation(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("invalid Grafana annotation ID: %#v", idStr)
	}

	_, err = client.UpdateAnnotation(id, annotation)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ReadAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	idStr := d.Id()
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

	if &annotation == nil {
		return diag.Errorf("unable to find Grafana annotation ID %d", id)
	}

	t := time.UnixMilli(annotation.Time)
	tEnd := time.UnixMilli(annotation.TimeEnd)

	d.Set("text", annotation.Text)
	d.Set("dashboard_id", annotation.DashboardID)
	d.Set("panel_id", annotation.PanelID)
	d.Set("tags", annotation.Tags)
	d.Set("time", t.Format(time.RFC3339))
	d.Set("time_end", tEnd.Format(time.RFC3339))
	d.SetId(strconv.FormatInt(annotation.ID, 10))

	return nil
}

func DeleteAnnotation(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	idStr := d.Id()
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
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	a := &gapi.Annotation{
		ID:          id,
		Text:        d.Get("text").(string),
		PanelID:     int64(d.Get("panel_id").(int)),
		DashboardID: int64(d.Get("dashboard_id").(int)),
		Tags:        setToStringSlice(d.Get("tags").(*schema.Set)),
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

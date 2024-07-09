package machinelearning

import (
	"context"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/prometheus/common/model"
)

var resourceAlertID = common.NewResourceID(common.StringIDField("id"))

func resourceAlert() *common.Resource {
	schema := &schema.Resource{

		Description: `
A job defines the queries and model parameters for a machine learning task.
`,

		CreateContext: checkClient(resourceAlertCreate),
		ReadContext:   checkClient(resourceAlertRead),
		UpdateContext: checkClient(resourceAlertUpdate),
		DeleteContext: checkClient(resourceAlertDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"job_id": {
				Description:  "The forecast this alert belongs to.",
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"job_id", "outlier_id"},
			},
			"outlier_id": {
				Description:  "The forecast this alert belongs to.",
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"job_id", "outlier_id"},
			},
			"id": {
				Description: "The ID of the alert.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"title": {
				Description: "The title of the alert.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"anomaly_condition": {
				Description:  "The condition for when to consider a point as anomalous.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"any", "low", "high"}, false),
			},
			"for": {
				Description: "How long values must be anomalous before firing an alert.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"threshold": {
				Description: "The threshold of points over the window that need to be anomalous to alert.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"window": {
				Description: "How much time to average values over",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"labels": {
				Description: "Labels to add to the alert generated in Grafana.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"annotations": {
				Description: "Annotations to add to the alert generated in Grafana.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			"no_data_state": {
				Description:  "How the alert should be processed when no data is returned by the underlying series",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Alerting", "NoData", "OK"}, false),
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryMachineLearning,
		"grafana_machine_learning_alert",
		resourceAlertID,
		schema,
	)
}

func resourceAlertCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	alert, err := makeMLAlert(d)
	if err != nil {
		return diag.FromErr(err)
	}
	jobID := d.Get("job_id").(string)
	if jobID != "" {
		alert, err = c.NewJobAlert(ctx, jobID, alert)
	} else {
		outlierID := d.Get("outlier_id").(string)
		alert, err = c.NewOutlierAlert(ctx, outlierID, alert)
	}
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(alert.ID)
	return resourceAlertRead(ctx, d, meta)
}

func resourceAlertRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	var (
		alert mlapi.Alert
		err   error
	)
	jobID := d.Get("job_id").(string)
	if jobID != "" {
		alert, err = c.JobAlert(ctx, jobID, d.Id())
	} else {
		outlierID := d.Get("outlier_id").(string)
		alert, err = c.OutlierAlert(ctx, outlierID, d.Id())
	}

	if err, shouldReturn := common.CheckReadError("alert", d, err); shouldReturn {
		return err
	}

	d.Set("title", alert.Title)
	d.Set("anomaly_condition", alert.AnomalyCondition)
	if alert.For > 0 {
		d.Set("for", alert.For.String())
	}
	d.Set("threshold", alert.Threshold)
	if alert.Window > 0 {
		d.Set("window", alert.Window.String())
	}
	d.Set("labels", alert.Labels)
	d.Set("annotations", alert.Annotations)
	d.Set("no_data_state", alert.NoDataState)

	return nil
}

func resourceAlertUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	alert, err := makeMLAlert(d)
	if err != nil {
		return diag.FromErr(err)
	}
	jobID := d.Get("job_id").(string)
	if jobID != "" {
		_, err = c.UpdateJobAlert(ctx, jobID, alert)
	} else {
		outlierID := d.Get("outlier_id").(string)
		_, err = c.UpdateOutlierAlert(ctx, outlierID, alert)
	}

	if err != nil {
		return diag.FromErr(err)
	}
	return resourceAlertRead(ctx, d, meta)
}

func resourceAlertDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	jobID := d.Get("job_id").(string)
	var err error
	if jobID != "" {
		err = c.DeleteJobAlert(ctx, jobID, d.Id())
	} else {
		outlierID := d.Get("outlier_id").(string)
		err = c.DeleteOutlierAlert(ctx, outlierID, d.Id())
	}
	return diag.FromErr(err)
}

func makeMLAlert(d *schema.ResourceData) (mlapi.Alert, error) {
	forClause, err := parseDuration(d.Get("for").(string))
	if err != nil {
		return mlapi.Alert{}, err
	}
	window, err := parseDuration(d.Get("window").(string))
	if err != nil {
		return mlapi.Alert{}, err
	}
	labels := map[string]string{}
	for k, v := range d.Get("labels").(map[string]interface{}) {
		labels[k] = v.(string)
	}
	annotations := map[string]string{}
	for k, v := range d.Get("annotations").(map[string]interface{}) {
		annotations[k] = v.(string)
	}
	return mlapi.Alert{
		ID:               d.Id(),
		Title:            d.Get("title").(string),
		AnomalyCondition: mlapi.AnomalyCondition(d.Get("anomaly_condition").(string)),
		For:              forClause,
		Threshold:        d.Get("threshold").(string),
		Window:           window,
		Labels:           labels,
		Annotations:      annotations,
		NoDataState:      mlapi.NoDataState(d.Get("no_data_state").(string)),
	}, nil
}

func parseDuration(s string) (model.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return model.ParseDuration(s)
}

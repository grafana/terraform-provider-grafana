package slo

import (
	"context"
	"fmt"
	"regexp"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceSlo() *schema.Resource {
	return &schema.Resource{
		Description: `
Resource manages Grafana SLOs. 

* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/api/)
* [Additional Information On Alerting Rule Annotations and Labels](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#templating/)
		`,
		CreateContext: resourceSloCreate,
		ReadContext:   resourceSloRead,
		UpdateContext: resourceSloUpdate,
		DeleteContext: resourceSloDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  `Name should be a short description of your indicator. Consider names like "API Availability"`,
				ValidateFunc: validation.StringLenBetween(0, 128),
			},
			"description": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  `Description is a free-text field that can provide more context to an SLO.`,
				ValidateFunc: validation.StringLenBetween(0, 1024),
			},
			"query": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: `Query describes the indicator that will be measured against the objective. Freeform Query types are currently supported.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Description:  `Query type must be one of: "freeform", "query", "ratio", or "threshold"`,
							ValidateFunc: validation.StringInSlice([]string{"freeform", "query", "ratio", "threshold"}, false),
							Required:     true,
						},
						"freeform": &schema.Schema{
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"query": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "Freeform Query Field",
									},
								},
							},
						},
						"ratio": &schema.Schema{
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"success_metric": &schema.Schema{
										Type:        schema.TypeString,
										Description: `Counter metric for success events (numerator)`,
										Required:    true,
									},
									"total_metric": &schema.Schema{
										Type:        schema.TypeString,
										Description: `Metric for total events (denominator)`,
										Required:    true,
									},
									"group_by_labels": &schema.Schema{
										Type:        schema.TypeList,
										Description: `Defines Group By Labels used for per-label alerting. These appear as variables on SLO dashboards to enable filtering and aggregation. Labels must adhere to Prometheus label name schema - "^[a-zA-Z_][a-zA-Z0-9_]*$"`,
										Optional:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
			"label": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: `Additional labels that will be attached to all metrics generated from the query. These labels are useful for grouping SLOs in dashboard views that you create by hand. Labels must adhere to Prometheus label name schema - "^[a-zA-Z_][a-zA-Z0-9_]*$"`,
				Elem:        keyvalueSchema,
			},
			"objectives": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: `Over each rolling time window, the remaining error budget will be calculated, and separate alerts can be generated for each time window based on the SLO burn rate or remaining error budget.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": &schema.Schema{
							Type:         schema.TypeFloat,
							Required:     true,
							ValidateFunc: validation.FloatBetween(0, 1),
							Description:  `Value between 0 and 1. If the value of the query is above the objective, the SLO is met.`,
						},
						"window": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  `A Prometheus-parsable time duration string like 24h, 60m. This is the time window the objective is measured over.`,
							ValidateFunc: validation.StringMatch(regexp.MustCompile(`^\d+(ms|s|m|h|d|w|y)$`), "Objective window must be a Prometheus-parsable time duration"),
						},
					},
				},
			},
			"alerting": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Description: `Configures the alerting rules that will be generated for each
				time window associated with the SLO. Grafana SLOs can generate
				alerts when the short-term error budget burn is very high, the
				long-term error budget burn rate is high, or when the remaining
				error budget is below a certain threshold. Annotations and Labels support templating.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Description: `Labels will be attached to all alerts generated by any of these rules.`,
							Elem:        keyvalueSchema,
						},
						"annotation": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Description: `Annotations will be attached to all alerts generated by any of these rules.`,
							Elem:        keyvalueSchema,
						},
						"fastburn": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Alerting Rules generated for Fast Burn alerts",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"label": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Labels to attach only to Fast Burn alerts.",
										Elem:        keyvalueSchema,
									},
									"annotation": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Annotations to attach only to Fast Burn alerts.",
										Elem:        keyvalueSchema,
									},
								},
							},
						},
						"slowburn": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Alerting Rules generated for Slow Burn alerts",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"label": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Labels to attach only to Slow Burn alerts.",
										Elem:        keyvalueSchema,
									},
									"annotation": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Annotations to attach only to Slow Burn alerts.",
										Elem:        keyvalueSchema,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

var keyvalueSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"key": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"value": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
	},
}

func resourceSloCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	slo, err := packSloResource(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to Pack SLO",
			Detail:   err.Error(),
		})
		return diags
	}

	client := m.(*common.Client).GrafanaAPI
	response, err := client.CreateSlo(slo)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create SLO - API",
			Detail:   fmt.Sprintf("API Error Message:%s", err.Error()),
		})
		return diags
	}

	d.SetId(response.UUID)
	resourceSloRead(ctx, d, m)

	return resourceSloRead(ctx, d, m)
}

// resourceSloRead - sends a GET Request to the single SLO Endpoint
func resourceSloRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	sloID := d.Id()

	client := m.(*common.Client).GrafanaAPI
	slo, err := client.GetSlo(sloID)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to Fetch Slo with ID: %s", sloID),
			Detail:   fmt.Sprintf("API Error Message:%s", err.Error()),
		})
		return diags
	}

	setTerraformState(d, slo)

	return diags
}

func resourceSloUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	sloID := d.Id()

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("query") || d.HasChange("label") || d.HasChange("objectives") || d.HasChange("alerting") {
		slo, err := packSloResource(d)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to Pack SLO",
				Detail:   err.Error(),
			})
			return diags
		}

		client := m.(*common.Client).GrafanaAPI

		err = client.UpdateSlo(sloID, slo)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Unable to Update Slo with ID: %s", sloID),
				Detail:   fmt.Sprintf("API Error Message:%s", err.Error()),
			})
			return diags
		}
	}

	return resourceSloRead(ctx, d, m)
}

func resourceSloDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sloID := d.Id()

	client := m.(*common.Client).GrafanaAPI

	return diag.FromErr(client.DeleteSlo(sloID))
}

// Fetches all the Properties defined on the Terraform SLO State Object and converts it
// to a Slo so that it can be converted to JSON and sent to the API
func packSloResource(d *schema.ResourceData) (gapi.Slo, error) {
	var (
		tfalerting gapi.Alerting
		tflabels   []gapi.Label
	)

	tfname := d.Get("name").(string)
	tfdescription := d.Get("description").(string)
	query := d.Get("query").([]interface{})[0].(map[string]interface{})
	tfquery, err := packQuery(query)
	if err != nil {
		return gapi.Slo{}, err
	}

	objectives := d.Get("objectives").([]interface{})
	tfobjective := packObjectives(objectives)

	labels := d.Get("label").([]interface{})
	if labels != nil {
		tflabels = packLabels(labels)
	}

	slo := gapi.Slo{
		UUID:        d.Id(),
		Name:        tfname,
		Description: tfdescription,
		Objectives:  tfobjective,
		Query:       tfquery,
		Alerting:    nil,
		Labels:      tflabels,
	}

	if alerting, ok := d.GetOk("alerting"); ok {
		alertData := alerting.([]interface{})

		// if the Alerting field is an empty block, alertData[0] has a value of nil
		if alertData[0] != nil {
			// only pack the Alerting TF fields if the user populates the Alerting field with blocks
			alert := alertData[0].(map[string]interface{})
			tfalerting = packAlerting(alert)
		}

		slo.Alerting = &tfalerting
	}

	return slo, nil
}

func packQuery(query map[string]interface{}) (gapi.Query, error) {
	if query["type"] == "freeform" {
		freeformquery := query["freeform"].([]interface{})[0].(map[string]interface{})
		querystring := freeformquery["query"].(string)

		sloQuery := gapi.Query{
			Freeform: &gapi.FreeformQuery{Query: querystring},
			Type:     gapi.QueryTypeFreeform,
		}

		return sloQuery, nil
	}

	if query["type"] == "ratio" {
		ratioquery := query["ratio"].([]interface{})[0].(map[string]interface{})
		successMetric := ratioquery["success_metric"].(string)
		totalMetric := ratioquery["total_metric"].(string)
		groupByLabels := ratioquery["group_by_labels"].([]interface{})

		var labels []string

		for ind := range groupByLabels {
			if groupByLabels[ind] == nil {
				labels = append(labels, "")
				continue
			}
			labels = append(labels, groupByLabels[ind].(string))
		}

		sloQuery := gapi.Query{
			Ratio: &gapi.RatioQuery{
				SuccessMetric: gapi.MetricDef{
					PrometheusMetric: successMetric,
					Type:             nil,
				},
				TotalMetric: gapi.MetricDef{
					PrometheusMetric: totalMetric,
					Type:             nil,
				},
				GroupByLabels: labels,
			},
			Type: gapi.QueryTypeRatio,
		}

		return sloQuery, nil
	}

	return gapi.Query{}, fmt.Errorf("%s query type not implemented", query["type"])
}

func packObjectives(tfobjectives []interface{}) []gapi.Objective {
	objectives := []gapi.Objective{}

	for ind := range tfobjectives {
		tfobjective := tfobjectives[ind].(map[string]interface{})
		objective := gapi.Objective{
			Value:  tfobjective["value"].(float64),
			Window: tfobjective["window"].(string),
		}
		objectives = append(objectives, objective)
	}

	return objectives
}

func packLabels(tfLabels []interface{}) []gapi.Label {
	labelSlice := []gapi.Label{}

	for ind := range tfLabels {
		currLabel := tfLabels[ind].(map[string]interface{})
		curr := gapi.Label{
			Key:   currLabel["key"].(string),
			Value: currLabel["value"].(string),
		}

		labelSlice = append(labelSlice, curr)
	}

	return labelSlice
}

func packAlerting(tfAlerting map[string]interface{}) gapi.Alerting {
	annots := tfAlerting["annotation"].([]interface{})
	tfAnnots := packLabels(annots)

	labels := tfAlerting["label"].([]interface{})
	tfLabels := packLabels(labels)

	fastBurn := tfAlerting["fastburn"].([]interface{})
	tfFastBurn := packAlertMetadata(fastBurn)

	slowBurn := tfAlerting["slowburn"].([]interface{})
	tfSlowBurn := packAlertMetadata(slowBurn)

	alerting := gapi.Alerting{
		Annotations: tfAnnots,
		Labels:      tfLabels,
		FastBurn:    &tfFastBurn,
		SlowBurn:    &tfSlowBurn,
	}

	return alerting
}

func packAlertMetadata(metadata []interface{}) gapi.AlertingMetadata {
	meta := metadata[0].(map[string]interface{})

	labels := meta["label"].([]interface{})
	tflabels := packLabels(labels)

	annots := meta["annotation"].([]interface{})
	tfannots := packLabels(annots)

	apiMetadata := gapi.AlertingMetadata{
		Labels:      tflabels,
		Annotations: tfannots,
	}

	return apiMetadata
}

func setTerraformState(d *schema.ResourceData, slo gapi.Slo) {
	d.Set("name", slo.Name)
	d.Set("description", slo.Description)

	d.Set("query", unpackQuery(slo.Query))

	retLabels := unpackLabels(&slo.Labels)
	d.Set("label", retLabels)

	retObjectives := unpackObjectives(slo.Objectives)
	d.Set("objectives", retObjectives)

	retAlerting := unpackAlerting(slo.Alerting)
	d.Set("alerting", retAlerting)
}

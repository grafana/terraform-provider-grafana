package slo

import (
	"context"
	"fmt"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSlo() *schema.Resource {
	return &schema.Resource{
		Description: `
Resource manages Grafana SLOs. 

* [Official documentation](https://grafana.com/docs/grafana-cloud/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/slo/api/)
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
				Type:        schema.TypeString,
				Required:    true,
				Description: `Name should be a short description of your indicator. Consider names like "API Availability"`,
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: `Description is a free-text field that can provide more context to an SLO.`,
			},
			"query": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: `Query describes the indicator that will be measured against the objective.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"query_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: `Query type must be one of freeform, ratio, percentile, or threshold Queries.`,
						},
						"freeform_query": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"ratio_query": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"success_metric": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"metric": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"type": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"total_metric": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"metric": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"type": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
						"percentile_query": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"histogram_metric": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"metric": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"type": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"percentile": {
										Type:     schema.TypeFloat,
										Optional: true,
									},
								},
							},
						},
						"threshold_query": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"threshold_metric": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"metric": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
												"type": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
								},
							},
						},
						"threshold": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"value": &schema.Schema{
										Type:     schema.TypeFloat,
										Optional: true,
									},
									"operator": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"group_by_labels": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"labels": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: `Additional labels that will be attached to all metrics generated from the query. These labels are useful for grouping SLOs in dashboard views that you create by hand.`,
				Elem: &schema.Resource{
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
				},
			},
			"objectives": &schema.Schema{
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: `Over each rolling time window, the remaining error budget will be calculated, and separate alerts can be generated for each time window based on the SLO burn rate or remaining error budget.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"objective_value": &schema.Schema{
							Type:        schema.TypeFloat,
							Required:    true,
							Description: `Value between 0 and 1. If the value of the query is above the objective, the SLO is met.`,
						},
						"objective_window": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: `A Prometheus-parsable time duration string like 24h, 60m. This is the time window the objective is measured over.`,
						},
					},
				},
			},
			"dashboard_uid": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: `A reference to a dashboard that the plugin has installed in Grafana based on this SLO. This field is read-only, it is generated by the Grafana SLO Plugin.`,
			},
			"alerting": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Description: `Configures the alerting rules that will be generated for each
				time window associated with the SLO. Grafana SLOs can generate
				alerts when the short-term error budget burn is very high, the
				long-term error budget burn rate is high, or when the remaining
				error budget is below a certain threshold.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"labels": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Description: `Labels will be attached to all alerts generated by any of these rules.`,
							Elem: &schema.Resource{
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
							},
						},
						"annotations": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Description: `Annotations will be attached to all alerts generated by any of these rules.`,
							Elem: &schema.Resource{
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
							},
						},
						"fastburn": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Alerting Rules generated for Fast Burn alerts",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"labels": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Labels to attach only to Fast Burn alerts.",
										Elem: &schema.Resource{
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
										},
									},
									"annotations": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Annotations to attach only to Fast Burn alerts.",
										Elem: &schema.Resource{
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
										},
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
									"labels": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Labels to attach only to Slow Burn alerts.",
										Elem: &schema.Resource{
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
										},
									},
									"annotations": &schema.Schema{
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Annotations to attach only to Slow Burn alerts.",
										Elem: &schema.Resource{
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
										},
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

// SLO Resource is defined by the user within the Terraform State file
// When 'terraform apply' is executed, it sends a POST Request and converts
// the data within the Terraform State into a JSON Object which is then sent to the API
// Following this, a READ is executed for the newly created SLO, which is then displayed within the
// terminal that Terraform is running in
func resourceSloCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	slo := packSloResource(d)

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

	// Get the response back from the API, we need to set the ID of the Terraform Resource
	d.SetId(response.UUID)

	// Executes a READ, displays the newly created SLO Resource within Terraform
	resourceSloRead(ctx, d, m)

	return diags
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

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("query") || d.HasChange("labels") || d.HasChange("objectives") || d.HasChange("alerting") {
		slo := packSloResource(d)

		client := m.(*common.Client).GrafanaAPI
		err := client.UpdateSlo(sloID, slo)

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
	var diags diag.Diagnostics

	sloID := d.Id()

	client := m.(*common.Client).GrafanaAPI
	client.DeleteSlo(sloID)

	d.SetId("")

	return diags
}

// Fetches all the Properties defined on the Terraform SLO State Object and converts it
// to a Slo so that it can be converted to JSON and sent to the API
func packSloResource(d *schema.ResourceData) gapi.Slo {
	var (
		tfalerting gapi.Alerting
		tflabels   []gapi.Label
	)

	tfname := d.Get("name").(string)
	tfdescription := d.Get("description").(string)

	// query := d.Get("query").(string)
	// tfquery := packQuery(query)

	query := d.Get("query").([]interface{}) // type assertion
	queryElements := query[0].(map[string]interface{})
	tfquery := packQuery(queryElements)

	// Assumes that each SLO only has one Objective Value and one Objective Window
	// Need to Adjust this for Multiple Objectives(?)
	objectives := d.Get("objectives").([]interface{})
	objective := objectives[0].(map[string]interface{})
	tfobjective := packObjective(objective)

	labels := d.Get("labels").([]interface{})
	if labels != nil {
		tflabels = packLabels(labels)
	}

	alerting := d.Get("alerting").([]interface{})
	if len(alerting) > 0 {
		alert := alerting[0].(map[string]interface{})
		tfalerting = packAlerting(alert)
	}

	slo := gapi.Slo{
		UUID:        d.Id(),
		Name:        tfname,
		Description: tfdescription,
		Objectives:  tfobjective,
		Query:       tfquery,
		Alerting:    &tfalerting,
		Labels:      tflabels,
	}

	return slo
}

func packQuery(tfquery map[string]interface{}) gapi.Query {
	var query gapi.Query

	querytype := tfquery["query_type"].(string)
	// querythresholdstruct := tfquery["threshold"].([]interface{})
	// querylabelsstruct := tfquery["group_by_labels"].([]interface{})

	switch querytype {
	case "freeform":
		freeformQuery := tfquery["freeform_query"].(string)
		query = gapi.Query{
			FreeformQuery: generateFreeformQuery(freeformQuery),
			// GroupByLabels: generateQueryLabels(querylabelsstruct),
		}
		return query

	// case "ratio":
	// 	ratioQuery := tfquery["ratio_query"].([]interface{})[0].(map[string]interface{})

	// 	query = gapi.Query{
	// 		RatioQuery:    generateRatioQuery(ratioQuery),
	// 		GroupByLabels: generateQueryLabels(querylabelsstruct),
	// 	}
	// 	return query
	// case "percentile":
	// 	percentileQuery := tfquery["percentile_query"].([]interface{})[0].(map[string]interface{})

	// 	query = gapi.Query{
	// 		PercentileQuery: generatePercentileQuery(percentileQuery),
	// 		GroupByLabels:   generateQueryLabels(querylabelsstruct),
	// 	}
	// case "threshold":
	// 	thresholdQuery := tfquery["threshold_query"].([]interface{})[0].(map[string]interface{})

	// 	query = gapi.Query{
	// 		ThresholdQuery: generateThresholdQuery(thresholdQuery),
	// 		Threshold:      generateThreshold(querythresholdstruct),
	// 		GroupByLabels:  generateQueryLabels(querylabelsstruct),
	// 	}
	// 	return query

	default:
		return query
	}
}

func generateFreeformQuery(query string) gapi.FreeformQuery {
	return gapi.FreeformQuery{
		Query: query,
	}
}

func packObjective(tfobjective map[string]interface{}) []gapi.Objective {
	objective := gapi.Objective{
		Value:  tfobjective["objective_value"].(float64),
		Window: tfobjective["objective_window"].(string),
	}

	objectiveSlice := []gapi.Objective{}
	objectiveSlice = append(objectiveSlice, objective)

	return objectiveSlice
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
	annots := tfAlerting["annotations"].([]interface{})
	tfAnnots := packLabels(annots)

	labels := tfAlerting["labels"].([]interface{})
	tfLabels := packLabels(labels)

	fastBurn := tfAlerting["fastburn"].([]interface{})
	tfFastBurn := packAlertMetadata(fastBurn)

	slowBurn := tfAlerting["slowburn"].([]interface{})
	tfSlowBurn := packAlertMetadata(slowBurn)

	alerting := gapi.Alerting{
		Annotations: &tfAnnots,
		Labels:      &tfLabels,
		FastBurn:    &tfFastBurn,
		SlowBurn:    &tfSlowBurn,
	}

	return alerting
}

func packAlertMetadata(metadata []interface{}) gapi.AlertMetadata {
	meta := metadata[0].(map[string]interface{})

	labels := meta["labels"].([]interface{})
	tflabels := packLabels(labels)

	annots := meta["annotations"].([]interface{})
	tfannots := packLabels(annots)

	apiMetadata := gapi.AlertMetadata{
		Labels:      &tflabels,
		Annotations: &tfannots,
	}

	return apiMetadata
}

func setTerraformState(d *schema.ResourceData, slo gapi.Slo) {
	d.Set("name", slo.Name)
	d.Set("description", slo.Description)
	d.Set("dashboard_uid", slo.DrillDownDashboardRef.UID)
	d.Set("query", unpackQuery(slo.Query))

	retLabels := unpackLabels(&slo.Labels)
	d.Set("labels", retLabels)

	retObjectives := unpackObjectives(slo.Objectives)
	d.Set("objectives", retObjectives)

	retAlerting := unpackAlerting(slo.Alerting)
	d.Set("alerting", retAlerting)
}

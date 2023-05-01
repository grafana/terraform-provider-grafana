package slo

import (
	"context"
	"errors"
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

func resourceSloCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	slo, err := packSloResource(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to Pack SLO",
			Detail:   fmt.Sprintf("Error Message:%s", err.Error()),
		})
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
	return diags
}

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
		slo, err := packSloResource(d)

		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to Pack SLO",
				Detail:   fmt.Sprintf("Error Message:%s", err.Error()),
			})
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
	var diags diag.Diagnostics

	sloID := d.Id()

	client := m.(*common.Client).GrafanaAPI
	client.DeleteSlo(sloID)

	d.SetId("")

	return diags
}

func packSloResource(d *schema.ResourceData) (gapi.Slo, error) {
	var (
		tfalerting gapi.Alerting
		tflabels   []gapi.Label
	)

	tfname := d.Get("name").(string)
	tfdescription := d.Get("description").(string)

	query := d.Get("query").([]interface{})
	queryElements := query[0].(map[string]interface{})
	tfquery, err := packQuery(queryElements)

	if err != nil {
		return gapi.Slo{}, err
	}

	// Assumes that each SLO only has one Objective Value and one Objective Window
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

	return slo, nil
}

func packQuery(tfquery map[string]interface{}) (gapi.Query, error) {
	var query gapi.Query

	queryType := tfquery["query_type"].(string)
	queryLabelsStruct := tfquery["group_by_labels"].([]interface{})

	switch queryType {
	case "freeform":
		freeformQuery := tfquery["freeform_query"].(string)
		query = gapi.Query{
			FreeformQuery: packFreeformQuery(freeformQuery),
			GroupByLabels: packQueryLabels(queryLabelsStruct),
		}
		return query, nil
	case "ratio":
		return query, errors.New("ratio query not yet implemented")
	case "percentile":
		return query, errors.New("percentile query not yet implemented")
	case "threshold":
		return query, errors.New("threshold query not yet implemented")
	default:
		return query, errors.New("query must be of type freeform, ratio, percentile, or threshold")
	}
}

func packQueryLabels(tfquerylabels []interface{}) []string {
	var retQueryLabels []string

	for _, label := range tfquerylabels {
		strLabel := label.(string)
		retQueryLabels = append(retQueryLabels, strLabel)

	}
	return retQueryLabels
}

func packFreeformQuery(query string) gapi.FreeformQuery {
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

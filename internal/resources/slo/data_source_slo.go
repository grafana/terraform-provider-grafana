package slo

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceSlo() *schema.Resource {
	return &schema.Resource{
		Description: `
Datasource for retrieving all SLOs.
		
* [Official documentation](https://grafana.com/docs/grafana-cloud/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/slo/api/)
				`,
		ReadContext: datasourceSloRead,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"slos": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: `Returns a list of all SLOs"`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:        schema.TypeString,
							Description: `A unique, random identifier. This value will also be the name of the resource stored in the API server. This value is read-only.`,
							Computed:    true,
						},
						"name": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: `Name should be a short description of your indicator. Consider names like "API Availability"`,
						},
						"description": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
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
								},
							},
						},
						"labels": &schema.Schema{
							Type:        schema.TypeList,
							Computed:    true,
							Description: `Additional labels that will be attached to all metrics generated from the query. These labels are useful for grouping SLOs in dashboard views that you create by hand.`,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"objectives": &schema.Schema{
							Type:        schema.TypeList,
							Computed:    true,
							Description: `Over each rolling time window, the remaining error budget will be calculated, and separate alerts can be generated for each time window based on the SLO burn rate or remaining error budget.`,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"value": &schema.Schema{
										Type:        schema.TypeFloat,
										Computed:    true,
										Description: `Value between 0 and 1. If the value of the query is above the objective, the SLO is met.`,
									},
									"window": &schema.Schema{
										Type:        schema.TypeString,
										Computed:    true,
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
							Computed: true,
							Description: `Configures the alerting rules that will be generated for each
								time window associated with the SLO. Grafana SLOs can generate
								alerts when the short-term error budget burn is very high, the
								long-term error budget burn rate is high, or when the remaining
								error budget is below a certain threshold.`,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"labels": &schema.Schema{
										Type:        schema.TypeList,
										Computed:    true,
										Description: `Labels will be attached to all alerts generated by any of these rules.`,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"value": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"annotations": &schema.Schema{
										Type:        schema.TypeList,
										Computed:    true,
										Description: `Annotations will be attached to all alerts generated by any of these rules.`,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"value": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"fastburn": &schema.Schema{
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Alerting Rules generated for Fast Burn alerts",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"labels": &schema.Schema{
													Type:        schema.TypeList,
													Computed:    true,
													Description: "Labels to attach only to Fast Burn alerts.",
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
												"annotations": &schema.Schema{
													Type:        schema.TypeList,
													Computed:    true,
													Description: "Annotations to attach only to Fast Burn alerts.",
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
											},
										},
									},
									"slowburn": &schema.Schema{
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Alerting Rules generated for Slow Burn alerts",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"labels": &schema.Schema{
													Type:        schema.TypeList,
													Computed:    true,
													Description: "Labels to attach only to Slow Burn alerts.",
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
												"annotations": &schema.Schema{
													Type:        schema.TypeList,
													Computed:    true,
													Description: "Annotations to attach only to Slow Burn alerts.",
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
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
				},
			},
		},
	}
}

func datasourceSloRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := m.(*common.Client).GrafanaAPI
	apiSlos, _ := client.ListSlos()

	terraformSlos := []interface{}{}

	if len(apiSlos.Slos) == 0 {
		d.SetId("slos")
		d.Set("slos", terraformSlos)

		return diags
	}

	for _, slo := range apiSlos.Slos {
		terraformSlo := convertDatasourceSlo(slo)
		terraformSlos = append(terraformSlos, terraformSlo)
	}

	d.SetId("slos")
	d.Set("slos", terraformSlos)

	return diags
}

func convertDatasourceSlo(slo gapi.Slo) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["uuid"] = slo.UUID
	ret["name"] = slo.Name
	ret["description"] = slo.Description
	ret["dashboard_uid"] = slo.DrillDownDashboardRef.UID
	ret["query"] = unpackQuery(slo.Query)

	retLabels := unpackLabels(&slo.Labels)
	ret["labels"] = retLabels

	retObjectives := unpackObjectives(slo.Objectives)
	ret["objectives"] = retObjectives

	retAlerting := unpackAlerting(slo.Alerting)
	ret["alerting"] = retAlerting

	return ret
}

func unpackQuery(query gapi.Query) []map[string]interface{} {
	retQueries := []map[string]interface{}{}

	if query.FreeformQuery.Query != "" {
		retQuery := make(map[string]interface{})
		retQuery["query_type"] = "freeform"
		retQuery["freeform_query"] = query.FreeformQuery.Query
		retQueries = append(retQueries, retQuery)
	}

	return retQueries
}

func unpackObjectives(objectives []gapi.Objective) []map[string]interface{} {
	retObjectives := []map[string]interface{}{}

	for _, objective := range objectives {
		retObjective := make(map[string]interface{})
		retObjective["value"] = objective.Value
		retObjective["window"] = objective.Window
		retObjectives = append(retObjectives, retObjective)
	}

	return retObjectives
}

func unpackLabels(labels *[]gapi.Label) []map[string]interface{} {
	retLabels := []map[string]interface{}{}

	if labels != nil {
		for _, label := range *labels {
			retLabel := make(map[string]interface{})
			retLabel["key"] = label.Key
			retLabel["value"] = label.Value
			retLabels = append(retLabels, retLabel)
		}
		return retLabels
	}

	return nil
}

func unpackAlerting(alertData *gapi.Alerting) []map[string]interface{} {
	retAlertData := []map[string]interface{}{}

	if alertData == nil {
		return retAlertData
	}

	alertObject := make(map[string]interface{})
	alertObject["labels"] = unpackLabels(alertData.Labels)
	alertObject["annotations"] = unpackLabels(alertData.Annotations)

	if alertData.FastBurn != nil {
		alertObject["fastburn"] = unpackAlertingMetadata(*alertData.FastBurn)
	}

	if alertData.SlowBurn != nil {
		alertObject["slowburn"] = unpackAlertingMetadata(*alertData.SlowBurn)
	}

	retAlertData = append(retAlertData, alertObject)
	return retAlertData
}

func unpackAlertingMetadata(metaData gapi.AlertMetadata) []map[string]interface{} {
	retAlertMetaData := []map[string]interface{}{}
	labelsAnnotsStruct := make(map[string]interface{})

	if metaData.Annotations != nil {
		retAnnotations := unpackLabels(metaData.Annotations)
		labelsAnnotsStruct["annotations"] = retAnnotations
	}

	if metaData.Labels != nil {
		retLabels := unpackLabels(metaData.Labels)
		labelsAnnotsStruct["labels"] = retLabels
	}

	retAlertMetaData = append(retAlertMetaData, labelsAnnotsStruct)
	return retAlertMetaData
}

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
		ReadContext: datasourceSloRead,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"slos": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"labels": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"query": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"objectives": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"objective_value": &schema.Schema{
										Type:     schema.TypeFloat,
										Optional: true,
									},
									"objective_window": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"dashboard_uid": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"alerting": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"labels": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"value": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"annotations": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"value": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"fastburn": &schema.Schema{
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"labels": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
												"annotations": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
											},
										},
									},
									"slowburn": &schema.Schema{
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"labels": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
												"annotations": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"value": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
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

// Function sends a GET request to the SLO API Endpoint which returns a list of all SLOs
// Maps the API Response body to the Terraform Schema and displays as a READ in the terminal
func datasourceSloRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := m.(*common.Client).GrafanaAPI
	apiSlos, _ := client.ListSlos()

	terraformSlos := []interface{}{}

	for _, slo := range apiSlos.Slos {
		terraformSlo := convertDatasourceSlo(slo)
		terraformSlos = append(terraformSlos, terraformSlo)
	}

	d.Set("slos", terraformSlos)
	d.SetId(apiSlos.Slos[0].UUID)

	return diags
}

func convertDatasourceSlo(slo gapi.Slo) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["uuid"] = slo.UUID
	ret["name"] = slo.Name
	ret["description"] = slo.Description
	ret["dashboard_uid"] = slo.DrilldownDashboardRef.UID
	ret["query"] = unpackQuery(slo.Query)

	retLabels := unpackLabels(slo.Labels)
	ret["labels"] = retLabels

	retObjectives := unpackObjectives(slo.Objectives)
	ret["objectives"] = retObjectives

	retAlerting := unpackAlerting(slo.Alerting)
	ret["alerting"] = retAlerting

	return ret
}

// TBD for Other Query Types Once Implemented
func unpackQuery(query gapi.Query) string {
	if query.FreeformQuery.Query != "" {
		return query.FreeformQuery.Query
	}

	return "Query Type Not Implemented"
}

func unpackObjectives(objectives []gapi.Objective) []map[string]interface{} {
	retObjectives := []map[string]interface{}{}

	for _, objective := range objectives {
		retObjective := make(map[string]interface{})
		retObjective["objective_value"] = objective.Value
		retObjective["objective_window"] = objective.Window
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

	alertObject := make(map[string]interface{})
	alertObject["name"] = alertData.Name
	alertObject["labels"] = unpackLabels(alertData.Labels)
	alertObject["annotations"] = unpackLabels(alertData.Annotations)
	alertObject["fastburn"] = unpackAlertingMetadata(*alertData.FastBurn)
	alertObject["slowburn"] = unpackAlertingMetadata(*alertData.SlowBurn)

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

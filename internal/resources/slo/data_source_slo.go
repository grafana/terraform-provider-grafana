package slo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

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
						"service": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
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
						"dashboard_ref": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
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

func datasourceSloRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serverPort := 3000
	requestURL := fmt.Sprintf("http://localhost:%d/api/plugins/grafana-slo-app/resources/v1/slo", serverPort)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		log.Fatalln(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var sloList SloList

	err = json.Unmarshal(b, &sloList)
	if err != nil {
		fmt.Println("error:", err)
	}

	terrformSlos := []interface{}{}
	for _, slo := range sloList.Slos {
		terraformSlo := convertDatasourceSlo(slo)
		terrformSlos = append(terrformSlos, terraformSlo)
	}

	d.Set("slos", terrformSlos)
	d.SetId(sloList.Slos[0].Uuid)

	var diags diag.Diagnostics
	return diags
}

func convertDatasourceSlo(slo Slo) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["uuid"] = slo.Uuid
	ret["name"] = slo.Name
	ret["description"] = slo.Description
	ret["service"] = slo.Service
	ret["query"] = unpackQuery(slo.Query)

	retLabels := unpackLabels(slo.Labels)
	ret["labels"] = retLabels

	retDashboard := unpackDashboard(slo)
	ret["dashboard_ref"] = retDashboard

	retObjectives := unpackObjectives(slo.Objectives)
	ret["objectives"] = retObjectives

	retAlerting := unpackAlerting(slo.Alerting)
	ret["alerting"] = retAlerting

	return ret

}

// TBD for Other Query Types Once Implemented
func unpackQuery(query Query) string {
	if query.FreeformQuery.Query != "" {
		return query.FreeformQuery.Query
	}

	return "Query Type Not Implemented"

}

func unpackObjectives(objectives []Objective) []map[string]interface{} {
	retObjectives := []map[string]interface{}{}

	for _, objective := range objectives {
		retObjective := make(map[string]interface{})
		retObjective["objective_value"] = objective.Value
		retObjective["objective_window"] = objective.Window
		retObjectives = append(retObjectives, retObjective)
	}

	return retObjectives
}

func unpackLabels(labels *[]Label) []map[string]interface{} {
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

func unpackDashboard(slo Slo) map[string]interface{} {
	retDashboard := make(map[string]interface{})

	if slo.DrilldownDashboardRef != nil {
		retDashboard["dashboard_id"] = strconv.Itoa(slo.DrilldownDashboardRef.ID)
		retDashboard["dashboard_uid"] = slo.DrilldownDashboardRef.UID
	}

	if slo.DrilldownDashboardUid != "" {
		retDashboard["dashboard_uid"] = slo.DrilldownDashboardUid
	}

	return retDashboard
}

func unpackAlerting(AlertData *Alerting) []map[string]interface{} {
	retAlertData := []map[string]interface{}{}

	alertObject := make(map[string]interface{})
	alertObject["name"] = AlertData.Name
	alertObject["labels"] = unpackLabels(AlertData.Labels)
	alertObject["annotations"] = unpackLabels(AlertData.Annotations)
	alertObject["fastburn"] = unpackAlertingMetadata(*AlertData.FastBurn)
	alertObject["slowburn"] = unpackAlertingMetadata(*AlertData.SlowBurn)

	retAlertData = append(retAlertData, alertObject)

	return retAlertData

}

func unpackAlertingMetadata(Metadata AlertMetadata) []map[string]interface{} {
	retAlertMetaData := []map[string]interface{}{}
	labelsAnnotsStruct := make(map[string]interface{})

	if Metadata.Annotations != nil {
		retAnnotations := unpackLabels(Metadata.Annotations)
		labelsAnnotsStruct["annotations"] = retAnnotations
	}

	if Metadata.Labels != nil {
		retLabels := unpackLabels(Metadata.Labels)
		labelsAnnotsStruct["labels"] = retLabels
	}

	retAlertMetaData = append(retAlertMetaData, labelsAnnotsStruct)

	return retAlertMetaData
}

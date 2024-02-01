package slo

import (
	"context"

	slo "github.com/grafana/slo-openapi-client/go"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceSlo() *schema.Resource {
	return &schema.Resource{
		Description: `
Datasource for retrieving all SLOs.
		
* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/api/)
* [Additional Information On Alerting Rule Annotations and Labels](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#templating/)
				`,
		ReadContext: datasourceSloRead,
		Schema: map[string]*schema.Schema{
			"slos": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `Returns a list of all SLOs"`,
				Elem: &schema.Resource{
					Schema: common.CloneResourceSchemaForDatasource(ResourceSlo(), map[string]*schema.Schema{
						"uuid": {
							Type:        schema.TypeString,
							Description: `A unique, random identifier. This value will also be the name of the resource stored in the API server. This value is read-only.`,
							Computed:    true,
						},
					}),
				},
			},
		},
	}
}

// Function sends a GET request to the SLO API Endpoint which returns a list of all SLOs
// Maps the API Response body to the Terraform Schema and displays as a READ in the terminal
func datasourceSloRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := m.(*common.Client).SLOClient
	req := client.DefaultAPI.V1SloGet(ctx)
	apiSlos, _, err := req.Execute()
	if err != nil {
		return apiError("Could not retrieve SLOs", err)
	}

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

func convertDatasourceSlo(slo slo.Slo) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["uuid"] = slo.Uuid
	ret["name"] = slo.Name
	ret["description"] = slo.Description

	ret["query"] = unpackQuery(slo.Query)

	ret["destination_datasource"] = unpackDestinationDatasource(slo.DestinationDatasource)

	retLabels := unpackLabels(slo.Labels)
	ret["label"] = retLabels

	retObjectives := unpackObjectives(slo.Objectives)
	ret["objectives"] = retObjectives

	retAlerting := unpackAlerting(slo.Alerting)
	ret["alerting"] = retAlerting

	return ret
}

func unpackQuery(apiquery slo.Query) []map[string]interface{} {
	retQuery := []map[string]interface{}{}

	if apiquery.Type == QueryTypeFreeform {
		query := map[string]interface{}{"type": QueryTypeFreeform}

		freeformquerystring := map[string]interface{}{"query": apiquery.Freeform.Query}
		freeform := []map[string]interface{}{}
		freeform = append(freeform, freeformquerystring)
		query["freeform"] = freeform

		retQuery = append(retQuery, query)
	}

	if apiquery.Type == QueryTypeRatio {
		query := map[string]interface{}{"type": QueryTypeRatio}

		ratio := []map[string]interface{}{}
		body := map[string]interface{}{
			"success_metric":  apiquery.Ratio.SuccessMetric.PrometheusMetric,
			"total_metric":    apiquery.Ratio.TotalMetric.PrometheusMetric,
			"group_by_labels": apiquery.Ratio.GroupByLabels,
		}

		ratio = append(ratio, body)
		query["ratio"] = ratio

		retQuery = append(retQuery, query)
	}

	return retQuery
}

func unpackObjectives(objectives []slo.Objective) []map[string]interface{} {
	retObjectives := []map[string]interface{}{}

	for _, objective := range objectives {
		retObjective := make(map[string]interface{})
		retObjective["value"] = objective.Value
		retObjective["window"] = objective.Window
		retObjectives = append(retObjectives, retObjective)
	}

	return retObjectives
}

func unpackLabels(labelsInterface interface{}) []map[string]interface{} {
	retLabels := []map[string]interface{}{}

	var labels []slo.Label
	switch v := labelsInterface.(type) {
	case *[]slo.Label:
		labels = *v
	case []slo.Label:
		labels = v
	case []interface{}:
		for _, labelInterface := range v {
			switch v := labelInterface.(type) {
			case map[string]interface{}:
				label := slo.Label{
					Key:   v["key"].(string),
					Value: v["value"].(string),
				}
				labels = append(labels, label)
			case slo.Label:
				labels = append(labels, v)
			}
		}
	}

	for _, label := range labels {
		retLabel := make(map[string]interface{})
		retLabel["key"] = label.Key
		retLabel["value"] = label.Value
		retLabels = append(retLabels, retLabel)
	}
	return retLabels
}

func unpackAlerting(alertData *slo.Alerting) []map[string]interface{} {
	retAlertData := []map[string]interface{}{}

	if alertData == nil {
		return retAlertData
	}

	alertObject := make(map[string]interface{})
	alertObject["label"] = unpackLabels(alertData.Labels)
	alertObject["annotation"] = unpackLabels(alertData.Annotations)

	if alertData.FastBurn != nil {
		alertObject["fastburn"] = unpackAlertingMetadata(*alertData.FastBurn)
	}

	if alertData.SlowBurn != nil {
		alertObject["slowburn"] = unpackAlertingMetadata(*alertData.SlowBurn)
	}

	retAlertData = append(retAlertData, alertObject)
	return retAlertData
}

func unpackAlertingMetadata(metaData slo.AlertingMetadata) []map[string]interface{} {
	retAlertMetaData := []map[string]interface{}{}
	labelsAnnotsStruct := make(map[string]interface{})

	if metaData.Annotations != nil {
		retAnnotations := unpackLabels(metaData.Annotations)
		labelsAnnotsStruct["annotation"] = retAnnotations
	}

	if metaData.Labels != nil {
		retLabels := unpackLabels(metaData.Labels)
		labelsAnnotsStruct["label"] = retLabels
	}

	retAlertMetaData = append(retAlertMetaData, labelsAnnotsStruct)
	return retAlertMetaData
}

func unpackDestinationDatasource(destinationDatasource *slo.DestinationDatasource) []map[string]interface{} {
	retDestinationDatasources := []map[string]interface{}{}

	retDestinationDatasource := make(map[string]interface{})
	retDestinationDatasource["uid"] = destinationDatasource.Uid

	retDestinationDatasources = append(retDestinationDatasources, retDestinationDatasource)

	return retDestinationDatasources
}

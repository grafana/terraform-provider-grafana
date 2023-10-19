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
		
* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/api/)
* [Additional Information On Alerting Rule Annotations and Labels](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#templating/)
				`,
		ReadContext: datasourceSloRead,
		Schema: map[string]*schema.Schema{
			"slos": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: `Returns a list of all SLOs"`,
				Elem: &schema.Resource{
					Schema: common.CloneResourceSchemaForDatasource(ResourceSlo(), map[string]*schema.Schema{
						"uuid": &schema.Schema{
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

	ret["query"] = unpackQuery(slo.Query)

	retLabels := unpackLabels(&slo.Labels)
	ret["label"] = retLabels

	retObjectives := unpackObjectives(slo.Objectives)
	ret["objectives"] = retObjectives

	retAlerting := unpackAlerting(slo.Alerting)
	ret["alerting"] = retAlerting

	return ret
}

func unpackQuery(apiquery gapi.Query) []map[string]interface{} {
	retQuery := []map[string]interface{}{}

	if apiquery.Type == gapi.QueryTypeFreeform {
		query := map[string]interface{}{"type": gapi.QueryTypeFreeform}

		freeformquerystring := map[string]interface{}{"query": apiquery.Freeform.Query}
		freeform := []map[string]interface{}{}
		freeform = append(freeform, freeformquerystring)
		query["freeform"] = freeform

		retQuery = append(retQuery, query)
	}

	if apiquery.Type == gapi.QueryTypeRatio {
		query := map[string]interface{}{"type": gapi.QueryTypeRatio}

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
	alertObject["label"] = unpackLabels(&alertData.Labels)
	alertObject["annotation"] = unpackLabels(&alertData.Annotations)

	if alertData.FastBurn != nil {
		alertObject["fastburn"] = unpackAlertingMetadata(*alertData.FastBurn)
	}

	if alertData.SlowBurn != nil {
		alertObject["slowburn"] = unpackAlertingMetadata(*alertData.SlowBurn)
	}

	retAlertData = append(retAlertData, alertObject)
	return retAlertData
}

func unpackAlertingMetadata(metaData gapi.AlertingMetadata) []map[string]interface{} {
	retAlertMetaData := []map[string]interface{}{}
	labelsAnnotsStruct := make(map[string]interface{})

	if metaData.Annotations != nil {
		retAnnotations := unpackLabels(&metaData.Annotations)
		labelsAnnotsStruct["annotation"] = retAnnotations
	}

	if metaData.Labels != nil {
		retLabels := unpackLabels(&metaData.Labels)
		labelsAnnotsStruct["label"] = retLabels
	}

	retAlertMetaData = append(retAlertMetaData, labelsAnnotsStruct)
	return retAlertMetaData
}

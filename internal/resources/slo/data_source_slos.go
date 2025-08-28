package slo

import (
	"context"
	"encoding/json"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceSlo() *common.DataSource {
	schema := &schema.Resource{
		Description: `
Datasource for retrieving all SLOs.
		
* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/api/)
* [Additional Information On Alerting Rule Annotations and Labels](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#templating/)
				`,
		ReadContext: withClient[schema.ReadContextFunc](datasourceSloRead),
		Schema: map[string]*schema.Schema{
			"slos": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `Returns a list of all SLOs"`,
				Elem: &schema.Resource{
					Schema: common.CloneResourceSchemaForDatasource(resourceSlo().Schema, map[string]*schema.Schema{
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
	return common.NewLegacySDKDataSource(common.CategorySLO, "grafana_slos", schema)
}

// Function sends a GET request to the SLO API Endpoint which returns a list of all SLOs
// Maps the API Response body to the Terraform Schema and displays as a READ in the terminal
func datasourceSloRead(ctx context.Context, d *schema.ResourceData, client *slo.APIClient) diag.Diagnostics {
	var diags diag.Diagnostics

	req := client.DefaultAPI.V1SloGet(ctx)
	apiSlos, _, err := req.Execute()
	if err != nil {
		return apiError("Could not retrieve SLOs", err)
	}

	terraformSlos := []any{}

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

func convertDatasourceSlo(slo slo.SloV00Slo) map[string]any {
	ret := make(map[string]any)

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
	ret["search_expression"] = slo.SearchExpression

	return ret
}

func unpackQuery(apiquery slo.SloV00Query) []map[string]any {
	retQuery := []map[string]any{}

	if apiquery.Type == QueryTypeFreeform {
		query := map[string]any{"type": QueryTypeFreeform}

		freeformquerystring := map[string]any{"query": apiquery.Freeform.Query}
		freeform := []map[string]any{}
		freeform = append(freeform, freeformquerystring)
		query["freeform"] = freeform

		retQuery = append(retQuery, query)
	}

	if apiquery.Type == QueryTypeRatio {
		query := map[string]any{"type": QueryTypeRatio}

		ratio := []map[string]any{}
		body := map[string]any{
			"success_metric":  apiquery.Ratio.SuccessMetric.PrometheusMetric,
			"total_metric":    apiquery.Ratio.TotalMetric.PrometheusMetric,
			"group_by_labels": apiquery.Ratio.GroupByLabels,
		}

		ratio = append(ratio, body)
		query["ratio"] = ratio

		retQuery = append(retQuery, query)
	}

	if apiquery.Type == QueryTypeGrafanaQueries {
		query := map[string]any{"type": "grafana_queries"}

		grafanaQueries := []map[string]any{}
		queryString, _ := json.Marshal(apiquery.GrafanaQueries.GetGrafanaQueries())
		grafanaQueriesString := map[string]any{"grafana_queries": string(queryString)}
		grafanaQueries = append(grafanaQueries, grafanaQueriesString)
		query["grafana_queries"] = grafanaQueries
		retQuery = append(retQuery, query)
	}

	return retQuery
}

func unpackObjectives(objectives []slo.SloV00Objective) []map[string]any {
	retObjectives := []map[string]any{}

	for _, objective := range objectives {
		retObjective := make(map[string]any)
		retObjective["value"] = objective.Value
		retObjective["window"] = objective.Window
		retObjectives = append(retObjectives, retObjective)
	}

	return retObjectives
}

func unpackLabels(labelsInterface any) []map[string]any {
	retLabels := []map[string]any{}

	var labels []slo.SloV00Label
	switch v := labelsInterface.(type) {
	case *[]slo.SloV00Label:
		labels = *v
	case []slo.SloV00Label:
		labels = v
	case []any:
		for _, labelInterface := range v {
			switch v := labelInterface.(type) {
			case map[string]any:
				label := slo.SloV00Label{
					Key:   v["key"].(string),
					Value: v["value"].(string),
				}
				labels = append(labels, label)
			case slo.SloV00Label:
				labels = append(labels, v)
			}
		}
	}

	for _, label := range labels {
		retLabel := make(map[string]any)
		retLabel["key"] = label.Key
		retLabel["value"] = label.Value
		retLabels = append(retLabels, retLabel)
	}
	return retLabels
}

func unpackAlerting(alertData *slo.SloV00Alerting) []map[string]any {
	retAlertData := []map[string]any{}

	if alertData == nil {
		return retAlertData
	}

	alertObject := make(map[string]any)
	alertObject["label"] = unpackLabels(alertData.Labels)
	alertObject["annotation"] = unpackLabels(alertData.Annotations)

	if alertData.FastBurn != nil {
		alertObject["fastburn"] = unpackAlertingMetadata(*alertData.FastBurn)
	}

	if alertData.SlowBurn != nil {
		alertObject["slowburn"] = unpackAlertingMetadata(*alertData.SlowBurn)
	}

	if alertData.AdvancedOptions != nil {
		alertObject["advanced_options"] = unpackAdvancedOptions(*alertData.AdvancedOptions)
	}

	retAlertData = append(retAlertData, alertObject)
	return retAlertData
}

func unpackAlertingMetadata(metaData slo.SloV00AlertingMetadata) []map[string]any {
	retAlertMetaData := []map[string]any{}
	labelsAnnotsStruct := make(map[string]any)

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

func unpackAdvancedOptions(options slo.SloV00AdvancedOptions) []map[string]any {
	retAdvancedOptions := []map[string]any{}
	minFailuresStruct := make(map[string]any)

	if options.MinFailures != nil {
		minFailuresStruct["min_failures"] = int(*options.MinFailures)
	}

	retAdvancedOptions = append(retAdvancedOptions, minFailuresStruct)
	return retAdvancedOptions
}

func unpackDestinationDatasource(destinationDatasource *slo.SloV00DestinationDatasource) []map[string]any {
	retDestinationDatasources := []map[string]any{}

	retDestinationDatasource := make(map[string]any)
	retDestinationDatasource["uid"] = destinationDatasource.Uid

	retDestinationDatasources = append(retDestinationDatasources, retDestinationDatasource)

	return retDestinationDatasources
}

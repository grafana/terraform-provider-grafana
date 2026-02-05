package slo

import (
	"encoding/json"

	"github.com/grafana/slo-openapi-client/go/slo"
)

// These utility functions are used by the SDKv2 resource (resource_slo.go)
// They convert API responses to the map[string]any format expected by SDKv2

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

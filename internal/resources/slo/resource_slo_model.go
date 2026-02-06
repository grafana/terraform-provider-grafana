package slo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// sloResourceModel represents the resource model for an SLO
// It extends sloItemModel with additional resource-specific fields
type sloResourceModel struct {
	ID                    types.String                      `tfsdk:"id"`
	UUID                  types.String                      `tfsdk:"uuid"`
	Name                  types.String                      `tfsdk:"name"`
	Description           types.String                      `tfsdk:"description"`
	FolderUID             types.String                      `tfsdk:"folder_uid"`
	Query                 []queryModel                      `tfsdk:"query"`
	DestinationDatasource []destinationDatasourceModel      `tfsdk:"destination_datasource"`
	Label                 []labelModel                      `tfsdk:"label"`
	Objectives            []objectiveModel                  `tfsdk:"objectives"`
	Alerting              []alertingModel                   `tfsdk:"alerting"`
	SearchExpression      types.String                      `tfsdk:"search_expression"`
}

// packSloResourceModel converts the Terraform model to an API model
func packSloResourceModel(ctx context.Context, model *sloResourceModel) (slo.SloV00Slo, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Use custom UUID if set, otherwise empty string (API will generate)
	tfUUID := ""
	if !model.UUID.IsNull() && model.UUID.ValueString() != "" {
		tfUUID = model.UUID.ValueString()
	}

	// Pack query
	if len(model.Query) == 0 {
		diags.AddError("Missing required field", "query is required")
		return slo.SloV00Slo{}, diags
	}
	tfquery, queryDiags := packQuery(ctx, model.Query[0])
	diags.Append(queryDiags...)

	// Pack objectives
	tfobjectives := packObjectives(model.Objectives)

	// Pack labels
	var tflabels []slo.SloV00Label
	if len(model.Label) > 0 {
		tflabels = packLabels(model.Label)
	}

	apiSlo := slo.SloV00Slo{
		Uuid:                  tfUUID,
		Name:                  model.Name.ValueString(),
		Description:           model.Description.ValueString(),
		Objectives:            tfobjectives,
		Query:                 tfquery,
		Labels:                tflabels,
		Alerting:              nil,
		DestinationDatasource: nil,
	}

	// Pack search expression
	if !model.SearchExpression.IsNull() && model.SearchExpression.ValueString() != "" {
		apiSlo.SearchExpression = common.Ref(model.SearchExpression.ValueString())
	}

	// Pack alerting
	if len(model.Alerting) > 0 {
		tfalerting := packAlerting(model.Alerting[0])
		apiSlo.Alerting = &tfalerting
	}

	// Pack destination datasource
	if len(model.DestinationDatasource) > 0 {
		tfdestinationdatasource := packDestinationDatasource(model.DestinationDatasource[0])
		apiSlo.DestinationDatasource = &tfdestinationdatasource
	}

	// Pack folder
	if !model.FolderUID.IsNull() && model.FolderUID.ValueString() != "" {
		tffolder := packFolder(model.FolderUID.ValueString())
		apiSlo.Folder = &tffolder
	}

	return apiSlo, diags
}

func packQuery(ctx context.Context, query queryModel) (slo.SloV00Query, diag.Diagnostics) {
	var diags diag.Diagnostics

	queryType := query.Type.ValueString()

	switch queryType {
	case "freeform":
		if len(query.Freeform) == 0 {
			diags.AddError("Invalid query", "freeform query is required when type is freeform")
			return slo.SloV00Query{}, diags
		}
		querystring := query.Freeform[0].Query.ValueString()

		return slo.SloV00Query{
			Freeform: &slo.SloV00FreeformQuery{Query: querystring},
			Type:     QueryTypeFreeform,
		}, diags

	case "ratio":
		if len(query.Ratio) == 0 {
			diags.AddError("Invalid query", "ratio query is required when type is ratio")
			return slo.SloV00Query{}, diags
		}
		ratioQuery := query.Ratio[0]
		successMetric := ratioQuery.SuccessMetric.ValueString()
		totalMetric := ratioQuery.TotalMetric.ValueString()

		var labels []string
		if !ratioQuery.GroupByLabels.IsNull() {
			labelsDiags := ratioQuery.GroupByLabels.ElementsAs(ctx, &labels, false)
			diags.Append(labelsDiags...)
		}

		return slo.SloV00Query{
			Ratio: &slo.SloV00RatioQuery{
				SuccessMetric: slo.SloV00MetricDef{
					PrometheusMetric: successMetric,
				},
				TotalMetric: slo.SloV00MetricDef{
					PrometheusMetric: totalMetric,
				},
				GroupByLabels: labels,
			},
			Type: QueryTypeRatio,
		}, diags

	case "grafana_queries":
		if len(query.GrafanaQueries) == 0 {
			diags.AddError("Invalid query", "grafana_queries is required when type is grafana_queries")
			return slo.SloV00Query{}, diags
		}

		querystring := query.GrafanaQueries[0].GrafanaQueries.ValueString()

		// Validate the JSON
		if err := ValidateGrafanaQuery(querystring); err != nil {
			diags.AddError("Invalid grafana_queries", err.Error())
			return slo.SloV00Query{}, diags
		}

		var queryMapList []map[string]any
		err := json.Unmarshal([]byte(querystring), &queryMapList)
		if err != nil {
			diags.AddError("Failed to parse grafana_queries", err.Error())
			return slo.SloV00Query{}, diags
		}

		return slo.SloV00Query{
			GrafanaQueries: &slo.SloV00GrafanaQueries{GrafanaQueries: queryMapList},
			Type:           QueryTypeGrafanaQueries,
		}, diags

	default:
		diags.AddError("Unsupported query type", fmt.Sprintf("query type '%s' not implemented", queryType))
		return slo.SloV00Query{}, diags
	}
}

func packObjectives(objectives []objectiveModel) []slo.SloV00Objective {
	apiObjectives := []slo.SloV00Objective{}

	for _, obj := range objectives {
		// Validate window
		if err := ValidatePrometheusWindow(obj.Window.ValueString()); err != nil {
			// Log warning but continue - API will validate
			continue
		}

		apiObjectives = append(apiObjectives, slo.SloV00Objective{
			Value:  obj.Value.ValueFloat64(),
			Window: obj.Window.ValueString(),
		})
	}

	return apiObjectives
}

func packLabels(labels []labelModel) []slo.SloV00Label {
	apiLabels := []slo.SloV00Label{}

	for _, label := range labels {
		apiLabels = append(apiLabels, slo.SloV00Label{
			Key:   label.Key.ValueString(),
			Value: label.Value.ValueString(),
		})
	}

	return apiLabels
}

func packAlerting(alerting alertingModel) slo.SloV00Alerting {
	apiAlerting := slo.SloV00Alerting{
		Annotations: packLabels(alerting.Annotation),
		Labels:      packLabels(alerting.Label),
		FastBurn:    nil,
		SlowBurn:    nil,
	}

	// Pack FastBurn
	if len(alerting.FastBurn) > 0 {
		fastBurn := packAlertMetadata(alerting.FastBurn[0])
		apiAlerting.FastBurn = &fastBurn
	}

	// Pack SlowBurn
	if len(alerting.SlowBurn) > 0 {
		slowBurn := packAlertMetadata(alerting.SlowBurn[0])
		apiAlerting.SlowBurn = &slowBurn
	}

	// Pack AdvancedOptions
	if len(alerting.AdvancedOptions) > 0 && !alerting.AdvancedOptions[0].MinFailures.IsNull() {
		minFailures := alerting.AdvancedOptions[0].MinFailures.ValueInt64()
		apiAlerting.SetAdvancedOptions(slo.SloV00AdvancedOptions{
			MinFailures: &minFailures,
		})
	}

	return apiAlerting
}

func packAlertMetadata(metadata alertingMetadataModel) slo.SloV00AlertingMetadata {
	return slo.SloV00AlertingMetadata{
		Labels:      packLabels(metadata.Label),
		Annotations: packLabels(metadata.Annotation),
	}
}

func packDestinationDatasource(ds destinationDatasourceModel) slo.SloV00DestinationDatasource {
	uid := ds.UID.ValueString()
	return slo.SloV00DestinationDatasource{
		Uid: &uid,
	}
}

func packFolder(folderUID string) slo.SloV00Folder {
	return slo.SloV00Folder{
		Uid: &folderUID,
	}
}

// unpackSloToResourceModel converts an API SLO to a Terraform resource model
// It reuses the convert functions from data_source_slos.go where possible
func unpackSloToResourceModel(ctx context.Context, apiSlo *slo.SloV00Slo) (*sloResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := &sloResourceModel{
		ID:          types.StringValue(apiSlo.Uuid),
		UUID:        types.StringValue(apiSlo.Uuid),
		Name:        types.StringValue(apiSlo.Name),
		Description: types.StringValue(apiSlo.Description),
	}

	// Unpack search expression
	if apiSlo.SearchExpression != nil {
		model.SearchExpression = types.StringValue(*apiSlo.SearchExpression)
	} else {
		model.SearchExpression = types.StringNull()
	}

	// Unpack folder UID
	if apiSlo.Folder != nil && apiSlo.Folder.Uid != nil {
		model.FolderUID = types.StringValue(*apiSlo.Folder.Uid)
	} else {
		model.FolderUID = types.StringNull()
	}

	// Reuse convert functions from data_source_slos.go
	queryModels, queryDiags := convertQueryToModel(ctx, apiSlo.Query)
	diags.Append(queryDiags...)
	model.Query = queryModels

	if apiSlo.DestinationDatasource != nil {
		model.DestinationDatasource = convertDestinationDatasourceToModel(apiSlo.DestinationDatasource)
	}

	model.Label = convertLabelsToModel(apiSlo.Labels)
	model.Objectives = convertObjectivesToModel(apiSlo.Objectives)

	if apiSlo.Alerting != nil {
		model.Alerting = convertAlertingToModel(apiSlo.Alerting)
	}

	return model, diags
}

package slo

import (
	"context"
	"encoding/json"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const dataSourceSlosName = "grafana_slos"

var (
	_ datasource.DataSource              = &slosDataSource{}
	_ datasource.DataSourceWithConfigure = &slosDataSource{}
)

func datasourceSlo() *common.DataSource {
	return common.NewDataSource(
		common.CategorySLO,
		dataSourceSlosName,
		&slosDataSource{},
	)
}

type slosDataSource struct {
	basePluginFrameworkDataSource
}

func (d *slosDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceSlosName
}

func (d *slosDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Data source for retrieving all SLOs.

* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/api/)
* [Additional Information On Alerting Rule Annotations and Labels](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#templating/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this datasource. This is a constant value.",
			},
		},
		Blocks: map[string]schema.Block{
			"slos": schema.ListNestedBlock{
				MarkdownDescription: "List of all SLOs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed:    true,
							Description: "A unique, random identifier. This value is read-only.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the SLO.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "Description of the SLO.",
						},
						"search_expression": schema.StringAttribute{
							Computed:    true,
							Description: "The search expression associated with this SLO.",
						},
					},
					Blocks: map[string]schema.Block{
						"query": schema.ListNestedBlock{
							MarkdownDescription: "Query configuration for the SLO.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Computed:    true,
										Description: "Type of query (freeform, ratio, grafana_queries, etc.).",
									},
								},
								Blocks: map[string]schema.Block{
									"freeform": schema.ListNestedBlock{
										MarkdownDescription: "Freeform query configuration.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"query": schema.StringAttribute{
													Computed:    true,
													Description: "The PromQL query string.",
												},
											},
										},
									},
									"ratio": schema.ListNestedBlock{
										MarkdownDescription: "Ratio query configuration.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"success_metric": schema.StringAttribute{
													Computed:    true,
													Description: "Counter metric for success events (numerator).",
												},
												"total_metric": schema.StringAttribute{
													Computed:    true,
													Description: "Metric for total events (denominator).",
												},
												"group_by_labels": schema.ListAttribute{
													Computed:    true,
													Description: "Labels used for grouping.",
													ElementType: types.StringType,
												},
											},
										},
									},
									"grafana_queries": schema.ListNestedBlock{
										MarkdownDescription: "Grafana queries configuration.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"grafana_queries": schema.StringAttribute{
													Computed:    true,
													Description: "JSON string containing the Grafana queries.",
												},
											},
										},
									},
								},
							},
						},
						"destination_datasource": schema.ListNestedBlock{
							MarkdownDescription: "Destination datasource configuration.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"uid": schema.StringAttribute{
										Computed:    true,
										Description: "UID of the destination datasource.",
									},
								},
							},
						},
						"label": schema.ListNestedBlock{
							MarkdownDescription: "Labels attached to the SLO.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"key": schema.StringAttribute{
										Computed:    true,
										Description: "Label key.",
									},
									"value": schema.StringAttribute{
										Computed:    true,
										Description: "Label value.",
									},
								},
							},
						},
						"objectives": schema.ListNestedBlock{
							MarkdownDescription: "Objectives for the SLO.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"value": schema.Float64Attribute{
										Computed:    true,
										Description: "Objective value (between 0 and 1).",
									},
									"window": schema.StringAttribute{
										Computed:    true,
										Description: "Time window for the objective.",
									},
								},
							},
						},
						"alerting": schema.ListNestedBlock{
							MarkdownDescription: "Alerting configuration for the SLO.",
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"label": schema.ListNestedBlock{
										MarkdownDescription: "Labels attached to alerts.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"key": schema.StringAttribute{
													Computed:    true,
													Description: "Label key.",
												},
												"value": schema.StringAttribute{
													Computed:    true,
													Description: "Label value.",
												},
											},
										},
									},
									"annotation": schema.ListNestedBlock{
										MarkdownDescription: "Annotations attached to alerts.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"key": schema.StringAttribute{
													Computed:    true,
													Description: "Annotation key.",
												},
												"value": schema.StringAttribute{
													Computed:    true,
													Description: "Annotation value.",
												},
											},
										},
									},
									"fastburn": schema.ListNestedBlock{
										MarkdownDescription: "Fast burn alert configuration.",
										NestedObject: schema.NestedBlockObject{
											Blocks: map[string]schema.Block{
												"label": schema.ListNestedBlock{
													MarkdownDescription: "Labels for fast burn alerts.",
													NestedObject: schema.NestedBlockObject{
														Attributes: map[string]schema.Attribute{
															"key": schema.StringAttribute{
																Computed:    true,
																Description: "Label key.",
															},
															"value": schema.StringAttribute{
																Computed:    true,
																Description: "Label value.",
															},
														},
													},
												},
												"annotation": schema.ListNestedBlock{
													MarkdownDescription: "Annotations for fast burn alerts.",
													NestedObject: schema.NestedBlockObject{
														Attributes: map[string]schema.Attribute{
															"key": schema.StringAttribute{
																Computed:    true,
																Description: "Annotation key.",
															},
															"value": schema.StringAttribute{
																Computed:    true,
																Description: "Annotation value.",
															},
														},
													},
												},
											},
										},
									},
									"slowburn": schema.ListNestedBlock{
										MarkdownDescription: "Slow burn alert configuration.",
										NestedObject: schema.NestedBlockObject{
											Blocks: map[string]schema.Block{
												"label": schema.ListNestedBlock{
													MarkdownDescription: "Labels for slow burn alerts.",
													NestedObject: schema.NestedBlockObject{
														Attributes: map[string]schema.Attribute{
															"key": schema.StringAttribute{
																Computed:    true,
																Description: "Label key.",
															},
															"value": schema.StringAttribute{
																Computed:    true,
																Description: "Label value.",
															},
														},
													},
												},
												"annotation": schema.ListNestedBlock{
													MarkdownDescription: "Annotations for slow burn alerts.",
													NestedObject: schema.NestedBlockObject{
														Attributes: map[string]schema.Attribute{
															"key": schema.StringAttribute{
																Computed:    true,
																Description: "Annotation key.",
															},
															"value": schema.StringAttribute{
																Computed:    true,
																Description: "Annotation value.",
															},
														},
													},
												},
											},
										},
									},
									"advanced_options": schema.ListNestedBlock{
										MarkdownDescription: "Advanced alerting options.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"min_failures": schema.Int64Attribute{
													Computed:    true,
													Description: "Minimum number of failures before alerting.",
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

func (d *slosDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data slosDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all SLOs from the API
	request := d.client.DefaultAPI.V1SloGet(ctx)
	apiSlos, _, err := request.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to retrieve SLOs",
			"Could not retrieve SLOs: "+err.Error(),
		)
		return
	}

	// Convert API response to model
	data.ID = types.StringValue("slos")
	data.SLOs = []sloItemModel{}

	if len(apiSlos.Slos) > 0 {
		for _, apiSlo := range apiSlos.Slos {
			sloItem, diags := convertSloToItemModel(ctx, apiSlo)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.SLOs = append(data.SLOs, sloItem)
		}
	}

	// Save the data to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// slosDataSourceModel represents the data source model for all SLOs
type slosDataSourceModel struct {
	ID   types.String   `tfsdk:"id"`
	SLOs []sloItemModel `tfsdk:"slos"`
}

// sloItemModel represents a single SLO in the list
type sloItemModel struct {
	UUID                  types.String                 `tfsdk:"uuid"`
	Name                  types.String                 `tfsdk:"name"`
	Description           types.String                 `tfsdk:"description"`
	Query                 []queryModel                 `tfsdk:"query"`
	DestinationDatasource []destinationDatasourceModel `tfsdk:"destination_datasource"`
	Label                 []labelModel                 `tfsdk:"label"`
	Objectives            []objectiveModel             `tfsdk:"objectives"`
	Alerting              []alertingModel              `tfsdk:"alerting"`
	SearchExpression      types.String                 `tfsdk:"search_expression"`
}

type queryModel struct {
	Type           types.String          `tfsdk:"type"`
	Freeform       []freeformQueryModel  `tfsdk:"freeform"`
	Ratio          []ratioQueryModel     `tfsdk:"ratio"`
	GrafanaQueries []grafanaQueriesModel `tfsdk:"grafana_queries"`
}

type freeformQueryModel struct {
	Query types.String `tfsdk:"query"`
}

type ratioQueryModel struct {
	SuccessMetric types.String `tfsdk:"success_metric"`
	TotalMetric   types.String `tfsdk:"total_metric"`
	GroupByLabels types.List   `tfsdk:"group_by_labels"`
}

type grafanaQueriesModel struct {
	GrafanaQueries types.String `tfsdk:"grafana_queries"`
}

type destinationDatasourceModel struct {
	UID types.String `tfsdk:"uid"`
}

type labelModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type objectiveModel struct {
	Value  types.Float64 `tfsdk:"value"`
	Window types.String  `tfsdk:"window"`
}

type alertingModel struct {
	Label           []labelModel            `tfsdk:"label"`
	Annotation      []labelModel            `tfsdk:"annotation"`
	FastBurn        []alertingMetadataModel `tfsdk:"fastburn"`
	SlowBurn        []alertingMetadataModel `tfsdk:"slowburn"`
	AdvancedOptions []advancedOptionsModel  `tfsdk:"advanced_options"`
}

type alertingMetadataModel struct {
	Label      []labelModel `tfsdk:"label"`
	Annotation []labelModel `tfsdk:"annotation"`
}

type advancedOptionsModel struct {
	MinFailures types.Int64 `tfsdk:"min_failures"`
}

// convertSloToItemModel converts a single SLO API response to a model
func convertSloToItemModel(ctx context.Context, apiSlo slo.SloV00Slo) (sloItemModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	item := sloItemModel{
		UUID:        types.StringValue(apiSlo.Uuid),
		Name:        types.StringValue(apiSlo.Name),
		Description: types.StringValue(apiSlo.Description),
	}

	// Convert search expression
	if apiSlo.SearchExpression != nil {
		item.SearchExpression = types.StringValue(*apiSlo.SearchExpression)
	} else {
		item.SearchExpression = types.StringNull()
	}

	// Convert query
	queryModels, queryDiags := convertQueryToModel(ctx, apiSlo.Query)
	diags.Append(queryDiags...)
	item.Query = queryModels

	// Convert destination datasource
	if apiSlo.DestinationDatasource != nil {
		item.DestinationDatasource = convertDestinationDatasourceToModel(apiSlo.DestinationDatasource)
	}

	// Convert labels
	item.Label = convertLabelsToModel(apiSlo.Labels)

	// Convert objectives
	item.Objectives = convertObjectivesToModel(apiSlo.Objectives)

	// Convert alerting
	if apiSlo.Alerting != nil {
		item.Alerting = convertAlertingToModel(apiSlo.Alerting)
	}

	return item, diags
}

func convertQueryToModel(ctx context.Context, apiQuery slo.SloV00Query) ([]queryModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	queryModels := []queryModel{}

	query := queryModel{
		Type: types.StringValue(apiQuery.Type),
	}

	switch apiQuery.Type {
	case QueryTypeFreeform:
		if apiQuery.Freeform != nil {
			query.Freeform = []freeformQueryModel{
				{
					Query: types.StringValue(apiQuery.Freeform.Query),
				},
			}
		}

	case QueryTypeRatio:
		if apiQuery.Ratio != nil {
			groupByLabels, d := types.ListValueFrom(ctx, types.StringType, apiQuery.Ratio.GroupByLabels)
			diags.Append(d...)

			query.Ratio = []ratioQueryModel{
				{
					SuccessMetric: types.StringValue(apiQuery.Ratio.SuccessMetric.PrometheusMetric),
					TotalMetric:   types.StringValue(apiQuery.Ratio.TotalMetric.PrometheusMetric),
					GroupByLabels: groupByLabels,
				},
			}
		}

	case QueryTypeGrafanaQueries:
		if apiQuery.GrafanaQueries != nil {
			queryString, err := json.Marshal(apiQuery.GrafanaQueries.GetGrafanaQueries())
			if err != nil {
				diags.AddError("Failed to marshal Grafana queries", err.Error())
			} else {
				query.GrafanaQueries = []grafanaQueriesModel{
					{
						GrafanaQueries: types.StringValue(string(queryString)),
					},
				}
			}
		}
	}

	queryModels = append(queryModels, query)
	return queryModels, diags
}

func convertDestinationDatasourceToModel(apiDs *slo.SloV00DestinationDatasource) []destinationDatasourceModel {
	if apiDs == nil || apiDs.Uid == nil {
		return []destinationDatasourceModel{}
	}

	return []destinationDatasourceModel{
		{
			UID: types.StringValue(*apiDs.Uid),
		},
	}
}

func convertLabelsToModel(apiLabels []slo.SloV00Label) []labelModel {
	labels := []labelModel{}

	for _, apiLabel := range apiLabels {
		labels = append(labels, labelModel{
			Key:   types.StringValue(apiLabel.Key),
			Value: types.StringValue(apiLabel.Value),
		})
	}

	return labels
}

func convertObjectivesToModel(apiObjectives []slo.SloV00Objective) []objectiveModel {
	objectives := []objectiveModel{}

	for _, apiObjective := range apiObjectives {
		objectives = append(objectives, objectiveModel{
			Value:  types.Float64Value(apiObjective.Value),
			Window: types.StringValue(apiObjective.Window),
		})
	}

	return objectives
}

func convertAlertingToModel(apiAlerting *slo.SloV00Alerting) []alertingModel {
	if apiAlerting == nil {
		return []alertingModel{}
	}

	alerting := alertingModel{
		Label:      convertLabelsToModel(apiAlerting.Labels),
		Annotation: convertLabelsToModel(apiAlerting.Annotations),
	}

	// Convert FastBurn
	if apiAlerting.FastBurn != nil {
		alerting.FastBurn = []alertingMetadataModel{
			{
				Label:      convertLabelsToModel(apiAlerting.FastBurn.Labels),
				Annotation: convertLabelsToModel(apiAlerting.FastBurn.Annotations),
			},
		}
	}

	// Convert SlowBurn
	if apiAlerting.SlowBurn != nil {
		alerting.SlowBurn = []alertingMetadataModel{
			{
				Label:      convertLabelsToModel(apiAlerting.SlowBurn.Labels),
				Annotation: convertLabelsToModel(apiAlerting.SlowBurn.Annotations),
			},
		}
	}

	// Convert AdvancedOptions
	if apiAlerting.AdvancedOptions != nil && apiAlerting.AdvancedOptions.MinFailures != nil {
		alerting.AdvancedOptions = []advancedOptionsModel{
			{
				MinFailures: types.Int64Value(*apiAlerting.AdvancedOptions.MinFailures),
			},
		}
	}

	return []alertingModel{alerting}
}

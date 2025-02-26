package slo

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/grafana/slo-openapi-client/go/slo"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	QueryTypeFreeform       string = "freeform"
	QueryTypeHistogram      string = "histogram"
	QueryTypeRatio          string = "ratio"
	QueryTypeThreshold      string = "threshold"
	QueryTypeGrafanaQueries string = "grafanaQueries"
)

var resourceSloID = common.NewResourceID(common.StringIDField("uuid"))

func resourceSlo() *common.Resource {
	schema := &schema.Resource{
		Description: `
Resource manages Grafana SLOs. 

* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/)
* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/api/)
* [Additional Information On Alerting Rule Annotations and Labels](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#templating/)
		`,
		CreateContext: withClient[schema.CreateContextFunc](resourceSloCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceSloRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceSloUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceSloDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  `Name should be a short description of your indicator. Consider names like "API Availability"`,
				ValidateFunc: validation.StringLenBetween(0, 128),
			},
			"description": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  `Description is a free-text field that can provide more context to an SLO.`,
				ValidateFunc: validation.StringLenBetween(0, 1024),
			},
			"folder_uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: `UID for the SLO folder`,
			},
			"destination_datasource": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: `Destination Datasource sets the datasource defined for an SLO`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uid": {
							Type:        schema.TypeString,
							Description: `UID for the Datasource`,
							Required:    true,
						},
					},
				},
			},
			"query": {
				Type:        schema.TypeList,
				Required:    true,
				Description: `Query describes the indicator that will be measured against the objective. Freeform Query types are currently supported.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:         schema.TypeString,
							Description:  `Query type must be one of: "freeform", "query", "ratio", "grafana_queries" or "threshold"`,
							ValidateFunc: validation.StringInSlice([]string{"freeform", "query", "ratio", "threshold", "grafana_queries"}, false),
							Required:     true,
						},
						"freeform": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"query": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Freeform Query Field - valid promQl",
									},
								},
							},
						},
						"grafana_queries": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Optional:    true,
							Description: "Array for holding a set of grafana queries",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"grafana_queries": {
										Type:             schema.TypeString,
										Required:         true,
										Description:      "Query Object - Array of Grafana Query JSON objects",
										ValidateDiagFunc: ValidateGrafanaQuery(),
									},
								},
							},
						},
						"ratio": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"success_metric": {
										Type:        schema.TypeString,
										Description: `Counter metric for success events (numerator)`,
										Required:    true,
									},
									"total_metric": {
										Type:        schema.TypeString,
										Description: `Metric for total events (denominator)`,
										Required:    true,
									},
									"group_by_labels": {
										Type:        schema.TypeList,
										Description: `Defines Group By Labels used for per-label alerting. These appear as variables on SLO dashboards to enable filtering and aggregation. Labels must adhere to Prometheus label name schema - "^[a-zA-Z_][a-zA-Z0-9_]*$"`,
										Optional:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
			"label": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: `Additional labels that will be attached to all metrics generated from the query. These labels are useful for grouping SLOs in dashboard views that you create by hand. Labels must adhere to Prometheus label name schema - "^[a-zA-Z_][a-zA-Z0-9_]*$"`,
				Elem:        keyvalueSchema,
			},
			"objectives": {
				Type:        schema.TypeList,
				Required:    true,
				Description: `Over each rolling time window, the remaining error budget will be calculated, and separate alerts can be generated for each time window based on the SLO burn rate or remaining error budget.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": {
							Type:         schema.TypeFloat,
							Required:     true,
							ValidateFunc: validation.FloatBetween(0, 1),
							Description:  `Value between 0 and 1. If the value of the query is above the objective, the SLO is met.`,
						},
						"window": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  `A Prometheus-parsable time duration string like 24h, 60m. This is the time window the objective is measured over.`,
							ValidateFunc: validation.StringMatch(regexp.MustCompile(`^\d+(ms|s|m|h|d|w|y)$`), "Objective window must be a Prometheus-parsable time duration"),
						},
					},
				},
			},
			"alerting": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Description: `Configures the alerting rules that will be generated for each
				time window associated with the SLO. Grafana SLOs can generate
				alerts when the short-term error budget burn is very high, the
				long-term error budget burn rate is high, or when the remaining
				error budget is below a certain threshold. Annotations and Labels support templating.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: `Labels will be attached to all alerts generated by any of these rules.`,
							Elem:        keyvalueSchema,
						},
						"annotation": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: `Annotations will be attached to all alerts generated by any of these rules.`,
							Elem:        keyvalueSchema,
						},
						"fastburn": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Alerting Rules generated for Fast Burn alerts",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"label": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Labels to attach only to Fast Burn alerts.",
										Elem:        keyvalueSchema,
									},
									"annotation": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Annotations to attach only to Fast Burn alerts.",
										Elem:        keyvalueSchema,
									},
								},
							},
						},
						"slowburn": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Optional:    true,
							Description: "Alerting Rules generated for Slow Burn alerts",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"label": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Labels to attach only to Slow Burn alerts.",
										Elem:        keyvalueSchema,
									},
									"annotation": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Annotations to attach only to Slow Burn alerts.",
										Elem:        keyvalueSchema,
									},
								},
							},
						},
						"advanced_options": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Optional:    true,
							Description: "Advanced Options for Alert Rules",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"min_failures": {
										Type:         schema.TypeInt,
										Optional:     true,
										Description:  "Minimum number of failed events to trigger an alert",
										ValidateFunc: validation.IntAtLeast(0),
									},
								},
							},
						},
					},
				},
			},
			"search_expression": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of a search expression in Grafana Asserts. This is used in the SLO UI to open the Asserts RCA workbench and in alerts to link to the RCA workbench.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategorySLO,
		"grafana_slo",
		resourceSloID,
		schema,
	).
		WithLister(listSlos).
		WithPreferredResourceNameField("name")
}

var keyvalueSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"key": {
			Type:        schema.TypeString,
			Required:    true,
			Description: `Key for filtering and identification`,
		},
		"value": {
			Type:        schema.TypeString,
			Required:    true,
			Description: `Templatable value`,
		},
	},
}

func listSlos(ctx context.Context, client *common.Client, data any) ([]string, error) {
	sloClient := client.SLOClient
	if sloClient == nil {
		return nil, fmt.Errorf("client not configured for SLO API")
	}

	slolist, _, err := sloClient.DefaultAPI.V1SloGet(ctx).Execute()
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, slo := range slolist.Slos {
		ids = append(ids, slo.Uuid)
	}
	return ids, nil
}

func resourceSloCreate(ctx context.Context, d *schema.ResourceData, client *slo.APIClient) diag.Diagnostics {
	var diags diag.Diagnostics

	sloModel, err := packSloResource(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to Pack SLO",
			Detail:   err.Error(),
		})
		return diags
	}

	req := client.DefaultAPI.V1SloPost(ctx).SloV00Slo(sloModel)
	response, _, err := req.Execute()

	if err != nil {
		return apiError("Unable to create SLO - API", err)
	}

	d.SetId(response.Uuid)

	return resourceSloRead(ctx, d, client)
}

// resourceSloRead - sends a GET Request to the single SLO Endpoint
func resourceSloRead(ctx context.Context, d *schema.ResourceData, client *slo.APIClient) diag.Diagnostics {
	var diags diag.Diagnostics

	sloID := d.Id()

	req := client.DefaultAPI.V1SloIdGet(ctx, sloID)
	slo, r, err := req.Execute()
	if err != nil {
		if r != nil && r.StatusCode == 404 {
			return common.WarnMissing("SLO", d)
		}
		return apiError("Unable to read SLO - API", err)
	}

	setTerraformState(d, *slo)

	return diags
}

func resourceSloUpdate(ctx context.Context, d *schema.ResourceData, client *slo.APIClient) diag.Diagnostics {
	var diags diag.Diagnostics
	sloID := d.Id()

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("query") || d.HasChange("label") || d.HasChange("objectives") || d.HasChange("alerting") || d.HasChange("destination_datasource") || d.HasChange("folder_uid") {
		sloV00Slo, err := packSloResource(d)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to Pack SLO",
				Detail:   err.Error(),
			})
			return diags
		}

		req := client.DefaultAPI.V1SloIdPut(ctx, sloID).SloV00Slo(sloV00Slo)
		if _, err := req.Execute(); err != nil {
			return apiError("Unable to Update SLO - API", err)
		}
	}

	return resourceSloRead(ctx, d, client)
}

func resourceSloDelete(ctx context.Context, d *schema.ResourceData, client *slo.APIClient) diag.Diagnostics {
	sloID := d.Id()

	req := client.DefaultAPI.V1SloIdDelete(ctx, sloID)
	_, err := req.Execute()

	return apiError("Unable to Delete SLO - API", err)
}

// Fetches all the Properties defined on the Terraform SLO State Object and converts it
// to a Slo so that it can be converted to JSON and sent to the API
func packSloResource(d *schema.ResourceData) (slo.SloV00Slo, error) {
	var (
		tfalerting              slo.SloV00Alerting
		tflabels                []slo.SloV00Label
		tfdestinationdatasource slo.SloV00DestinationDatasource
		tffolder                slo.SloV00Folder
	)

	tfname := d.Get("name").(string)
	tfdescription := d.Get("description").(string)
	query := d.Get("query").([]interface{})[0].(map[string]interface{})
	tfquery, err := packQuery(query)
	if err != nil {
		return slo.SloV00Slo{}, err
	}

	objectives := d.Get("objectives").([]interface{})
	tfobjective := packObjectives(objectives)

	labels := d.Get("label").([]interface{})
	if labels != nil {
		tflabels = packLabels(labels)
	}

	req := slo.SloV00Slo{
		Uuid:                  d.Id(),
		Name:                  tfname,
		Description:           tfdescription,
		Objectives:            tfobjective,
		Query:                 tfquery,
		Alerting:              nil,
		Labels:                tflabels,
		DestinationDatasource: nil,
	}

	// Check the Optional Search Expression Field
	if searchexpression, ok := d.GetOk("search_expression"); ok && searchexpression != "" {
		req.SearchExpression = common.Ref(searchexpression.(string))
	}

	// Check the Optional Alerting Field
	if alerting, ok := d.GetOk("alerting"); ok {
		alertData, ok := alerting.([]interface{})
		if ok && len(alertData) > 0 {
			alert, ok := alertData[0].(map[string]interface{})
			if ok {
				tfalerting = packAlerting(alert)
			}
		}
		req.Alerting = &tfalerting
	}

	// Check the Optional Destination Datasource Field
	if rawdestinationdatasource, ok := d.GetOk("destination_datasource"); ok {
		destinationDatasourceData, ok := rawdestinationdatasource.([]interface{})

		if ok && len(destinationDatasourceData) > 0 {
			destinationdatasource := destinationDatasourceData[0].(map[string]interface{})
			tfdestinationdatasource, _ = packDestinationDatasource(destinationdatasource)
		}

		req.DestinationDatasource = &tfdestinationdatasource
	}

	// Check the Optional Folder UID Field
	if rawfolderuid, ok := d.GetOk("folder_uid"); ok {
		folderUIDData, ok := rawfolderuid.(string)

		if ok {
			tffolder = packFolder(folderUIDData)
		}

		req.Folder = &tffolder
	}

	return req, nil
}

func packDestinationDatasource(destinationdatasource map[string]interface{}) (slo.SloV00DestinationDatasource, error) {
	packedDestinationDatasource := slo.SloV00DestinationDatasource{}

	if destinationdatasource["uid"].(string) != "" {
		datasourceUID := destinationdatasource["uid"].(string)
		packedDestinationDatasource.Uid = common.Ref(datasourceUID)
	}

	return packedDestinationDatasource, nil
}

func packFolder(folderuid string) slo.SloV00Folder {
	return slo.SloV00Folder{
		Uid: &folderuid,
	}
}

func packQuery(query map[string]interface{}) (slo.SloV00Query, error) {
	if query["type"] == "freeform" {
		freeformquery := query["freeform"].([]interface{})[0].(map[string]interface{})
		querystring := freeformquery["query"].(string)

		sloQuery := slo.SloV00Query{
			Freeform: &slo.SloV00FreeformQuery{Query: querystring},
			Type:     QueryTypeFreeform,
		}

		return sloQuery, nil
	}

	if query["type"] == "ratio" {
		ratioquery := query["ratio"].([]interface{})[0].(map[string]interface{})
		successMetric := ratioquery["success_metric"].(string)
		totalMetric := ratioquery["total_metric"].(string)
		groupByLabels := ratioquery["group_by_labels"].([]interface{})

		var labels []string

		for ind := range groupByLabels {
			if groupByLabels[ind] == nil {
				labels = append(labels, "")
				continue
			}
			labels = append(labels, groupByLabels[ind].(string))
		}

		sloQuery := slo.SloV00Query{
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
		}

		return sloQuery, nil
	}

	if query["type"] == "grafana_queries" {
		// This is safe
		grafanaInterface := query["grafana_queries"].([]interface{})

		if len(grafanaInterface) == 0 {
			return slo.SloV00Query{}, fmt.Errorf("grafana_queries must be set")
		}

		grafanaquery := grafanaInterface[0].(map[string]interface{})
		querystring := grafanaquery["grafana_queries"].(string)

		var queryMapList []map[string]interface{}
		err := json.Unmarshal([]byte(querystring), &queryMapList)

		// We validate the JSON structure this should never occur
		if err != nil {
			return slo.SloV00Query{}, err
		}

		sloQuery := slo.SloV00Query{
			GrafanaQueries: &slo.SloV00GrafanaQueries{GrafanaQueries: queryMapList},
			Type:           QueryTypeGrafanaQueries,
		}

		return sloQuery, nil
	}

	return slo.SloV00Query{}, fmt.Errorf("%s query type not implemented", query["type"])
}

func packObjectives(tfobjectives []interface{}) []slo.SloV00Objective {
	objectives := []slo.SloV00Objective{}

	for ind := range tfobjectives {
		tfobjective := tfobjectives[ind].(map[string]interface{})
		objective := slo.SloV00Objective{
			Value:  tfobjective["value"].(float64),
			Window: tfobjective["window"].(string),
		}
		objectives = append(objectives, objective)
	}

	return objectives
}

func packLabels(tfLabels []interface{}) []slo.SloV00Label {
	labelSlice := []slo.SloV00Label{}

	for ind := range tfLabels {
		currLabel := tfLabels[ind].(map[string]interface{})
		curr := slo.SloV00Label{
			Key:   currLabel["key"].(string),
			Value: currLabel["value"].(string),
		}

		labelSlice = append(labelSlice, curr)
	}

	return labelSlice
}

func packAlerting(tfAlerting map[string]interface{}) slo.SloV00Alerting {
	var tfAnnots []slo.SloV00Label
	var tfLabels []slo.SloV00Label
	var tfFastBurn slo.SloV00AlertingMetadata
	var tfSlowBurn slo.SloV00AlertingMetadata
	var tfAdvancedOptions slo.SloV00AdvancedOptions

	annots, ok := tfAlerting["annotation"].([]interface{})
	if ok {
		tfAnnots = packLabels(annots)
	}

	labels, ok := tfAlerting["label"].([]interface{})
	if ok {
		tfLabels = packLabels(labels)
	}

	fastBurn, ok := tfAlerting["fastburn"].([]interface{})
	if ok {
		tfFastBurn = packAlertMetadata(fastBurn)
	}

	slowBurn, ok := tfAlerting["slowburn"].([]interface{})
	if ok {
		tfSlowBurn = packAlertMetadata(slowBurn)
	}

	alerting := slo.SloV00Alerting{
		Annotations: tfAnnots,
		Labels:      tfLabels,
		FastBurn:    &tfFastBurn,
		SlowBurn:    &tfSlowBurn,
	}

	// All options in advanced options will be optional
	// Adding a second feature will need to make a better way of checking what is there
	if failures := tfAlerting["advanced_options"]; failures != nil {
		lf, ok := failures.([]interface{})
		if ok && len(lf) > 0 {
			lf2, ok := lf[0].(map[string]interface{})
			if ok {
				i64 := int64(lf2["min_failures"].(int))
				tfAdvancedOptions = slo.SloV00AdvancedOptions{
					MinFailures: &i64,
				}
				alerting.SetAdvancedOptions(tfAdvancedOptions)
			}
		}
	}

	return alerting
}

func packAlertMetadata(metadata []interface{}) slo.SloV00AlertingMetadata {
	var tflabels []slo.SloV00Label
	var tfannots []slo.SloV00Label

	if len(metadata) > 0 {
		meta, ok := metadata[0].(map[string]interface{})
		if ok {
			labels, ok := meta["label"].([]interface{})
			if ok {
				tflabels = packLabels(labels)
			}

			annots, ok := meta["annotation"].([]interface{})
			if ok {
				tfannots = packLabels(annots)
			}
		}
	}

	apiMetadata := slo.SloV00AlertingMetadata{
		Labels:      tflabels,
		Annotations: tfannots,
	}

	return apiMetadata
}

func setTerraformState(d *schema.ResourceData, slo slo.SloV00Slo) {
	d.Set("name", slo.Name)
	d.Set("description", slo.Description)

	d.Set("query", unpackQuery(slo.Query))

	retLabels := unpackLabels(&slo.Labels)
	d.Set("label", retLabels)

	retDestinationDatasource := unpackDestinationDatasource(slo.DestinationDatasource)
	d.Set("destination_datasource", retDestinationDatasource)

	retObjectives := unpackObjectives(slo.Objectives)
	d.Set("objectives", retObjectives)

	retAlerting := unpackAlerting(slo.Alerting)
	d.Set("alerting", retAlerting)
	d.Set("search_expression", slo.SearchExpression)
}

func apiError(action string, err error) diag.Diagnostics {
	if err == nil {
		return nil
	}
	detail := err.Error()
	if err, ok := err.(*slo.GenericOpenAPIError); ok {
		detail += "\n" + string(err.Body())
	}
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Error,
			Summary:  action,
			Detail:   detail,
		},
	}
}

func ValidateGrafanaQuery() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics

		v, ok := i.(string)
		if !ok {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Bad Format",
				Detail:        fmt.Sprintf("expected type of %s to be string", path),
				AttributePath: path,
			})
			return diags
		}

		var gmrQuery []map[string]any
		err := json.Unmarshal([]byte(v), &gmrQuery)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Bad Format",
				Detail:        "expected grafana queries to be valid JSON format",
				AttributePath: path,
			})
			return diags
		}

		if len(gmrQuery) == 0 {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Missing Required Field",
				Detail:        "expected grafana queries to have at least one query",
				AttributePath: path,
			})
			return diags
		}

		for _, queryObj := range gmrQuery {
			currentPath := path.Copy()

			refID, ok := queryObj["refId"]
			if !ok {
				diags = append(diags, diag.Diagnostic{
					Severity:      diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query %v to have a refId", queryObj),
					AttributePath: append(currentPath, cty.IndexStep{Key: cty.StringVal("refId")}),
				})
				return diags
			}

			source := queryObj["datasource"]
			s, ok := source.(map[string]interface{})
			if !ok {
				diags = append(diags, diag.Diagnostic{
					Severity:      diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query (refId:%s) to have a datasource", refID),
					AttributePath: append(currentPath, cty.IndexStep{Key: cty.StringVal("datasource")}),
				})
				return diags
			}

			currentPath = append(currentPath, cty.IndexStep{Key: cty.StringVal("datasource")})
			_, ok = s["type"]
			if !ok {
				diags = append(diags, diag.Diagnostic{
					Severity:      diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query (refId:%s) to have a type", refID),
					AttributePath: append(currentPath.Copy(), cty.IndexStep{Key: cty.StringVal("type")}),
				})
			}
			_, ok = s["uid"]
			if !ok {
				diags = append(diags, diag.Diagnostic{
					Severity:      diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query (refId:%s) to have a uid", refID),
					AttributePath: append(currentPath.Copy(), cty.IndexStep{Key: cty.StringVal("uid")}),
				})
			}
		}
		return diags
	}
}

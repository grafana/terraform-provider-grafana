package slo

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkv2diag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	sdkv2schema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	QueryTypeFreeform       string = "freeform"
	QueryTypeHistogram      string = "histogram"
	QueryTypeRatio          string = "ratio"
	QueryTypeThreshold      string = "threshold"
	QueryTypeGrafanaQueries string = "grafanaQueries"

	// Asserts integration constants
	AssertsProvenanceLabel = "grafana_slo_provenance"
	AssertsProvenanceValue = "asserts"
	AssertsRequestHeader   = "Grafana-Asserts-Request"
)

// Compile-time interface checks
var (
	_ resource.Resource                   = &sloResource{}
	_ resource.ResourceWithConfigure      = &sloResource{}
	_ resource.ResourceWithImportState    = &sloResource{}
	_ resource.ResourceWithValidateConfig = &sloResource{}
)

var (
	resourceSloName = "grafana_slo"
	resourceSloID   = common.NewResourceID(common.StringIDField("uuid"))
)

// ---------------------------------------------------------------------------
// Model structs
// ---------------------------------------------------------------------------

type resourceSloModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	FolderUID             types.String `tfsdk:"folder_uid"`
	UUID                  types.String `tfsdk:"uuid"`
	SearchExpression      types.String `tfsdk:"search_expression"`
	DestinationDatasource types.List   `tfsdk:"destination_datasource"`
	Query                 types.List   `tfsdk:"query"`
	Label                 types.List   `tfsdk:"label"`
	Objectives            types.List   `tfsdk:"objectives"`
	Alerting              types.List   `tfsdk:"alerting"`
}

type sloDestinationDatasourceModel struct {
	UID types.String `tfsdk:"uid"`
}

func sloDestinationDatasourceModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uid": types.StringType,
	}
}

type sloQueryModel struct {
	Type           types.String `tfsdk:"type"`
	Freeform       types.List   `tfsdk:"freeform"`
	GrafanaQueries types.List   `tfsdk:"grafana_queries"`
	Ratio          types.List   `tfsdk:"ratio"`
}

func sloQueryModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":            types.StringType,
		"freeform":        types.ListType{ElemType: types.ObjectType{AttrTypes: sloFreeformQueryModelAttrTypes()}},
		"grafana_queries": types.ListType{ElemType: types.ObjectType{AttrTypes: sloGrafanaQueriesModelAttrTypes()}},
		"ratio":           types.ListType{ElemType: types.ObjectType{AttrTypes: sloRatioQueryModelAttrTypes()}},
	}
}

type sloFreeformQueryModel struct {
	Query types.String `tfsdk:"query"`
}

func sloFreeformQueryModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"query": types.StringType,
	}
}

type sloGrafanaQueriesModel struct {
	GrafanaQueries types.String `tfsdk:"grafana_queries"`
}

func sloGrafanaQueriesModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"grafana_queries": types.StringType,
	}
}

type sloRatioQueryModel struct {
	SuccessMetric types.String `tfsdk:"success_metric"`
	TotalMetric   types.String `tfsdk:"total_metric"`
	GroupByLabels types.List   `tfsdk:"group_by_labels"`
}

func sloRatioQueryModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"success_metric":  types.StringType,
		"total_metric":    types.StringType,
		"group_by_labels": types.ListType{ElemType: types.StringType},
	}
}

type sloLabelModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func sloLabelModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
	}
}

type sloObjectiveModel struct {
	Value  types.Float64 `tfsdk:"value"`
	Window types.String  `tfsdk:"window"`
}

func sloObjectiveModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"value":  types.Float64Type,
		"window": types.StringType,
	}
}

type sloAlertingModel struct {
	Label           types.List `tfsdk:"label"`
	Annotation      types.List `tfsdk:"annotation"`
	FastBurn        types.List `tfsdk:"fastburn"`
	SlowBurn        types.List `tfsdk:"slowburn"`
	AdvancedOptions types.List `tfsdk:"advanced_options"`
}

func sloAlertingModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"label":            types.ListType{ElemType: types.ObjectType{AttrTypes: sloLabelModelAttrTypes()}},
		"annotation":       types.ListType{ElemType: types.ObjectType{AttrTypes: sloLabelModelAttrTypes()}},
		"fastburn":         types.ListType{ElemType: types.ObjectType{AttrTypes: sloAlertingMetadataModelAttrTypes()}},
		"slowburn":         types.ListType{ElemType: types.ObjectType{AttrTypes: sloAlertingMetadataModelAttrTypes()}},
		"advanced_options": types.ListType{ElemType: types.ObjectType{AttrTypes: sloAdvancedOptionsModelAttrTypes()}},
	}
}

type sloAlertingMetadataModel struct {
	Label      types.List `tfsdk:"label"`
	Annotation types.List `tfsdk:"annotation"`
	Enrichment types.List `tfsdk:"enrichment"`
}

func sloAlertingMetadataModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"label":      types.ListType{ElemType: types.ObjectType{AttrTypes: sloLabelModelAttrTypes()}},
		"annotation": types.ListType{ElemType: types.ObjectType{AttrTypes: sloLabelModelAttrTypes()}},
		"enrichment": types.ListType{ElemType: types.ObjectType{AttrTypes: sloEnrichmentModelAttrTypes()}},
	}
}

type sloEnrichmentModel struct {
	Type types.String `tfsdk:"type"`
}

func sloEnrichmentModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
	}
}

type sloAdvancedOptionsModel struct {
	MinFailures types.Int64 `tfsdk:"min_failures"`
}

func sloAdvancedOptionsModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"min_failures": types.Int64Type,
	}
}

// ---------------------------------------------------------------------------
// Resource struct + Configure + Metadata
// ---------------------------------------------------------------------------

type sloResource struct {
	client *slo.APIClient
}

func (r *sloResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *common.Client, got: %T", req.ProviderData),
		)
		return
	}

	if client.SLOClient == nil {
		resp.Diagnostics.AddError(
			"SLO API client not configured",
			"The SLO API client is required for this resource. Set the url and auth provider attributes.",
		)
		return
	}

	r.client = client.SLOClient
}

func (r *sloResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceSloName
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// Shared block definitions to reduce repetition.
func keyValueBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"key": schema.StringAttribute{
					Required:    true,
					Description: "Key for filtering and identification",
				},
				"value": schema.StringAttribute{
					Required:    true,
					Description: "Templatable value",
				},
			},
		},
	}
}

func enrichmentBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required:    true,
					Description: `Type of the alert enrichment. Currently only "assistantInvestigation" is supported.`,
					Validators: []validator.String{
						stringvalidator.OneOf("assistantInvestigation"),
					},
				},
			},
		},
	}
}

func alertingMetadataBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"label": func() schema.ListNestedBlock {
					b := keyValueBlock()
					b.Description = "Labels to attach only to this alert type."
					return b
				}(),
				"annotation": func() schema.ListNestedBlock {
					b := keyValueBlock()
					b.Description = "Annotations to attach only to this alert type."
					return b
				}(),
				"enrichment": func() schema.ListNestedBlock {
					b := enrichmentBlock()
					b.Description = "Enrichments to attach only to this alert type."
					return b
				}(),
			},
		},
	}
}

func (r *sloResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource manages Grafana SLOs (Service Level Objectives).\n\n" +
			"* [Official documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/)\n" +
			"* [API documentation](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/api/)\n" +
			"* [Additional Information On Alerting Rule Annotations and Labels](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#templating/)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the SLO. This is the same as the UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: `Name should be a short description of your indicator. Consider names like "API Availability"`,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 128),
				},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: `Description is a free-text field that can provide more context to an SLO.`,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
			"folder_uid": schema.StringAttribute{
				Optional:    true,
				Description: `UID for the SLO folder`,
			},
			"uuid": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: `UUID for the SLO. Custom UUIDs can be set. If not provided, a random UUID will be generated by the API.`,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"search_expression": schema.StringAttribute{
				Optional:    true,
				Description: "The name of a search expression in Grafana Asserts. This is used in the SLO UI to open the Asserts RCA workbench and in alerts to link to the RCA workbench.",
			},
		},
		Blocks: map[string]schema.Block{
			"destination_datasource": schema.ListNestedBlock{
				Description: `Destination Datasource sets the datasource defined for an SLO`,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"uid": schema.StringAttribute{
							Required:    true,
							Description: `UID for the Datasource`,
							Validators: []validator.String{
								nonEmptyStringValidator{},
							},
						},
					},
				},
			},
			"query": schema.ListNestedBlock{
				Description: `Query describes the indicator that will be measured against the objective. Freeform Query types are currently supported.`,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: `Query type must be one of: "freeform", "query", "ratio", "grafana_queries" or "threshold"`,
							Validators: []validator.String{
								stringvalidator.OneOf("freeform", "query", "ratio", "threshold", "grafana_queries"),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"freeform": schema.ListNestedBlock{
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"query": schema.StringAttribute{
										Required:    true,
										Description: "Freeform Query Field - valid promQl",
									},
								},
							},
						},
						"grafana_queries": schema.ListNestedBlock{
							Description: "Array for holding a set of grafana queries",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"grafana_queries": schema.StringAttribute{
										Required:    true,
										Description: "Query Object - Array of Grafana Query JSON objects",
										Validators: []validator.String{
											grafanaQueryValidator{},
										},
									},
								},
							},
						},
						"ratio": schema.ListNestedBlock{
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"success_metric": schema.StringAttribute{
										Required:    true,
										Description: `Counter metric for success events (numerator)`,
									},
									"total_metric": schema.StringAttribute{
										Required:    true,
										Description: `Metric for total events (denominator)`,
									},
									"group_by_labels": schema.ListAttribute{
										Optional:    true,
										ElementType: types.StringType,
										Description: `Defines Group By Labels used for per-label alerting. These appear as variables on SLO dashboards to enable filtering and aggregation. Labels must adhere to Prometheus label name schema - "^[a-zA-Z_][a-zA-Z0-9_]*$"`,
									},
								},
							},
						},
					},
				},
			},
			"label": func() schema.ListNestedBlock {
				b := keyValueBlock()
				b.Description = `Additional labels that will be attached to all metrics generated from the query. These labels are useful for grouping SLOs in dashboard views that you create by hand. Labels must adhere to Prometheus label name schema - "^[a-zA-Z_][a-zA-Z0-9_]*$"`
				return b
			}(),
			"objectives": schema.ListNestedBlock{
				Description: `Over each rolling time window, the remaining error budget will be calculated, and separate alerts can be generated for each time window based on the SLO burn rate or remaining error budget.`,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.Float64Attribute{
							Required:    true,
							Description: `Value between 0 and 1. If the value of the query is above the objective, the SLO is met.`,
							Validators: []validator.Float64{
								float64validator.Between(0, 1),
							},
						},
						"window": schema.StringAttribute{
							Required:    true,
							Description: `A Prometheus-parsable time duration string like 24h, 60m. This is the time window the objective is measured over.`,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^\d+(ms|s|m|h|d|w|y)$`),
									"Objective window must be a Prometheus-parsable time duration",
								),
							},
						},
					},
				},
			},
			"alerting": schema.ListNestedBlock{
				Description: "Configures the alerting rules that will be generated for each " +
					"time window associated with the SLO. Grafana SLOs can generate " +
					"alerts when the short-term error budget burn is very high, the " +
					"long-term error budget burn rate is high, or when the remaining " +
					"error budget is below a certain threshold. Annotations and Labels support templating.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"label": func() schema.ListNestedBlock {
							b := keyValueBlock()
							b.Description = `Labels will be attached to all alerts generated by any of these rules.`
							return b
						}(),
						"annotation": func() schema.ListNestedBlock {
							b := keyValueBlock()
							b.Description = `Annotations will be attached to all alerts generated by any of these rules.`
							return b
						}(),
						"fastburn": func() schema.ListNestedBlock {
							b := alertingMetadataBlock()
							b.Description = "Alerting Rules generated for Fast Burn alerts"
							return b
						}(),
						"slowburn": func() schema.ListNestedBlock {
							b := alertingMetadataBlock()
							b.Description = "Alerting Rules generated for Slow Burn alerts"
							return b
						}(),
						"advanced_options": schema.ListNestedBlock{
							Description: "Advanced Options for Alert Rules",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"min_failures": schema.Int64Attribute{
										Optional:    true,
										Description: "Minimum number of failed events to trigger an alert",
										Validators: []validator.Int64{
											int64validator.AtLeast(0),
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

// ---------------------------------------------------------------------------
// ValidateConfig — checks for blocks that are required but cannot express
// MinItems at the Terraform protocol level in the Plugin Framework.
// This preserves the SDKv2 "Insufficient destination_datasource blocks" behavior.
// ---------------------------------------------------------------------------

func (r *sloResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config resourceSloModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// destination_datasource is required (MinItems: 1 in SDKv2)
	if config.DestinationDatasource.IsNull() || len(config.DestinationDatasource.Elements()) == 0 {
		resp.Diagnostics.AddError(
			"Insufficient destination_datasource blocks",
			"At least 1 \"destination_datasource\" blocks are required.",
		)
	}
}

// ---------------------------------------------------------------------------
// Custom validators (Framework)
// ---------------------------------------------------------------------------

// nonEmptyStringValidator rejects empty strings with a message matching
// the SDKv2-era "uid must be a non-empty string" so that existing tests
// continue to pass unchanged.
type nonEmptyStringValidator struct{}

func (v nonEmptyStringValidator) Description(_ context.Context) string {
	return "uid must be a non-empty string"
}

func (v nonEmptyStringValidator) MarkdownDescription(_ context.Context) string {
	return "uid must be a non-empty string"
}

func (v nonEmptyStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			"uid must be a non-empty string",
		)
	}
}

type grafanaQueryValidator struct{}

func (v grafanaQueryValidator) Description(_ context.Context) string {
	return "Validates that the value is a valid Grafana queries JSON array"
}

func (v grafanaQueryValidator) MarkdownDescription(_ context.Context) string {
	return "Validates that the value is a valid Grafana queries JSON array"
}

func (v grafanaQueryValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueString()

	var gmrQuery []map[string]any
	if err := json.Unmarshal([]byte(val), &gmrQuery); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Bad Format",
			"expected grafana queries to be valid JSON format",
		)
		return
	}

	if len(gmrQuery) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Required Field",
			"expected grafana queries to have at least one query",
		)
		return
	}

	for _, queryObj := range gmrQuery {
		refID, ok := queryObj["refId"]
		if !ok {
			obj, _ := json.Marshal(queryObj)
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Missing Required Field",
				fmt.Sprintf("expected grafana query to have a 'refId' field (%s)", obj),
			)
			return
		}

		source := queryObj["datasource"]
		s, ok := source.(map[string]any)
		if !ok {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Missing Required Field",
				fmt.Sprintf("expected grafana query to have a 'datasource' field (refId:%s)", refID),
			)
			return
		}

		if _, ok = s["type"]; !ok {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Missing Required Field",
				fmt.Sprintf("expected grafana query datasource field to have a 'type' field (refId:%s)", refID),
			)
		}
		if _, ok = s["uid"]; !ok {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Missing Required Field",
				fmt.Sprintf("expected grafana query datasource field to have a 'uid' field (refId:%s)", refID),
			)
		}
	}
}

// ValidateGrafanaQuery returns an SDKv2-style validator for the grafana_queries field.
// Kept exported for unit test compatibility (TestValidateGrafanaQuery).
func ValidateGrafanaQuery() sdkv2schema.SchemaValidateDiagFunc {
	return func(i any, path cty.Path) sdkv2diag.Diagnostics {
		var diags sdkv2diag.Diagnostics

		v, ok := i.(string)
		if !ok {
			diags = append(diags, sdkv2diag.Diagnostic{
				Severity:      sdkv2diag.Error,
				Summary:       "Bad Format",
				Detail:        fmt.Sprintf("expected type of %s to be string", path),
				AttributePath: path,
			})
			return diags
		}

		var gmrQuery []map[string]any
		err := json.Unmarshal([]byte(v), &gmrQuery)
		if err != nil {
			diags = append(diags, sdkv2diag.Diagnostic{
				Severity:      sdkv2diag.Error,
				Summary:       "Bad Format",
				Detail:        "expected grafana queries to be valid JSON format",
				AttributePath: path,
			})
			return diags
		}

		if len(gmrQuery) == 0 {
			diags = append(diags, sdkv2diag.Diagnostic{
				Severity:      sdkv2diag.Error,
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
				obj, _ := json.Marshal(queryObj)
				diags = append(diags, sdkv2diag.Diagnostic{
					Severity:      sdkv2diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query to have a 'refId' field (%s)", obj),
					AttributePath: append(currentPath, cty.IndexStep{Key: cty.StringVal("refId")}),
				})
				return diags
			}

			source := queryObj["datasource"]
			s, ok := source.(map[string]any)
			if !ok {
				diags = append(diags, sdkv2diag.Diagnostic{
					Severity:      sdkv2diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query to have a 'datasource' field (refId:%s)", refID),
					AttributePath: append(currentPath, cty.IndexStep{Key: cty.StringVal("datasource")}),
				})
				return diags
			}

			currentPath = append(currentPath, cty.IndexStep{Key: cty.StringVal("datasource")})
			_, ok = s["type"]
			if !ok {
				diags = append(diags, sdkv2diag.Diagnostic{
					Severity:      sdkv2diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query datasource field to have a 'type' field (refId:%s)", refID),
					AttributePath: append(currentPath.Copy(), cty.IndexStep{Key: cty.StringVal("type")}),
				})
			}
			_, ok = s["uid"]
			if !ok {
				diags = append(diags, sdkv2diag.Diagnostic{
					Severity:      sdkv2diag.Error,
					Summary:       "Missing Required Field",
					Detail:        fmt.Sprintf("expected grafana query datasource field to have a 'uid' field (refId:%s)", refID),
					AttributePath: append(currentPath.Copy(), cty.IndexStep{Key: cty.StringVal("uid")}),
				})
			}
		}
		return diags
	}
}

// ---------------------------------------------------------------------------
// Asserts integration helpers (unchanged from SDKv2)
// ---------------------------------------------------------------------------

func hasAssertsProvenanceLabel(labels []slo.SloV00Label) bool {
	for _, label := range labels {
		if label.Key == AssertsProvenanceLabel && label.Value == AssertsProvenanceValue {
			return true
		}
	}
	return false
}

func createAssertsSLOClient(baseClient *slo.APIClient) *slo.APIClient {
	config := slo.NewConfiguration()
	config.Host = baseClient.GetConfig().Host
	config.Scheme = baseClient.GetConfig().Scheme
	config.HTTPClient = baseClient.GetConfig().HTTPClient
	config.DefaultHeader = make(map[string]string)
	for k, v := range baseClient.GetConfig().DefaultHeader {
		if k == "Grafana-Terraform-Provider" {
			continue
		}
		config.DefaultHeader[k] = v
	}
	config.DefaultHeader[AssertsRequestHeader] = "true"
	return slo.NewAPIClient(config)
}

// ---------------------------------------------------------------------------
// Lister (unchanged)
// ---------------------------------------------------------------------------

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
	for _, s := range slolist.Slos {
		ids = append(ids, s.Uuid)
	}
	return ids, nil
}

// ---------------------------------------------------------------------------
// Factory function
// ---------------------------------------------------------------------------

func makeResourceSlo() *common.Resource {
	return common.NewResource(
		common.CategorySLO,
		resourceSloName,
		resourceSloID,
		&sloResource{},
	).WithLister(listSlos).WithPreferredResourceNameField("name")
}

// ---------------------------------------------------------------------------
// CRUD methods
// ---------------------------------------------------------------------------

func (r *sloResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceSloModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sloModel, diags := packSloFromModel(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := r.client
	if hasAssertsProvenanceLabel(sloModel.Labels) {
		apiClient = createAssertsSLOClient(r.client)
	}

	response, _, err := apiClient.DefaultAPI.V1SloPost(ctx).SloV00Slo(sloModel).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to create SLO - API", formatAPIError(err))
		return
	}

	readData, diags := r.read(ctx, response.Uuid)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *sloResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceSloModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *sloResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceSloModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sloModel, diags := packSloFromModel(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := r.client
	if hasAssertsProvenanceLabel(sloModel.Labels) {
		apiClient = createAssertsSLOClient(r.client)
	}

	sloID := plan.ID.ValueString()
	if _, err := apiClient.DefaultAPI.V1SloIdPut(ctx, sloID).SloV00Slo(sloModel).Execute(); err != nil {
		resp.Diagnostics.AddError("Unable to update SLO", formatAPIError(err))
		return
	}

	readData, diags := r.read(ctx, sloID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *sloResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceSloModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.DefaultAPI.V1SloIdDelete(ctx, state.ID.ValueString()).Execute(); err != nil {
		resp.Diagnostics.AddError("Unable to delete SLO", formatAPIError(err))
	}
}

func (r *sloResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "SLO with ID "+req.ID+" not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

// ---------------------------------------------------------------------------
// Private read helper (API → model)
// ---------------------------------------------------------------------------

func (r *sloResource) read(ctx context.Context, id string) (*resourceSloModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	sloResp, httpResp, err := r.client.DefaultAPI.V1SloIdGet(ctx, id).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			return nil, diags
		}
		diags.AddError("Unable to read SLO", formatAPIError(err))
		return nil, diags
	}

	model := &resourceSloModel{
		ID:          types.StringValue(sloResp.Uuid),
		UUID:        types.StringValue(sloResp.Uuid),
		Name:        types.StringValue(sloResp.Name),
		Description: types.StringValue(sloResp.Description),
	}

	// folder_uid
	if sloResp.Folder != nil && sloResp.Folder.Uid != nil && *sloResp.Folder.Uid != "" {
		model.FolderUID = types.StringValue(*sloResp.Folder.Uid)
	} else {
		model.FolderUID = types.StringNull()
	}

	// search_expression
	if sloResp.SearchExpression != nil && *sloResp.SearchExpression != "" {
		model.SearchExpression = types.StringValue(*sloResp.SearchExpression)
	} else {
		model.SearchExpression = types.StringNull()
	}

	// destination_datasource
	var dsDiags diag.Diagnostics
	model.DestinationDatasource, dsDiags = convertDestDatasourceToList(ctx, sloResp.DestinationDatasource)
	diags.Append(dsDiags...)

	// query
	model.Query, dsDiags = convertQueryToList(ctx, sloResp.Query)
	diags.Append(dsDiags...)

	// objectives
	model.Objectives, dsDiags = convertObjectivesToList(ctx, sloResp.Objectives)
	diags.Append(dsDiags...)

	// label
	if len(sloResp.Labels) > 0 {
		model.Label, dsDiags = convertLabelsToList(ctx, sloResp.Labels)
		diags.Append(dsDiags...)
	} else {
		model.Label = types.ListNull(types.ObjectType{AttrTypes: sloLabelModelAttrTypes()})
	}

	// alerting
	if sloResp.Alerting != nil {
		model.Alerting, dsDiags = convertAlertingToList(ctx, sloResp.Alerting)
		diags.Append(dsDiags...)
	} else {
		model.Alerting = types.ListNull(types.ObjectType{AttrTypes: sloAlertingModelAttrTypes()})
	}

	return model, diags
}

// ---------------------------------------------------------------------------
// API → model conversion helpers
// ---------------------------------------------------------------------------

func convertDestDatasourceToList(ctx context.Context, ds *slo.SloV00DestinationDatasource) (types.List, diag.Diagnostics) {
	if ds == nil {
		return types.ListNull(types.ObjectType{AttrTypes: sloDestinationDatasourceModelAttrTypes()}), nil
	}
	uid := ""
	if ds.Uid != nil {
		uid = *ds.Uid
	}
	models := []sloDestinationDatasourceModel{
		{UID: types.StringValue(uid)},
	}
	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloDestinationDatasourceModelAttrTypes()}, models)
}

func convertQueryToList(ctx context.Context, apiQuery slo.SloV00Query) (types.List, diag.Diagnostics) {
	queryModel := sloQueryModel{
		Type:           types.StringValue(apiQuery.Type),
		Freeform:       types.ListNull(types.ObjectType{AttrTypes: sloFreeformQueryModelAttrTypes()}),
		GrafanaQueries: types.ListNull(types.ObjectType{AttrTypes: sloGrafanaQueriesModelAttrTypes()}),
		Ratio:          types.ListNull(types.ObjectType{AttrTypes: sloRatioQueryModelAttrTypes()}),
	}

	var d diag.Diagnostics

	switch apiQuery.Type {
	case QueryTypeFreeform:
		if apiQuery.Freeform != nil {
			freeformModels := []sloFreeformQueryModel{
				{Query: types.StringValue(apiQuery.Freeform.Query)},
			}
			queryModel.Freeform, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloFreeformQueryModelAttrTypes()}, freeformModels)
			if d.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: sloQueryModelAttrTypes()}), d
			}
		}
	case QueryTypeRatio:
		if apiQuery.Ratio != nil {
			ratioModel := sloRatioQueryModel{
				SuccessMetric: types.StringValue(apiQuery.Ratio.SuccessMetric.PrometheusMetric),
				TotalMetric:   types.StringValue(apiQuery.Ratio.TotalMetric.PrometheusMetric),
			}
			if len(apiQuery.Ratio.GroupByLabels) > 0 {
				ratioModel.GroupByLabels, d = types.ListValueFrom(ctx, types.StringType, apiQuery.Ratio.GroupByLabels)
				if d.HasError() {
					return types.ListNull(types.ObjectType{AttrTypes: sloQueryModelAttrTypes()}), d
				}
			} else {
				ratioModel.GroupByLabels = types.ListNull(types.StringType)
			}
			queryModel.Ratio, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloRatioQueryModelAttrTypes()}, []sloRatioQueryModel{ratioModel})
			if d.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: sloQueryModelAttrTypes()}), d
			}
		}
	case QueryTypeGrafanaQueries:
		if apiQuery.GrafanaQueries != nil {
			queryString, _ := json.Marshal(apiQuery.GrafanaQueries.GetGrafanaQueries())
			gqModels := []sloGrafanaQueriesModel{
				{GrafanaQueries: types.StringValue(string(queryString))},
			}
			queryModel.GrafanaQueries, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloGrafanaQueriesModelAttrTypes()}, gqModels)
			if d.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: sloQueryModelAttrTypes()}), d
			}
		}
	}

	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloQueryModelAttrTypes()}, []sloQueryModel{queryModel})
}

func convertObjectivesToList(ctx context.Context, objectives []slo.SloV00Objective) (types.List, diag.Diagnostics) {
	models := make([]sloObjectiveModel, len(objectives))
	for i, obj := range objectives {
		models[i] = sloObjectiveModel{
			Value:  types.Float64Value(obj.Value),
			Window: types.StringValue(obj.Window),
		}
	}
	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloObjectiveModelAttrTypes()}, models)
}

func convertLabelsToList(ctx context.Context, labels []slo.SloV00Label) (types.List, diag.Diagnostics) {
	models := make([]sloLabelModel, len(labels))
	for i, l := range labels {
		models[i] = sloLabelModel{
			Key:   types.StringValue(l.Key),
			Value: types.StringValue(l.Value),
		}
	}
	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloLabelModelAttrTypes()}, models)
}

// isAlertingMetadataEmpty returns true when the API-returned metadata contains
// no user-visible data (no labels, annotations, or enrichments).  The SLO API
// always returns non-nil fastburn/slowburn objects even when the user's config
// did not specify them, so we need to treat those empty shells as absent.
func isAlertingMetadataEmpty(meta *slo.SloV00AlertingMetadata) bool {
	if meta == nil {
		return true
	}
	return len(meta.Labels) == 0 && len(meta.Annotations) == 0 && len(meta.Enrichments) == 0
}

func convertAlertingToList(ctx context.Context, alertData *slo.SloV00Alerting) (types.List, diag.Diagnostics) {
	if alertData == nil {
		return types.ListNull(types.ObjectType{AttrTypes: sloAlertingModelAttrTypes()}), nil
	}

	var diags diag.Diagnostics
	alertModel := sloAlertingModel{}

	// labels
	if len(alertData.Labels) > 0 {
		var d diag.Diagnostics
		alertModel.Label, d = convertLabelsToList(ctx, alertData.Labels)
		diags.Append(d...)
	} else {
		alertModel.Label = types.ListNull(types.ObjectType{AttrTypes: sloLabelModelAttrTypes()})
	}

	// annotations
	if len(alertData.Annotations) > 0 {
		var d diag.Diagnostics
		alertModel.Annotation, d = convertLabelsToList(ctx, alertData.Annotations)
		diags.Append(d...)
	} else {
		alertModel.Annotation = types.ListNull(types.ObjectType{AttrTypes: sloLabelModelAttrTypes()})
	}

	// fastburn — treat API-returned empty metadata (no labels/annotations/enrichments) as absent
	// so that configs with `alerting {}` (no fastburn block) don't get a phantom block in state
	if alertData.FastBurn != nil && !isAlertingMetadataEmpty(alertData.FastBurn) {
		var d diag.Diagnostics
		alertModel.FastBurn, d = convertAlertingMetadataToList(ctx, alertData.FastBurn)
		diags.Append(d...)
	} else {
		alertModel.FastBurn = types.ListNull(types.ObjectType{AttrTypes: sloAlertingMetadataModelAttrTypes()})
	}

	// slowburn — same treatment as fastburn
	if alertData.SlowBurn != nil && !isAlertingMetadataEmpty(alertData.SlowBurn) {
		var d diag.Diagnostics
		alertModel.SlowBurn, d = convertAlertingMetadataToList(ctx, alertData.SlowBurn)
		diags.Append(d...)
	} else {
		alertModel.SlowBurn = types.ListNull(types.ObjectType{AttrTypes: sloAlertingMetadataModelAttrTypes()})
	}

	// advanced_options
	if alertData.AdvancedOptions != nil {
		var d diag.Diagnostics
		alertModel.AdvancedOptions, d = convertAdvancedOptionsToList(ctx, alertData.AdvancedOptions)
		diags.Append(d...)
	} else {
		alertModel.AdvancedOptions = types.ListNull(types.ObjectType{AttrTypes: sloAdvancedOptionsModelAttrTypes()})
	}

	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloAlertingModelAttrTypes()}, []sloAlertingModel{alertModel})
}

func convertAlertingMetadataToList(ctx context.Context, meta *slo.SloV00AlertingMetadata) (types.List, diag.Diagnostics) {
	if meta == nil {
		return types.ListNull(types.ObjectType{AttrTypes: sloAlertingMetadataModelAttrTypes()}), nil
	}

	var diags diag.Diagnostics
	metaModel := sloAlertingMetadataModel{}

	if len(meta.Labels) > 0 {
		var d diag.Diagnostics
		metaModel.Label, d = convertLabelsToList(ctx, meta.Labels)
		diags.Append(d...)
	} else {
		metaModel.Label = types.ListNull(types.ObjectType{AttrTypes: sloLabelModelAttrTypes()})
	}

	if len(meta.Annotations) > 0 {
		var d diag.Diagnostics
		metaModel.Annotation, d = convertLabelsToList(ctx, meta.Annotations)
		diags.Append(d...)
	} else {
		metaModel.Annotation = types.ListNull(types.ObjectType{AttrTypes: sloLabelModelAttrTypes()})
	}

	if len(meta.Enrichments) > 0 {
		var d diag.Diagnostics
		metaModel.Enrichment, d = convertEnrichmentsToList(ctx, meta.Enrichments)
		diags.Append(d...)
	} else {
		metaModel.Enrichment = types.ListNull(types.ObjectType{AttrTypes: sloEnrichmentModelAttrTypes()})
	}

	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloAlertingMetadataModelAttrTypes()}, []sloAlertingMetadataModel{metaModel})
}

func convertEnrichmentsToList(ctx context.Context, enrichments []slo.SloV00AlertEnrichment) (types.List, diag.Diagnostics) {
	models := make([]sloEnrichmentModel, len(enrichments))
	for i, e := range enrichments {
		models[i] = sloEnrichmentModel{
			Type: types.StringValue(e.Type),
		}
	}
	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloEnrichmentModelAttrTypes()}, models)
}

func convertAdvancedOptionsToList(ctx context.Context, opts *slo.SloV00AdvancedOptions) (types.List, diag.Diagnostics) {
	if opts == nil {
		return types.ListNull(types.ObjectType{AttrTypes: sloAdvancedOptionsModelAttrTypes()}), nil
	}

	model := sloAdvancedOptionsModel{}
	if opts.MinFailures != nil {
		model.MinFailures = types.Int64Value(*opts.MinFailures)
	} else {
		model.MinFailures = types.Int64Null()
	}

	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sloAdvancedOptionsModelAttrTypes()}, []sloAdvancedOptionsModel{model})
}

// ---------------------------------------------------------------------------
// Pack helpers (model → API)
// ---------------------------------------------------------------------------

func packSloFromModel(ctx context.Context, data *resourceSloModel) (slo.SloV00Slo, diag.Diagnostics) {
	var diags diag.Diagnostics

	apiSlo := slo.SloV00Slo{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	// UUID: use custom if set, otherwise empty (API generates one)
	if !data.UUID.IsNull() && !data.UUID.IsUnknown() && data.UUID.ValueString() != "" {
		apiSlo.Uuid = data.UUID.ValueString()
	}

	// folder_uid
	if !data.FolderUID.IsNull() && data.FolderUID.ValueString() != "" {
		folderUID := data.FolderUID.ValueString()
		apiSlo.Folder = &slo.SloV00Folder{Uid: &folderUID}
	}

	// search_expression
	if !data.SearchExpression.IsNull() && data.SearchExpression.ValueString() != "" {
		apiSlo.SearchExpression = common.Ref(data.SearchExpression.ValueString())
	}

	// destination_datasource
	ds, d := packDestDatasourceFromModel(ctx, data.DestinationDatasource)
	diags.Append(d...)
	apiSlo.DestinationDatasource = ds

	// query
	q, d := packQueryFromModel(ctx, data.Query)
	diags.Append(d...)
	apiSlo.Query = q

	// objectives
	objs, d := packObjectivesFromModel(ctx, data.Objectives)
	diags.Append(d...)
	apiSlo.Objectives = objs

	// labels
	if !data.Label.IsNull() && !data.Label.IsUnknown() {
		labels, d := packLabelsFromModel(ctx, data.Label)
		diags.Append(d...)
		apiSlo.Labels = labels
	}

	// alerting
	if !data.Alerting.IsNull() && !data.Alerting.IsUnknown() {
		alerting, d := packAlertingFromModel(ctx, data.Alerting)
		diags.Append(d...)
		apiSlo.Alerting = alerting
	}

	return apiSlo, diags
}

func packDestDatasourceFromModel(ctx context.Context, list types.List) (*slo.SloV00DestinationDatasource, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []sloDestinationDatasourceModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() || len(models) == 0 {
		return nil, diags
	}
	uid := models[0].UID.ValueString()
	return &slo.SloV00DestinationDatasource{Uid: &uid}, diags
}

func packQueryFromModel(ctx context.Context, list types.List) (slo.SloV00Query, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return slo.SloV00Query{}, diags
	}
	var queries []sloQueryModel
	diags.Append(list.ElementsAs(ctx, &queries, false)...)
	if diags.HasError() || len(queries) == 0 {
		return slo.SloV00Query{}, diags
	}

	q := queries[0]
	queryType := q.Type.ValueString()

	switch queryType {
	case "freeform":
		if !q.Freeform.IsNull() && !q.Freeform.IsUnknown() {
			var freeformModels []sloFreeformQueryModel
			diags.Append(q.Freeform.ElementsAs(ctx, &freeformModels, false)...)
			if len(freeformModels) > 0 {
				return slo.SloV00Query{
					Type:     QueryTypeFreeform,
					Freeform: &slo.SloV00FreeformQuery{Query: freeformModels[0].Query.ValueString()},
				}, diags
			}
		}
	case "ratio":
		if !q.Ratio.IsNull() && !q.Ratio.IsUnknown() {
			var ratioModels []sloRatioQueryModel
			diags.Append(q.Ratio.ElementsAs(ctx, &ratioModels, false)...)
			if len(ratioModels) > 0 {
				rm := ratioModels[0]
				var groupByLabels []string
				if !rm.GroupByLabels.IsNull() && !rm.GroupByLabels.IsUnknown() {
					diags.Append(rm.GroupByLabels.ElementsAs(ctx, &groupByLabels, false)...)
				}
				return slo.SloV00Query{
					Type: QueryTypeRatio,
					Ratio: &slo.SloV00RatioQuery{
						SuccessMetric: slo.SloV00MetricDef{PrometheusMetric: rm.SuccessMetric.ValueString()},
						TotalMetric:   slo.SloV00MetricDef{PrometheusMetric: rm.TotalMetric.ValueString()},
						GroupByLabels: groupByLabels,
					},
				}, diags
			}
		}
	case "grafana_queries":
		if !q.GrafanaQueries.IsNull() && !q.GrafanaQueries.IsUnknown() {
			var gqModels []sloGrafanaQueriesModel
			diags.Append(q.GrafanaQueries.ElementsAs(ctx, &gqModels, false)...)
			if len(gqModels) > 0 {
				var queryMapList []map[string]any
				if err := json.Unmarshal([]byte(gqModels[0].GrafanaQueries.ValueString()), &queryMapList); err != nil {
					diags.AddError("Invalid grafana_queries JSON", err.Error())
					return slo.SloV00Query{}, diags
				}
				return slo.SloV00Query{
					Type:           QueryTypeGrafanaQueries,
					GrafanaQueries: &slo.SloV00GrafanaQueries{GrafanaQueries: queryMapList},
				}, diags
			}
		}
	}

	return slo.SloV00Query{Type: queryType}, diags
}

func packObjectivesFromModel(ctx context.Context, list types.List) ([]slo.SloV00Objective, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []sloObjectiveModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	objectives := make([]slo.SloV00Objective, len(models))
	for i, m := range models {
		objectives[i] = slo.SloV00Objective{
			Value:  m.Value.ValueFloat64(),
			Window: m.Window.ValueString(),
		}
	}
	return objectives, diags
}

func packLabelsFromModel(ctx context.Context, list types.List) ([]slo.SloV00Label, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []sloLabelModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	labels := make([]slo.SloV00Label, len(models))
	for i, m := range models {
		labels[i] = slo.SloV00Label{
			Key:   m.Key.ValueString(),
			Value: m.Value.ValueString(),
		}
	}
	return labels, diags
}

func packAlertingFromModel(ctx context.Context, list types.List) (*slo.SloV00Alerting, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []sloAlertingModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() || len(models) == 0 {
		return &slo.SloV00Alerting{}, diags
	}

	m := models[0]
	alerting := &slo.SloV00Alerting{}

	// labels
	if !m.Label.IsNull() && !m.Label.IsUnknown() {
		labels, d := packLabelsFromModel(ctx, m.Label)
		diags.Append(d...)
		alerting.Labels = labels
	}

	// annotations
	if !m.Annotation.IsNull() && !m.Annotation.IsUnknown() {
		annots, d := packLabelsFromModel(ctx, m.Annotation)
		diags.Append(d...)
		alerting.Annotations = annots
	}

	// fastburn
	if !m.FastBurn.IsNull() && !m.FastBurn.IsUnknown() {
		fb, d := packAlertMetadataFromModel(ctx, m.FastBurn)
		diags.Append(d...)
		alerting.FastBurn = fb
	} else {
		alerting.FastBurn = &slo.SloV00AlertingMetadata{}
	}

	// slowburn
	if !m.SlowBurn.IsNull() && !m.SlowBurn.IsUnknown() {
		sb, d := packAlertMetadataFromModel(ctx, m.SlowBurn)
		diags.Append(d...)
		alerting.SlowBurn = sb
	} else {
		alerting.SlowBurn = &slo.SloV00AlertingMetadata{}
	}

	// advanced_options
	if !m.AdvancedOptions.IsNull() && !m.AdvancedOptions.IsUnknown() {
		ao, d := packAdvancedOptionsFromModel(ctx, m.AdvancedOptions)
		diags.Append(d...)
		if ao != nil {
			alerting.SetAdvancedOptions(*ao)
		}
	}

	return alerting, diags
}

func packAlertMetadataFromModel(ctx context.Context, list types.List) (*slo.SloV00AlertingMetadata, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []sloAlertingMetadataModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() || len(models) == 0 {
		return &slo.SloV00AlertingMetadata{}, diags
	}

	m := models[0]
	meta := &slo.SloV00AlertingMetadata{}

	if !m.Label.IsNull() && !m.Label.IsUnknown() {
		labels, d := packLabelsFromModel(ctx, m.Label)
		diags.Append(d...)
		meta.Labels = labels
	}

	if !m.Annotation.IsNull() && !m.Annotation.IsUnknown() {
		annots, d := packLabelsFromModel(ctx, m.Annotation)
		diags.Append(d...)
		meta.Annotations = annots
	}

	if !m.Enrichment.IsNull() && !m.Enrichment.IsUnknown() {
		enrichments, d := packEnrichmentsFromModel(ctx, m.Enrichment)
		diags.Append(d...)
		meta.Enrichments = enrichments
	}

	return meta, diags
}

func packEnrichmentsFromModel(ctx context.Context, list types.List) ([]slo.SloV00AlertEnrichment, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []sloEnrichmentModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	enrichments := make([]slo.SloV00AlertEnrichment, len(models))
	for i, m := range models {
		enrichments[i] = slo.SloV00AlertEnrichment{
			Type: m.Type.ValueString(),
		}
	}
	return enrichments, diags
}

func packAdvancedOptionsFromModel(ctx context.Context, list types.List) (*slo.SloV00AdvancedOptions, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []sloAdvancedOptionsModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() || len(models) == 0 {
		return nil, diags
	}

	opts := &slo.SloV00AdvancedOptions{}
	if !models[0].MinFailures.IsNull() && !models[0].MinFailures.IsUnknown() {
		v := models[0].MinFailures.ValueInt64()
		opts.MinFailures = &v
	}

	return opts, diags
}

// ---------------------------------------------------------------------------
// Error formatting
// ---------------------------------------------------------------------------

func formatAPIError(err error) string {
	if err == nil {
		return ""
	}
	detail := err.Error()
	if apiErr, ok := err.(*slo.GenericOpenAPIError); ok {
		detail += "\n" + string(apiErr.Body())
	}
	return detail
}

package slo

import (
	"context"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSlo() *schema.Resource {
	return &schema.Resource{
		Description: `
		* [Official documentation](https://grafana.com/docs/grafana-cloud/slo/)
		* [API documentation](https://grafana.com/docs/grafana-cloud/slo/api/)
		`,
		CreateContext: resourceSloCreate,
		ReadContext:   resourceSloRead,
		UpdateContext: resourceSloUpdate,
		DeleteContext: resourceSloDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"service": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
			"objectives": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
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
			"dashboard_uid": {
				Type:     schema.TypeString,
				Computed: true,
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
							Optional: true,
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
							Optional: true,
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
			"last_updated": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

// SLO Resource is defined by the user within the Terraform State file
// When 'terraform apply' is executed, it sends a POST Request and converts
// the data within the Terraform State into a JSON Object which is then sent to the API
// Following this, a READ is executed for the newly created SLO, which is then displayed within the
// terminal that Terraform is running in
func resourceSloCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	slo := packSloResource(d)

	client := m.(*common.Client).GrafanaAPI
	response, _ := client.CreateSlo(slo)

	// Get the response back from the API, we need to set the ID of the Terraform Resource
	d.SetId(response.Uuid)

	// Executes a READ, displays the newly created SLO Resource within Terraform
	resourceSloRead(ctx, d, m)

	return diags
}

// resourceSloRead - sends a GET Request to the single SLO Endpoint
func resourceSloRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	sloId := d.Id()

	client := m.(*common.Client).GrafanaAPI
	slo, _ := client.GetSlo(sloId)

	setTerraformState(d, slo)

	return diags
}

func resourceSloUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sloID := d.Id()

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("service") || d.HasChange("query") || d.HasChange("labels") || d.HasChange("objectives") || d.HasChange("alerting") {
		slo := packSloResource(d)

		grafanaClient := m.(*common.Client)
		grafanaClient.GrafanaAPI.UpdateSlo(sloID, slo)

		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	return resourceSloRead(ctx, d, m)
}

func resourceSloDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	sloID := d.Id()

	client := m.(*common.Client).GrafanaAPI
	client.DeleteSlo(sloID)

	d.SetId("")

	return diags
}

// Fetches all the Properties defined on the Terraform SLO State Object and converts it
// to a Slo so that it can be converted to JSON and sent to the API
func packSloResource(d *schema.ResourceData) gapi.Slo {
	tfname := d.Get("name").(string)
	tfdescription := d.Get("description").(string)
	tfservice := d.Get("service").(string)
	query := d.Get("query").(string)
	tfquery := packQuery(query)

	// Assumes that each SLO only has one Objective Value and one Objective Window
	objectives := d.Get("objectives").([]interface{})
	objective := objectives[0].(map[string]interface{})
	tfobjective := packObjective(objective)

	labels := d.Get("labels").([]interface{})
	tflabels := packLabels(labels)

	alerting := d.Get("alerting").([]interface{})
	alert := alerting[0].(map[string]interface{})
	tfalerting := packAlerting(alert)

	slo := gapi.Slo{
		Uuid:        d.Id(),
		Name:        tfname,
		Description: tfdescription,
		Service:     tfservice,
		Objectives:  tfobjective,
		Query:       tfquery,
		Alerting:    &tfalerting,
		Labels:      &tflabels,
	}

	return slo
}

func packQuery(query string) gapi.Query {
	sloQuery := gapi.Query{
		FreeformQuery: gapi.FreeformQuery{
			Query: query,
		},
	}

	return sloQuery
}

func packObjective(tfobjective map[string]interface{}) []gapi.Objective {
	objective := gapi.Objective{
		Value:  tfobjective["objective_value"].(float64),
		Window: tfobjective["objective_window"].(string),
	}

	objectiveSlice := []gapi.Objective{}
	objectiveSlice = append(objectiveSlice, objective)

	return objectiveSlice
}

func packLabels(tfLabels []interface{}) []gapi.Label {
	labelSlice := []gapi.Label{}

	for ind := range tfLabels {
		currLabel := tfLabels[ind].(map[string]interface{})
		curr := gapi.Label{
			Key:   currLabel["key"].(string),
			Value: currLabel["value"].(string),
		}

		labelSlice = append(labelSlice, curr)

	}

	return labelSlice
}

func packAlerting(tfAlerting map[string]interface{}) gapi.Alerting {
	annots := tfAlerting["annotations"].([]interface{})
	tfAnnots := packLabels(annots)

	labels := tfAlerting["labels"].([]interface{})
	tfLabels := packLabels(labels)

	fastBurn := tfAlerting["fastburn"].([]interface{})
	tfFastBurn := packAlertMetadata(fastBurn)

	slowBurn := tfAlerting["slowburn"].([]interface{})
	tfSlowBurn := packAlertMetadata(slowBurn)

	alerting := gapi.Alerting{
		Name:        tfAlerting["name"].(string),
		Annotations: &tfAnnots,
		Labels:      &tfLabels,
		FastBurn:    &tfFastBurn,
		SlowBurn:    &tfSlowBurn,
	}

	return alerting
}

func packAlertMetadata(metadata []interface{}) gapi.AlertMetadata {
	meta := metadata[0].(map[string]interface{})

	labels := meta["labels"].([]interface{})
	tflabels := packLabels(labels)

	annots := meta["annotations"].([]interface{})
	tfannots := packLabels(annots)

	apiMetadata := gapi.AlertMetadata{
		Labels:      &tflabels,
		Annotations: &tfannots,
	}

	return apiMetadata
}

func setTerraformState(d *schema.ResourceData, slo gapi.Slo) {
	d.Set("name", slo.Name)
	d.Set("description", slo.Description)
	d.Set("service", slo.Service)
	d.Set("query", unpackQuery(slo.Query))
	retLabels := unpackLabels(slo.Labels)

	d.Set("labels", retLabels)

	retDashboard := unpackDashboard(slo)
	d.Set("dashboard_uid", retDashboard)

	retObjectives := unpackObjectives(slo.Objectives)
	d.Set("objectives", retObjectives)

	retAlerting := unpackAlerting(slo.Alerting)
	d.Set("alerting", retAlerting)
}

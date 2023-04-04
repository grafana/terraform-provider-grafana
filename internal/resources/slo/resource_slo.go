package slo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSlo() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSloCreate,
		ReadContext:   resourceSloRead,
		UpdateContext: resourceSloUpdate,
		DeleteContext: resourceSloDelete,
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

func resourceSloCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	sloPost := packSloResource(d)

	body, err := json.Marshal(sloPost)
	if err != nil {
		log.Fatalln(err)
	}
	bodyReader := bytes.NewReader(body)

	serverPort := 3000
	requestURL := fmt.Sprintf("http://localhost:%d/api/plugins/grafana-slo-app/resources/v1/slo", serverPort)
	req, err := http.NewRequest(http.MethodPost, requestURL, bodyReader)
	if err != nil {
		log.Fatalln(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var response POSTResponse

	err = json.Unmarshal(b, &response)
	if err != nil {
		fmt.Println("error:", err)
	}

	d.SetId(response.Uuid)
	resourceSloRead(ctx, d, m)

	return diags
}

type POSTResponse struct {
	Message string `json:"message,omitempty"`
	Uuid    string `json:"uuid,omitempty"`
}

func packSloResource(d *schema.ResourceData) Slo {
	tfname := d.Get("name").(string)
	tfdescription := d.Get("description").(string)
	tfservice := d.Get("service").(string)
	query := d.Get("query").(string)
	tfquery := packQuery(query)

	// Assumes that each SLO only have one Objective Value and one Objective Window
	objectives := d.Get("objectives").([]interface{})
	objective := objectives[0].(map[string]interface{})
	tfobjective := packObjective(objective)

	labels := d.Get("labels").([]interface{})
	tflabels := packLabels(labels)

	alerting := d.Get("alerting").([]interface{})
	alert := alerting[0].(map[string]interface{})
	tfalerting := packAlerting(alert)

	sloPost := Slo{
		Uuid:        d.Id(),
		Name:        tfname,
		Description: tfdescription,
		Service:     tfservice,
		Objectives:  tfobjective,
		Query:       tfquery,
		Alerting:    &tfalerting,
		Labels:      &tflabels,
	}

	return sloPost

}

func packQuery(query string) Query {
	sloQuery := Query{
		FreeformQuery: FreeformQuery{
			Query: query,
		},
	}

	return sloQuery
}

func packObjective(tfobjective map[string]interface{}) []Objective {
	objective := Objective{
		Value:  tfobjective["objective_value"].(float64),
		Window: tfobjective["objective_window"].(string),
	}

	objectiveSlice := []Objective{}
	objectiveSlice = append(objectiveSlice, objective)

	return objectiveSlice
}

func packLabels(tfLabels []interface{}) []Label {
	labelSlice := []Label{}

	for ind := range tfLabels {
		currLabel := tfLabels[ind].(map[string]interface{})
		curr := Label{
			Key:   currLabel["key"].(string),
			Value: currLabel["value"].(string),
		}

		labelSlice = append(labelSlice, curr)

	}

	return labelSlice
}

func packAlerting(tfAlerting map[string]interface{}) Alerting {
	annots := tfAlerting["annotations"].([]interface{})
	tfAnnots := packLabels(annots)

	labels := tfAlerting["labels"].([]interface{})
	tfLabels := packLabels(labels)

	fastBurn := tfAlerting["fastburn"].([]interface{})
	tfFastBurn := packAlertMetadata(fastBurn)

	slowBurn := tfAlerting["slowburn"].([]interface{})
	tfSlowBurn := packAlertMetadata(slowBurn)

	alerting := Alerting{
		Name:        tfAlerting["name"].(string),
		Annotations: &tfAnnots,
		Labels:      &tfLabels,
		FastBurn:    &tfFastBurn,
		SlowBurn:    &tfSlowBurn,
	}

	return alerting
}

func packAlertMetadata(metadata []interface{}) AlertMetadata {
	meta := metadata[0].(map[string]interface{})

	labels := meta["labels"].([]interface{})
	tflabels := packLabels(labels)

	annots := meta["annotations"].([]interface{})
	tfannots := packLabels(annots)

	apiMetadata := AlertMetadata{
		Labels:      &tflabels,
		Annotations: &tfannots,
	}

	return apiMetadata
}

func resourceSloRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sloID := d.Id()

	serverPort := 3000
	requestURL := fmt.Sprintf("http://localhost:%d/api/plugins/grafana-slo-app/resources/v1/slo/%s", serverPort, sloID)
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

	var slo Slo

	err = json.Unmarshal(b, &slo)
	if err != nil {
		fmt.Println("error:", err)
	}

	// When you return this, you only need to set the Id and the Dashboard - the information you get back from the API
	retDashboard := unpackDashboard(slo)

	d.Set("dashboard_ref", retDashboard)
	d.Set("name", slo.Name)

	var diags diag.Diagnostics
	return diags
}

func resourceSloDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sloID := d.Id()

	var diags diag.Diagnostics

	serverPort := 3000
	requestURL := fmt.Sprintf("http://localhost:%d/api/plugins/grafana-slo-app/resources/v1/slo/%s", serverPort, sloID)
	req, err := http.NewRequest(http.MethodDelete, requestURL, nil)
	if err != nil {
		log.Fatalln(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}

	defer resp.Body.Close()

	d.SetId("")

	return diags
}

func resourceSloUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sloID := d.Id()

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("service") || d.HasChange("query") || d.HasChange("labels") || d.HasChange("objectives") || d.HasChange("alerting") {
		sloPut := packSloResource(d)

		body, err := json.Marshal(sloPut)
		if err != nil {
			log.Fatalln(err)
		}
		bodyReader := bytes.NewReader(body)

		serverPort := 3000
		requestURL := fmt.Sprintf("http://localhost:%d/api/plugins/grafana-slo-app/resources/v1/slo/%s", serverPort, sloID)
		req, err := http.NewRequest(http.MethodPut, requestURL, bodyReader)
		if err != nil {
			log.Fatalln(err)
		}

		client := &http.Client{}
		_, err = client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}

		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	return resourceSloRead(ctx, d, m)
}

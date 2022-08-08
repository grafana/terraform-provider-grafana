package grafana

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func ResourceAlertRule() *schema.Resource {
	return &schema.Resource{
		Description: `TODO`,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"group": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"folder_uid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"interval_seconds": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "TODO",
			},
			"rules": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "TODO",
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "TODO",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "TODO",
						},
						"for": {
							Type:        schema.TypeInt,
							Required:    false,
							Description: "TODO",
						},
						"no_data_state": {
							Type:        schema.TypeString,
							Required:    false,
							Default:     "NoData",
							Description: "TODO",
						},
						"exec_err_state": {
							Type:        schema.TypeString,
							Required:    false,
							Default:     "Alerting",
							Description: "TODO",
						},
						"condition": {
							Type:        schema.TypeString,
							Required:    true, // TODO??
							Description: "TODO",
						},
						"data": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "TODO",
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ref_id": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "TODO",
									},
									"datasource_uid": {
										Type:        schema.TypeString,
										Required:    false,
										Description: "TODO",
									},
									"query_type": {
										Type:        schema.TypeString,
										Required:    false,
										Description: "TODO",
									},
									"model": {
										// TypeMap with no elem is equivalent to a JSON object.
										Type:        schema.TypeMap,
										Required:    true,
										Description: "TODO",
									},
									"relative_time_range": {
										Type:        schema.TypeMap,
										Required:    false,
										Description: "TODO",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"from": {
													Type:        schema.TypeInt,
													Required:    true,
													Description: "TODO",
												},
												"to": {
													Type:        schema.TypeInt,
													Required:    true,
													Description: "TODO",
												},
											},
										},
									},
								},
							},
						},
						"labels": {
							Type:        schema.TypeMap,
							Optional:    true,
							Default:     map[string]interface{}{},
							Description: "TODO",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"annotations": {
							Type:        schema.TypeMap,
							Optional:    true,
							Default:     map[string]interface{}{},
							Description: "TODO",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

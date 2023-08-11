package connections

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/common/connections"
)

func ResourceConnectionAWSService() *schema.Resource {
	return &schema.Resource{
		Description: `
An AWS AWSService definition instructs an AWS connection to pull metric data from AWS for a specific service to Grafana Cloud   
`,

		CreateContext: AWSServiceCreate,
		ReadContext:   AWSServiceRead,
		UpdateContext: AWSServiceUpdate,
		DeleteContext: AWSServiceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID of the grafana cloud instance",
				Type:        schema.TypeString,
				Required:    true,
			},
			"connection_name": {
				Description: "The connection to associate this service definition with",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "The name of the AWS service to collect",
				Type:        schema.TypeString,
				Required:    true,
			},
			"pull_interval": {
				Description: "How often to pull data expressed as a duration",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"metrics": {
				Description: "The regions this connection applies to",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the metric being pulled",
							Type:        schema.TypeString,
							Required:    true,
						},
						"statistics": {
							Description: "The statistics to pull for this metric",
							Type:        schema.TypeList,
							Required:    true,
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

func AWSServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).ConnectionsAPI
	stackID, c, s, err := makeService(d)
	if err != nil {
		return diag.FromErr(err)
	}
	err = client.CreateAWSService(ctx, stackID, c, s)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(toServiceID(stackID, c, s.Name))
	return AWSServiceRead(ctx, d, meta)
}

func AWSServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).ConnectionsAPI
	stackID := d.Get("stack_id").(string)
	cn := d.Get("connection_name").(string)
	sn := d.Get("name").(string)
	service, err := c.GetAWSService(ctx, stackID, cn, sn)
	if err, shouldReturn := common.CheckReadError("AWSService", d, err); shouldReturn {
		return err
	}
	var setMetrics []map[string]interface{}
	for _, metric := range service.Metrics {
		setMetric := make(map[string]interface{})
		setMetric["name"] = metric.Name
		setMetric["statistics"] = metric.Statistics
		setMetrics = append(setMetrics, setMetric)
	}
	err = d.Set("name", service.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	if service.ScrapeIntervalSeconds != nil {
		duration := time.Duration(*service.ScrapeIntervalSeconds)
		err = d.Set("pull_interval", duration.String())
	}
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("metrics", setMetrics)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func AWSServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).ConnectionsAPI
	stackID, c, s, err := makeService(d)
	if err != nil {
		return diag.FromErr(err)
	}
	err = client.UpdateAWSService(ctx, stackID, c, s)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(toServiceID(stackID, c, s.Name))
	return AWSServiceRead(ctx, d, meta)
}

func AWSServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).ConnectionsAPI
	stackID := d.Get("stack_id").(string)
	cn := d.Get("connection_name").(string)
	sn := d.Get("name").(string)
	err := client.DeleteAWSService(ctx, stackID, cn, sn)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func makeService(d *schema.ResourceData) (string, string, connections.AWSService, error) {
	stackID := d.Get("stack_id").(string)
	connectionName := d.Get("connection_name").(string)
	var pullInterval *int64
	if interval := d.Get("pull_interval").(string); interval != "" {
		parsedInterval, err := time.ParseDuration(d.Get("pull_interval").(string))
		if err != nil {
			return "", "", connections.AWSService{}, fmt.Errorf("pull_interval is not a valid duration: %w", err)
		}
		pullIntervalSeconds := int64(parsedInterval.Seconds())
		pullInterval = &pullIntervalSeconds
	}

	var metrics []connections.Metric
	if setMetrics := d.Get("metrics").([]interface{}); setMetrics != nil {
		for _, setMetric := range setMetrics {
			inputs := setMetric.(map[string]interface{})
			name := inputs["name"].(string)
			statistics := common.ListToStringSlice(inputs["statistics"].([]interface{}))
			metrics = append(metrics, connections.Metric{
				Name:       name,
				Statistics: statistics,
			})
		}
	}

	return stackID, connectionName, connections.AWSService{
		Name:                  d.Get("name").(string),
		Metrics:               metrics,
		ScrapeIntervalSeconds: pullInterval,
	}, nil
}

func toServiceID(stackID, connectionName string, serviceName string) string {
	return fmt.Sprintf("%s_%s_%s", stackID, connectionName, serviceName)
}

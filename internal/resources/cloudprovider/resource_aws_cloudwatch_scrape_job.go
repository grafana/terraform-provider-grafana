package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	resourceAWSCWScrapeJobTerraformID = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("job_name"))
)

func resourceAWSCloudWatchScrapeJob() *common.Resource {
	schema := &schema.Resource{
		CreateContext: resourceAWSCloudWatchScrapeJobCreate,
		ReadContext:   resourceAWSCloudWatchScrapeJobRead,
		UpdateContext: resourceAWSCloudWatchScrapeJobUpdate,
		DeleteContext: resourceAWSCloudWatchScrapeJobDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importAWSCloudWatchScrapeJobState,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ job_name }}\".",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"stack_id": {
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "The name of the CloudWatch Scrape Job. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"enabled": {
				Description: "Whether the CloudWatch Scrape Job is enabled or not.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     true,
			},
			"aws_account_resource_id": {
				Description: "The ID assigned by the Grafana Cloud Provider API to an AWS Account resource that should be associated with this CloudWatch Scrape Job.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regions": {
				Description: "A set of AWS region names that this CloudWatch Scrape Job applies to.",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"service_configuration": {
				Description: "Each block is a service configuration that dictates what this CloudWatch Scrape Job should scrape for the specified AWS service.",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The name of the service to scrape. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported services, metrics, and their statistics.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"metrics": {
							Description: "A set of metrics to scrape.",
							Type:        schema.TypeSet,
							Required:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Description: "The name of the metric to scrape.",
										Type:        schema.TypeString,
										Required:    true,
									},
									"statistics": {
										Description: "A set of statistics to scrape.",
										Type:        schema.TypeSet,
										Required:    true,
										MinItems:    1,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
						"scrape_interval_seconds": {
							Description: "The interval in seconds to scrape the service. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported scrape intervals.",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     300,
						},
						"resource_discovery_tag_filters": {
							Description: "A set of tag filters to use for discovery of resource entities in the associated AWS account.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Description: "The key of the tag filter.",
										Type:        schema.TypeString,
										Required:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"value": {
										Description: "The value of the tag filter.",
										Type:        schema.TypeString,
										Required:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
						"tags_to_add_to_metrics": {
							Description: "A set of tags to add to all metrics exported by this scrape job, for use in PromQL queries.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"is_custom_namespace": {
							Description: "Whether the service name is a custom, user-generated metrics namespace, as opposed to a standard AWS service metrics namespace.",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
					},
				},
			},
		},
	}

	return common.NewResource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_cloudwatch_scrape_job",
		resourceAWSCWScrapeJobTerraformID,
		schema,
	)
}

func resourceAWSCloudWatchScrapeJobCreate(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {

	d.SetId(resourceAWSCWScrapeJobTerraformID.Make(TestAWSCloudWatchScrapeJobData.StackID, TestAWSCloudWatchScrapeJobData.Name))

	return resourceAWSCloudWatchScrapeJobRead(ctx, d, c)
}

func resourceAWSCloudWatchScrapeJobRead(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.Set("stack_id", TestAWSCloudWatchScrapeJobData.StackID)
	d.Set("name", TestAWSCloudWatchScrapeJobData.Name)
	d.Set("aws_account_resource_id", TestAWSCloudWatchScrapeJobData.AWSAccountResourceID)
	d.Set("enabled", TestAWSCloudWatchScrapeJobData.Enabled)
	d.Set("regions", TestAWSCloudWatchScrapeJobData.Regions)
	d.Set("service_configurations", TestAWSCloudWatchScrapeJobData.ServiceConfigurations)

	return diags
}

func resourceAWSCloudWatchScrapeJobUpdate(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	TestAWSCloudWatchScrapeJobData.StackID = d.Get("stack_id").(string)
	TestAWSCloudWatchScrapeJobData.Name = d.Get("name").(string)
	TestAWSCloudWatchScrapeJobData.AWSAccountResourceID = d.Get("aws_account_resource_id").(string)
	TestAWSCloudWatchScrapeJobData.Enabled = d.Get("enabled").(bool)
	TestAWSCloudWatchScrapeJobData.Regions = common.SetToStringSlice(d.Get("regions").(*schema.Set))
	TestAWSCloudWatchScrapeJobData.ServiceConfigurations = make([]cloudproviderapi.AWSCloudWatchServiceConfiguration, len(d.Get("service_configurations").([]interface{})))
	for i, serviceConfig := range d.Get("service_configurations").([]interface{}) {
		serviceConfigMap := serviceConfig.(map[string]interface{})
		TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].Name = serviceConfigMap["name"].(string)
		TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].ScrapeIntervalSeconds = int64(serviceConfigMap["scrape_interval_seconds"].(int))
		TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].IsCustomNamespace = serviceConfigMap["is_custom_namespace"].(bool)
		TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].Metrics = make([]cloudproviderapi.AWSCloudWatchMetric, len(serviceConfigMap["metrics"].([]interface{})))
		for j, metric := range serviceConfigMap["metrics"].([]interface{}) {
			metricMap := metric.(map[string]interface{})
			TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].Metrics[j].Name = metricMap["name"].(string)
			TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].Metrics[j].Statistics = common.SetToStringSlice(metricMap["statistics"].(*schema.Set))
		}
		TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].ResourceDiscoveryTagFilters = make([]cloudproviderapi.AWSCloudWatchTagFilter, len(serviceConfigMap["resource_discovery_tag_filters"].([]interface{})))
		for j, tagFilter := range serviceConfigMap["resource_discovery_tag_filters"].([]interface{}) {
			tagFilterMap := tagFilter.(map[string]interface{})
			TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].ResourceDiscoveryTagFilters[j].Key = tagFilterMap["key"].(string)
			TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].ResourceDiscoveryTagFilters[j].Value = tagFilterMap["value"].(string)
		}
		TestAWSCloudWatchScrapeJobData.ServiceConfigurations[i].TagsToAddToMetrics = common.SetToStringSlice(serviceConfigMap["tags_to_add_to_metrics"].(*schema.Set))
	}

	d.Set("stack_id", TestAWSCloudWatchScrapeJobData.StackID)
	d.Set("name", TestAWSCloudWatchScrapeJobData.Name)
	d.Set("aws_account_resource_id", TestAWSCloudWatchScrapeJobData.AWSAccountResourceID)
	d.Set("enabled", TestAWSCloudWatchScrapeJobData.Enabled)
	d.Set("regions", TestAWSCloudWatchScrapeJobData.Regions)
	d.Set("service_configurations", TestAWSCloudWatchScrapeJobData.ServiceConfigurations)

	return diags
}

func resourceAWSCloudWatchScrapeJobDelete(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.SetId("")

	return diags
}

func importAWSCloudWatchScrapeJobState(ctx context.Context, d *schema.ResourceData, c interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid import ID: %s", d.Id())
	}
	d.Set("stack_id", parts[0])
	d.Set("job_name", parts[1])
	return []*schema.ResourceData{d}, nil
}

package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type AWSCloudWatchServiceConfigurations struct {
	Name                  string
	Metrics               []AWSCloudWatchMetric
	ScrapeIntervalSeconds int64
}
type AWSCloudWatchMetric struct {
	Name       string
	Statistics []string
}

var TestAWSCloudWatchScrapeJobData = struct {
	StackID               string
	JobName               string
	JobEnabled            bool
	AWSAccountResourceID  string
	Regions               []string
	ServiceConfigurations []AWSCloudWatchServiceConfigurations
}{
	StackID:              "001",
	JobName:              "test-scrape-job",
	AWSAccountResourceID: "1",
	Regions:              []string{"us-east-1", "us-east-2", "us-west-1"},
	ServiceConfigurations: []AWSCloudWatchServiceConfigurations{
		{
			Name: "EC2",
			Metrics: []AWSCloudWatchMetric{
				{
					Name:       "CPUUtilization",
					Statistics: []string{"Average"},
				},
			},
			ScrapeIntervalSeconds: 300,
		},
	},
}

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
				Description: "The StackID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "The name of the CloudWatch Scrape Job. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"enabled": {
				Description: "Whether the CloudWatch Scrape Job is enabled or not. Defaults to true.",
				Type:        schema.TypeString,
				Required:    false,
				Default:     true,
			},
			"aws_account_resource_id": {
				Description: "",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regions": {
				Description: "A set of regions that this CloudWatch Scrape Job resource applies to.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"service_configurations": {
				Description: "A set of configurations that dictates what this CloudWatch Scrape Job resource should scrape.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The name of the service to scrape.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"metrics": {
							Description: "A set of metrics to scrape.",
							Type:        schema.TypeSet,
							Required:    true,
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
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
						"scrape_interval_seconds": {
							Description: "The interval in seconds to scrape the service.",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_cloudwatch_scrape_job",
		resourceAWSCWScrapeJobTerraformID,
		schema,
	)
}

func resourceAWSCloudWatchScrapeJobCreate(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	return diags
}

func resourceAWSCloudWatchScrapeJobRead(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	return diags
}

func resourceAWSCloudWatchScrapeJobUpdate(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	return diags
}

func resourceAWSCloudWatchScrapeJobDelete(ctx context.Context, d *schema.ResourceData, c interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
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

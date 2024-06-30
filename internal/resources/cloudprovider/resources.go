package cloudprovider

import (
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type awsCloudWatchScrapeJobModel struct {
	ID                   types.String `tfsdk:"id"`
	StackID              types.String `tfsdk:"stack_id"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	AWSAccountResourceID types.String `tfsdk:"aws_account_resource_id"`
	Regions              types.Set    `tfsdk:"regions"`
	// TODO(tristan): if the grafana provider is update the Terraform v6 schema,
	// we can consider adding additional support to use Set Nested Attributes, instead of Blocks.
	// See https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attribute-types
	ServiceConfigurationBlocks []awsCloudWatchScrapeJobServiceConfigurationModel `tfsdk:"service_configuration"`
}
type awsCloudWatchScrapeJobServiceConfigurationModel struct {
	Name                        types.String                           `tfsdk:"name"`
	Metrics                     []awsCloudWatchScrapeJobMetricModel    `tfsdk:"metric"`
	ScrapeIntervalSeconds       types.Int64                            `tfsdk:"scrape_interval_seconds"`
	ResourceDiscoveryTagFilters []awsCloudWatchScrapeJobTagFilterModel `tfsdk:"resource_discovery_tag_filter"`
	TagsToAddToMetrics          types.Set                              `tfsdk:"tags_to_add_to_metrics"`
	IsCustomNamespace           types.Bool                             `tfsdk:"is_custom_namespace"`
}
type awsCloudWatchScrapeJobMetricModel struct {
	Name       types.String `tfsdk:"name"`
	Statistics types.Set    `tfsdk:"statistics"`
}
type awsCloudWatchScrapeJobTagFilterModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*cloudproviderapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	return client.CloudProviderAPI, nil
}

func withClientForDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) (*cloudproviderapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected DataSource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	return client.CloudProviderAPI, nil
}

var DataSources = []*common.DataSource{
	makeDataSourceAWSAccount(),
	makeDatasourceAWSCloudWatchScrapeJob(),
	makeDatasourceAWSCloudWatchScrapeJobs(),
}

var Resources = []*common.Resource{
	makeResourceAWSAccount(),
	makeResourceAWSCloudWatchScrapeJob(),
}

// TestAWSCloudWatchScrapeJobData is only temporarily exported here until
// we have the resource handlers talking to the real API.
// TODO(tristan): move this to test package and unexport
// once we're using the actual API for interactions.
var TestAWSCloudWatchScrapeJobData = cloudproviderapi.AWSCloudWatchScrapeJob{
	StackID:              "001",
	Name:                 "test-scrape-job",
	Enabled:              true,
	AWSAccountResourceID: "1",
	Regions:              []string{"us-east-1", "us-east-2", "us-west-1"},
	ServiceConfigurations: []cloudproviderapi.AWSCloudWatchServiceConfiguration{
		{
			Name: "AWS/EC2",
			Metrics: []cloudproviderapi.AWSCloudWatchMetric{
				{
					Name:       "aws_ec2_cpuutilization",
					Statistics: []string{"Average"},
				},
				{
					Name:       "aws_ec2_status_check_failed",
					Statistics: []string{"Maximum"},
				},
			},
			ResourceDiscoveryTagFilters: []cloudproviderapi.AWSCloudWatchTagFilter{
				{
					Key:   "k8s.io/cluster-autoscaler/enabled",
					Value: "true",
				},
			},
			TagsToAddToMetrics: []string{"eks:cluster-name"},
		},
		{
			Name:                  "CoolApp",
			ScrapeIntervalSeconds: 300,
			Metrics: []cloudproviderapi.AWSCloudWatchMetric{
				{
					Name:       "CoolMetric",
					Statistics: []string{"Maximum", "Sum"},
				},
			},
			IsCustomNamespace: true,
		},
	},
}

package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *cloudproviderapi.Client) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).CloudProviderAPI
		if client == nil {
			return diag.Errorf("the Cloud Provider API client is required for this resource. Set the cloud_provider_access_token provider attribute")
		}
		return f(ctx, d, client)
	}
}

var DataSources = []*common.DataSource{
	datasourceAWSAccount(),
	datasourceAWSCloudWatchScrapeJob(),
	datasourceAWSCloudWatchScrapeJobs(),
}

var Resources = []*common.Resource{
	resourceAWSAccount(),
	resourceAWSCloudWatchScrapeJob(),
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

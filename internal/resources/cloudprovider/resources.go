package cloudprovider

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func withClient(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*cloudproviderapi.Client, error) {
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

var DataSources = []*common.DataSource{
	datasourceAWSAccount(),
	datasourceAWSCloudWatchScrapeJob(),
	datasourceAWSCloudWatchScrapeJobs(),
}

var Resources = []*common.Resource{
	makeResourceAWSAccount(),
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

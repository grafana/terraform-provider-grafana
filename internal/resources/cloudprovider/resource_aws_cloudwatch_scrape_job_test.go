package cloudprovider_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const testStackID = "1"

var testAWSCloudWatchScrapeJobData = cloudproviderapi.AWSCloudWatchScrapeJobRequest{
	Name:                  "test-scrape-job",
	Enabled:               true,
	RegionsSubsetOverride: []string{"eu-west-1", "us-east-1", "us-east-2"},
	ExportTags:            true,
	Services: []cloudproviderapi.AWSCloudWatchService{
		{
			Name:                  "AWS/EC2",
			ScrapeIntervalSeconds: 300,
			Metrics: []cloudproviderapi.AWSCloudWatchMetric{
				{
					Name:       "CPUUtilization",
					Statistics: []string{"Average"},
				},
				{
					Name:       "StatusCheckFailed",
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
	},
	CustomNamespaces: []cloudproviderapi.AWSCloudWatchCustomNamespace{
		{
			Name:                  "CoolApp",
			ScrapeIntervalSeconds: 300,
			Metrics: []cloudproviderapi.AWSCloudWatchMetric{
				{
					Name:       "CoolMetric",
					Statistics: []string{"Maximum", "Sum"},
				},
			},
		},
	},
}

func TestAccResourceAWSCloudWatchScrapeJob(t *testing.T) {
	t.Skip("Skipping test until we have a valid test case")
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: func() resource.TestCheckFunc {
			return func(s *terraform.State) error {
				return nil
			}
		}(),
		Steps: []resource.TestStep{
			{
				Config: awsCloudWatchScrapeJobResourceData(testStackID,
					testAWSCloudWatchScrapeJobData.Name,
					testAWSCloudWatchScrapeJobData.AWSAccountResourceID,
					regionsString(testAWSCloudWatchScrapeJobData.RegionsSubsetOverride),
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", testStackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", testAWSCloudWatchScrapeJobData.Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "aws_account_resource_id", testAWSCloudWatchScrapeJobData.AWSAccountResourceID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.RegionsSubsetOverride))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.0", testAWSCloudWatchScrapeJobData.RegionsSubsetOverride[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.1", testAWSCloudWatchScrapeJobData.RegionsSubsetOverride[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.2", testAWSCloudWatchScrapeJobData.RegionsSubsetOverride[2]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "export_tags", fmt.Sprintf("%t", testAWSCloudWatchScrapeJobData.ExportTags)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.name", testAWSCloudWatchScrapeJobData.Services[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.name", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.Services[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.key", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.value", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.0", testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].ScrapeIntervalSeconds)),
				),
			},
			// update to unset optional services field
			{
				Config: awsCloudWatchScrapeJobResourceData(testStackID,
					testAWSCloudWatchScrapeJobData.Name,
					testAWSCloudWatchScrapeJobData.AWSAccountResourceID,
					regionsString(testAWSCloudWatchScrapeJobData.RegionsSubsetOverride),
					"",
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				),
				Check: resource.ComposeTestCheckFunc(
					// expect this to be stored in the state as an empty list, not null
					resource.TestCheckResourceAttrSet("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", "0"),
				),
			},
			// update to re-add services but unset optional custom namespaces field
			{
				Config: awsCloudWatchScrapeJobResourceData(testStackID,
					testAWSCloudWatchScrapeJobData.Name,
					testAWSCloudWatchScrapeJobData.AWSAccountResourceID,
					regionsString(testAWSCloudWatchScrapeJobData.RegionsSubsetOverride),
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					"",
				),
				Check: resource.ComposeTestCheckFunc(
					// expect this to be stored in the state as an empty list, not null
					resource.TestCheckResourceAttrSet("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.#", "0"),

					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.name", testAWSCloudWatchScrapeJobData.Services[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.name", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.Services[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.key", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.value", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.0", testAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics[0]),
				),
			},
			// update to unset optional tags_to_add_for_metrics field in service block
			{
				Config: awsCloudWatchScrapeJobResourceData(testStackID,
					testAWSCloudWatchScrapeJobData.Name,
					testAWSCloudWatchScrapeJobData.AWSAccountResourceID,
					regionsString(testAWSCloudWatchScrapeJobData.RegionsSubsetOverride),
					func() string {
						svcs := testAWSCloudWatchScrapeJobData.Services
						if len(svcs) == 0 {
							return ""
						}
						svc := svcs[0]
						svc.TagsToAddToMetrics = []string{}
						return servicesString([]cloudproviderapi.AWSCloudWatchService{svc})
					}(),
					"",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.name", testAWSCloudWatchScrapeJobData.Services[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.name", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.Services[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.key", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.value", testAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Value),
					// expect this to be stored in the state as an empty list, not null
					resource.TestCheckResourceAttrSet("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.#", "0"),
				),
			},
		},
	})
}

func awsCloudWatchScrapeJobResourceData(stackID, jobName, awsAccountResourceID, regionsSubsetOverrideString, servicesString, customNamespacesString string) string {
	data := fmt.Sprintf(`
resource "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id = "%[1]s"
  name = "%[2]s"
  aws_account_resource_id = "%[3]s"
  regions_subset_override = [%[4]s]
	export_tags = true
  dynamic "service" {
    for_each = [%[5]s]
    content {
      name = service.value.name
      dynamic "metric" {
        for_each = service.value.metrics
        content {
          name = metric.value.name
          statistics = metric.value.statistics
        }
      }
      scrape_interval_seconds = service.value.scrape_interval_seconds
      dynamic "resource_discovery_tag_filter" {
        for_each = service.value.resource_discovery_tag_filters
        content {
          key = resource_discovery_tag_filter.value.key
          value = resource_discovery_tag_filter.value.value
        }
      
      }
      tags_to_add_to_metrics = service.value.tags_to_add_to_metrics
    }
  }
  dynamic "custom_namespace" {
    for_each = [%[6]s]
    content {
      name = custom_namespace.value.name
      dynamic "metric" {
        for_each = custom_namespace.value.metrics
        content {
          name = metric.value.name
          statistics = metric.value.statistics
        }
      }
      scrape_interval_seconds = custom_namespace.value.scrape_interval_seconds
    }
  }
}
`,
		stackID, jobName, awsAccountResourceID, regionsSubsetOverrideString, servicesString, customNamespacesString,
	)

	return data
}

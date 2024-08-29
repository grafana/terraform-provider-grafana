package cloudprovider_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudprovider"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceAWSCloudWatchScrapeJob(t *testing.T) {
	// TODO(tristan): switch to CloudInstanceTestsEnabled
	// as part of https://github.com/grafana/grafana-aws-app/issues/381
	t.Skip("not yet implemented. see TODO comment.")
	// testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		IsUnitTest:               true,
		// TODO(tristan): actually check for resource existence
		CheckDestroy: func() resource.TestCheckFunc {
			return func(s *terraform.State) error {
				return nil
			}
		}(),
		Steps: []resource.TestStep{
			{
				Config: awsCloudWatchScrapeJobResourceData(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", cloudprovider.TestStackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", cloudprovider.TestAWSCloudWatchScrapeJobData.Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "aws_account_resource_id", cloudprovider.TestAWSCloudWatchScrapeJobData.AWSAccountResourceID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Regions))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.0", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.1", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.2", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[2]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "export_tags", fmt.Sprintf("%t", cloudprovider.TestAWSCloudWatchScrapeJobData.ExportTags)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "disabled_reason", cloudprovider.TestAWSCloudWatchScrapeJobData.DisabledReason),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Services))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.metric.0.statistics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.scrape_interval_seconds", fmt.Sprintf("%d", cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.key", cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.resource_discovery_tag_filter.0.value", cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.Services[0].TagsToAddToMetrics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.scrape_interval_seconds", fmt.Sprintf("%d", cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces[0].ScrapeIntervalSeconds)),
				),
			},
		},
	})
}

func awsCloudWatchScrapeJobResourceData() string {
	data := fmt.Sprintf(`
resource "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id = "%[1]s"
  name = "%[2]s"
  aws_account_resource_id = "%[3]s"
  regions = [%[4]s]
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
		cloudprovider.TestStackID,
		cloudprovider.TestAWSCloudWatchScrapeJobData.Name,
		cloudprovider.TestAWSCloudWatchScrapeJobData.AWSAccountResourceID,
		regionsString(cloudprovider.TestAWSCloudWatchScrapeJobData.Regions),
		servicesString(cloudprovider.TestAWSCloudWatchScrapeJobData.Services),
		customNamespacesString(cloudprovider.TestAWSCloudWatchScrapeJobData.CustomNamespaces),
	)

	return data
}

func servicesString(svcs []cloudproviderapi.AWSCloudWatchService) string {
	if len(svcs) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	for _, svc := range svcs {
		fmt.Fprintf(b, "\n\t\t")
		fmt.Fprintf(b, `{
			name = "%[1]s",
			metrics = [%[2]s],
			scrape_interval_seconds = %[3]d,
			resource_discovery_tag_filters = [%[4]s],
			tags_to_add_to_metrics = [%[5]s],
		},`,
			svc.Name,
			metricsString(svc.Metrics),
			svc.ScrapeIntervalSeconds,
			tagFiltersString(svc.ResourceDiscoveryTagFilters),
			tagsString(svc.TagsToAddToMetrics),
		)
	}
	fmt.Fprintf(b, "\n\t\t")
	return b.String()
}

func customNamespacesString(customNamespaces []cloudproviderapi.AWSCloudWatchCustomNamespace) string {
	if len(customNamespaces) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	for _, customNamespace := range customNamespaces {
		fmt.Fprintf(b, "\n\t\t")
		fmt.Fprintf(b, `{
			name = "%[1]s",
			metrics = [%[2]s],
			scrape_interval_seconds = %[3]d,
		},`,
			customNamespace.Name,
			metricsString(customNamespace.Metrics),
			customNamespace.ScrapeIntervalSeconds,
		)
	}
	fmt.Fprintf(b, "\n\t\t")
	return b.String()
}

func metricsString(metrics []cloudproviderapi.AWSCloudWatchMetric) string {
	if len(metrics) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	for _, metric := range metrics {
		fmt.Fprintf(b, "\n\t\t\t")
		fmt.Fprintf(b, `{
				name = "%[1]s",
				statistics = [%[2]s],
			},`,
			metric.Name,
			statisticsString(metric.Statistics),
		)
	}
	fmt.Fprintf(b, "\n\t\t\t")
	return b.String()
}

func statisticsString(stats []string) string {
	if len(stats) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "\n\t\t\t\t\t")
	for _, stat := range stats {
		fmt.Fprintf(b, "\"%s\",", stat)
	}
	fmt.Fprintf(b, "\n\t\t\t\t")
	return b.String()
}

func tagFiltersString(filters []cloudproviderapi.AWSCloudWatchTagFilter) string {
	if len(filters) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "\n\t\t\t")
	for _, filter := range filters {
		fmt.Fprintf(b, `{
				key = "%[1]s",
				value = "%[2]s",
			},`,
			filter.Key,
			filter.Value,
		)
	}
	fmt.Fprintf(b, "\n\t\t\t")
	return b.String()
}

func tagsString(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "\n\t\t\t\t")
	for _, tag := range tags {
		fmt.Fprintf(b, "\"%s\",", tag)
	}
	fmt.Fprintf(b, "\n\t\t\t")
	return b.String()
}

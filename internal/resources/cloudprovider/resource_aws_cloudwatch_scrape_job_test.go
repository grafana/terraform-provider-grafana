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
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", cloudprovider.TestAWSCloudWatchScrapeJobData.StackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", cloudprovider.TestAWSCloudWatchScrapeJobData.Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "aws_account_resource_id", cloudprovider.TestAWSCloudWatchScrapeJobData.AWSAccountResourceID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.Regions))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.0", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.1", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions.2", cloudprovider.TestAWSCloudWatchScrapeJobData.Regions[2]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.metric.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.metric.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.metric.0.statistics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.metric.0.statistics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.scrape_interval_seconds", fmt.Sprintf("%d", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.resource_discovery_tag_filter.0.key", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.resource_discovery_tag_filter.0.value", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.tags_to_add_to_metrics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].TagsToAddToMetrics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.0.is_custom_namespace", fmt.Sprintf("%t", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[0].IsCustomNamespace)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.metric.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.metric.0.name", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.metric.0.statistics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.metric.0.statistics.0", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.scrape_interval_seconds", fmt.Sprintf("%d", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.tags_to_add_to_metrics.#", fmt.Sprintf("%d", len(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].TagsToAddToMetrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service_configuration.1.is_custom_namespace", fmt.Sprintf("%t", cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations[1].IsCustomNamespace)),
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
  dynamic "service_configuration" {
    for_each = [%[5]s]
    content {
      name = service_configuration.value.name
      dynamic "metric" {
        for_each = service_configuration.value.metrics
        content {
          name = metric.value.name
          statistics = metric.value.statistics
        }
      }
      scrape_interval_seconds = service_configuration.value.scrape_interval_seconds
      dynamic "resource_discovery_tag_filter" {
        for_each = service_configuration.value.resource_discovery_tag_filters
        content {
          key = resource_discovery_tag_filter.value.key
          value = resource_discovery_tag_filter.value.value
        }
      }
      tags_to_add_to_metrics = service_configuration.value.tags_to_add_to_metrics
      is_custom_namespace = service_configuration.value.is_custom_namespace
		}
  }
}
`,
		cloudprovider.TestAWSCloudWatchScrapeJobData.StackID,
		cloudprovider.TestAWSCloudWatchScrapeJobData.Name,
		cloudprovider.TestAWSCloudWatchScrapeJobData.AWSAccountResourceID,
		regionsString(cloudprovider.TestAWSCloudWatchScrapeJobData.Regions),
		serviceConfigurationsString(cloudprovider.TestAWSCloudWatchScrapeJobData.ServiceConfigurations),
	)

	return data
}

func serviceConfigurationsString(svcConfigs []cloudproviderapi.AWSCloudWatchServiceConfiguration) string {
	if len(svcConfigs) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	for _, svcConfig := range svcConfigs {
		fmt.Fprintf(b, "\n\t\t")
		fmt.Fprintf(b, `{
			name = "%[1]s",
			metrics = [%[2]s],
			scrape_interval_seconds = %[3]d,
			resource_discovery_tag_filters = [%[4]s],
			tags_to_add_to_metrics = [%[5]s],
			is_custom_namespace = %[6]t,
		},`,
			svcConfig.Name,
			metricsString(svcConfig.Metrics),
			svcConfig.ScrapeIntervalSeconds,
			tagFiltersString(svcConfig.ResourceDiscoveryTagFilters),
			tagsString(svcConfig.TagsToAddToMetrics),
			svcConfig.IsCustomNamespace,
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

package cloudprovider_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAWSCloudWatchScrapeJobData = cloudproviderapi.AWSCloudWatchScrapeJobResponse{
	Regions:    []string{"us-east-1", "us-east-2", "us-west-1"},
	ExportTags: true,
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
	StaticLabels: map[string]string{
		"label1": "value1",
		"label2": "value2",
	},
}

func TestAccResourceAWSCloudWatchScrapeJob(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	testCfg := makeTestConfig(t)

	jobName := "test-job" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	var gotJob cloudproviderapi.AWSCloudWatchScrapeJobResponse

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					regionsString([]string{testAWSCloudWatchScrapeJobData.Regions[0], testAWSCloudWatchScrapeJobData.Regions[1]}),
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				),
				Check: resource.ComposeTestCheckFunc(
					checkAWSCloudWatchScrapeJobResourceExists("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", testCfg.stackID, &gotJob),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "enabled", "false"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "disabled_reason", "disabled_by_user"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "aws_account_resource_id", testCfg.accountID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.#", "2"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.0", testAWSCloudWatchScrapeJobData.Regions[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.1", testAWSCloudWatchScrapeJobData.Regions[1]),
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
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.0.label", "label1"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.0.value", testAWSCloudWatchScrapeJobData.StaticLabels["label1"]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.1.label", "label2"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "static_label.1.value", testAWSCloudWatchScrapeJobData.StaticLabels["label2"]),
				),
			},
			// update to remove regions_subset_override so that the account's regions are used instead
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					"",
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "regions_subset_override.#", "0"),
				),
			},
			// update to enable the job
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					true,
					testCfg.accountID,
					"",
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "disabled_reason", ""),
				),
			},
			// update to disable the job again
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					"",
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "enabled", "false"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "disabled_reason", "disabled_by_user"),
				),
			},
			// update to unset optional services field
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					"",
					"",
					customNamespacesString(testAWSCloudWatchScrapeJobData.CustomNamespaces),
				),
				Check: resource.ComposeTestCheckFunc(
					// expect this to be stored in the state as an empty list, not null
					resource.TestCheckResourceAttrSet("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", "0"),

					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.name", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.metric.0.statistics.0", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].Metrics[0].Statistics[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSCloudWatchScrapeJobData.CustomNamespaces[0].ScrapeIntervalSeconds)),
				),
			},
			// update to re-add services but unset optional custom namespaces field
			{
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					"",
					servicesString(testAWSCloudWatchScrapeJobData.Services),
					"",
				),
				Check: resource.ComposeTestCheckFunc(
					// expect this to be stored in the state as an empty list, not null
					resource.TestCheckResourceAttrSet("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "custom_namespace.#"),
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
				Config: awsCloudWatchScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					"",
					func() string {
						svc := testAWSCloudWatchScrapeJobData.Services[0]
						svc.TagsToAddToMetrics = []string{}
						return servicesString([]cloudproviderapi.AWSCloudWatchService{svc})
					}(),
					"",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.#", fmt.Sprintf("%d", len(testAWSCloudWatchScrapeJobData.Services))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.name", testAWSCloudWatchScrapeJobData.Services[0].Name),
					// expect this to be stored in the state as an empty list, not null
					resource.TestCheckResourceAttrSet("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.#"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_cloudwatch_scrape_job.test", "service.0.tags_to_add_to_metrics.#", "0"),
				),
			},
			{
				ResourceName:      "grafana_cloud_provider_aws_cloudwatch_scrape_job.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: checkAWSCloudWatchScrapeJobResourceDestroy(testCfg.stackID, &gotJob),
	})
}

func checkAWSCloudWatchScrapeJobResourceExists(rn string, stackID string, job *cloudproviderapi.AWSCloudWatchScrapeJobResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		parts := strings.SplitN(rs.Primary.ID, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("Invalid ID: %s", rs.Primary.ID)
		}
		gotStackID := parts[0]
		jobName := parts[1]

		if gotStackID == "" {
			return fmt.Errorf("stack id not set")
		}

		if gotStackID != stackID {
			return fmt.Errorf("stack id mismatch, expected %s, but %s was in the TF state", stackID, gotStackID)
		}

		if jobName == "" {
			return fmt.Errorf("jobName not set")
		}

		client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
		gotJob, err := client.GetAWSCloudWatchScrapeJob(context.Background(), stackID, jobName)
		if err != nil {
			return fmt.Errorf("error getting account: %s", err)
		}

		*job = gotJob

		return nil
	}
}

func checkAWSCloudWatchScrapeJobResourceDestroy(stackID string, job *cloudproviderapi.AWSCloudWatchScrapeJobResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if job.Name == "" {
			return fmt.Errorf("checking deletion of empty job name")
		}

		client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
		_, err := client.GetAWSCloudWatchScrapeJob(context.Background(), stackID, job.Name)
		if err == nil {
			return fmt.Errorf("job still exists")
		} else if !common.IsNotFoundError(err) {
			return fmt.Errorf("unexpected error retrieving job: %s", err)
		}

		return nil
	}
}

func awsCloudWatchScrapeJobResourceData(stackID string, jobName string, enabled bool, awsAccountResourceID string, regionsSubsetOverrideString string, servicesString string, customNamespacesString string) string {
	data := fmt.Sprintf(`
resource "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id = "%[1]s"
  name = "%[2]s"
  enabled = %[3]t
  aws_account_resource_id = "%[4]s"
  regions_subset_override = [%[5]s]
  export_tags = true
  dynamic "service" {
    for_each = [%[6]s]
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
    for_each = [%[7]s]
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
		stackID, jobName, enabled, awsAccountResourceID, regionsSubsetOverrideString, servicesString, customNamespacesString,
	)

	return data
}

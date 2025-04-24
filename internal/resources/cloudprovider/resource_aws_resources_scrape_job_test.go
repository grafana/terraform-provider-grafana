package cloudprovider_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/zclconf/go-cty/cty"
)

var testAWSResourceMetadataScrapeJobData = cloudproviderapi.AWSResourceMetadataScrapeJobRequest{
	RegionsSubsetOverride: []string{"us-east-1", "us-east-2", "us-west-1"},
	Services: []cloudproviderapi.AWSResourceMetadataService{
		{
			Name:                  "AWS/EC2",
			ScrapeIntervalSeconds: 300,
			ResourceDiscoveryTagFilters: []cloudproviderapi.AWSResourceMetadataTagFilter{
				{
					Key:   "k8s.io/cluster-autoscaler/enabled",
					Value: "true",
				},
			},
		},
	},
	StaticLabels: map[string]string{
		"label1": "value1",
		"label2": "value2",
	},
}

func TestAccResourceAWSResourceMetadataScrapeJob(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	testCfg := makeTestConfig(t)

	jobName := "test-job" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	var gotJob cloudproviderapi.AWSResourceMetadataScrapeJobResponse

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: AWSResourceMetadataScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					testAWSResourceMetadataScrapeJobData,
				),
				Check: resource.ComposeTestCheckFunc(
					checkAWSResourceMetadataScrapeJobResourceExists("grafana_cloud_provider_aws_resources_scrape_job.test", testCfg.stackID, &gotJob),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "enabled", "false"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "disabled_reason", "disabled_by_user"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "aws_account_resource_id", testCfg.accountID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "regions_subset_override.#", "2"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "regions_subset_override.0", testAWSResourceMetadataScrapeJobData.RegionsSubsetOverride[0]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "regions_subset_override.1", testAWSResourceMetadataScrapeJobData.RegionsSubsetOverride[1]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.#", fmt.Sprintf("%d", len(testAWSResourceMetadataScrapeJobData.Services))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.name", testAWSResourceMetadataScrapeJobData.Services[0].Name),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.scrape_interval_seconds", fmt.Sprintf("%d", testAWSResourceMetadataScrapeJobData.Services[0].ScrapeIntervalSeconds)),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.resource_discovery_tag_filter.#", fmt.Sprintf("%d", len(testAWSResourceMetadataScrapeJobData.Services[0].ResourceDiscoveryTagFilters))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.resource_discovery_tag_filter.0.key", testAWSResourceMetadataScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Key),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.resource_discovery_tag_filter.0.value", testAWSResourceMetadataScrapeJobData.Services[0].ResourceDiscoveryTagFilters[0].Value),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "static_labels.label1", testAWSResourceMetadataScrapeJobData.StaticLabels["label1"]),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "static_labels.label2", testAWSResourceMetadataScrapeJobData.StaticLabels["label2"]),
				),
			},
			// update to remove regions_subset_override so that the account's regions are used instead
			{
				Config: AWSResourceMetadataScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					testAWSResourceMetadataScrapeJobData,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "regions_subset_override.#", "0"),
				),
			},
			// update to enable the job
			{
				Config: AWSResourceMetadataScrapeJobResourceData(testCfg.stackID,
					jobName,
					true,
					testCfg.accountID,
					testAWSResourceMetadataScrapeJobData,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "disabled_reason", ""),
				),
			},
			// update to disable the job again
			{
				Config: AWSResourceMetadataScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					testAWSResourceMetadataScrapeJobData,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "stack_id", testCfg.stackID),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "name", jobName),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "enabled", "false"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "disabled_reason", "disabled_by_user"),
				),
			},
			// update to unset optional tags_to_add_for_metrics field in service block
			{
				Config: AWSResourceMetadataScrapeJobResourceData(testCfg.stackID,
					jobName,
					false,
					testCfg.accountID,
					func() cloudproviderapi.AWSResourceMetadataScrapeJobRequest {
						req := testAWSResourceMetadataScrapeJobData
						req.Services[0].ResourceDiscoveryTagFilters = nil
						return req
					}(),
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.#", fmt.Sprintf("%d", len(testAWSResourceMetadataScrapeJobData.Services))),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.name", testAWSResourceMetadataScrapeJobData.Services[0].Name),
					// expect this to be stored in the state as an empty list, not null
					resource.TestCheckResourceAttrSet("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.service.0.resource_discovery_tag_filters.#"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_aws_resources_scrape_job.test", "service.0.service.0.resource_discovery_tag_filter.#", "0"),
				),
			},
			{
				ResourceName:      "grafana_cloud_provider_aws_resources_scrape_job.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: checkAWSResourceMetadataScrapeJobResourceDestroy(testCfg.stackID, &gotJob),
	})
}

func checkAWSResourceMetadataScrapeJobResourceExists(rn string, stackID string, job *cloudproviderapi.AWSResourceMetadataScrapeJobResponse) resource.TestCheckFunc {
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
		gotJob, err := client.GetAWSResourceMetadataScrapeJob(context.Background(), stackID, jobName)
		if err != nil {
			return fmt.Errorf("error getting account: %s", err)
		}

		*job = gotJob

		return nil
	}
}

func checkAWSResourceMetadataScrapeJobResourceDestroy(stackID string, job *cloudproviderapi.AWSResourceMetadataScrapeJobResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if job.Name == "" {
			return fmt.Errorf("checking deletion of empty job name")
		}

		client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
		_, err := client.GetAWSResourceMetadataScrapeJob(context.Background(), stackID, job.Name)
		if err == nil {
			return fmt.Errorf("job still exists")
		} else if !common.IsNotFoundError(err) {
			return fmt.Errorf("unexpected error retrieving job: %s", err)
		}

		return nil
	}
}

func AWSResourceMetadataScrapeJobResourceData(stackID string, jobName string, enabled bool, awsAccountResourceID string, req cloudproviderapi.AWSResourceMetadataScrapeJobRequest) string {
	req.Name = jobName
	req.Enabled = enabled
	req.AWSAccountResourceID = awsAccountResourceID

	hclFile := hclwrite.NewFile()
	jobRes := hclFile.Body().AppendNewBlock("resource", []string{"grafana_cloud_provider_aws_resources_scrape_job", strings.ToLower(toSnakeCase(req.Name))})

	jobBody := jobRes.Body()
	jobBody.SetAttributeValue("name", cty.StringVal(req.Name))
	jobBody.SetAttributeValue("enabled", cty.BoolVal(req.Enabled))
	jobBody.SetAttributeValue("aws_account_resource_id", cty.StringVal(req.AWSAccountResourceID))
	jobBody.SetAttributeValue("stack_id", cty.StringVal(stackID))

	if len(req.RegionsSubsetOverride) > 0 {
		jobBody.SetAttributeValue("regions_subset_override", cty.ListVal(mapCty(req.RegionsSubsetOverride, cty.StringVal)))
	}

	for _, svc := range req.Services {
		svcBody := jobBody.AppendNewBlock("service", nil).Body()
		svcBody.SetAttributeValue("name", cty.StringVal(svc.Name))
		svcBody.SetAttributeValue("scrape_interval_seconds", cty.NumberIntVal(svc.ScrapeIntervalSeconds))
		for _, tagFilter := range svc.ResourceDiscoveryTagFilters {
			svcBody := jobBody.AppendNewBlock("resource_discovery_tag_filter", nil).Body()
			svcBody.SetAttributeValue("key", cty.StringVal(tagFilter.Key))
			svcBody.SetAttributeValue("value", cty.StringVal(tagFilter.Value))
		}
	}

	var b bytes.Buffer
	hclFile.WriteTo(&b)
	return b.String()
}

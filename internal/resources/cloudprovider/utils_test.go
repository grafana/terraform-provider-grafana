package cloudprovider_test

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	stackID     string
	accountID   string
	accountName string
	roleARN     string
}

func makeTestConfig(t require.TestingT) testConfig {
	// Uses a pre-existing account resource so that we don't need to create a new one for every test run
	accountID := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_AWS_ACCOUNT_RESOURCE_ID")
	require.NotEmpty(t, accountID, "GRAFANA_CLOUD_PROVIDER_TEST_AWS_ACCOUNT_RESOURCE_ID must be set")

	roleARN := os.Getenv("GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN")
	require.NotEmpty(t, roleARN, "GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN must be set")

	stackID := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID")
	require.NotEmpty(t, stackID, "GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID must be set")

	// Make sure the account exists and matches the role ARN we expect for testing
	client := testutils.Provider.Meta().(*common.Client).CloudProviderAPI
	gotAccount, err := client.GetAWSAccount(context.Background(), stackID, accountID)
	require.NoError(t, err)
	require.Equal(t, roleARN, gotAccount.RoleARN)

	return testConfig{
		stackID:     stackID,
		accountID:   accountID,
		accountName: gotAccount.Name,
		roleARN:     roleARN,
	}
}

func regionsString(regions []string) string {
	b := new(bytes.Buffer)
	for _, region := range regions {
		fmt.Fprintf(b, "\n\t\t\"%s\",", region)
	}
	fmt.Fprintf(b, "\n\t")
	return b.String()
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

package cloudprovider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mustMetricList(t *testing.T, names ...string) types.List {
	t.Helper()
	elemType := types.ObjectType{AttrTypes: awsCloudWatchScrapeJobMetricTFModel{}.attrTypes()}
	elems := make([]attr.Value, len(names))
	for i, name := range names {
		elems[i] = types.ObjectValueMust(elemType.AttrTypes, map[string]attr.Value{
			"name":       types.StringValue(name),
			"statistics": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("Average")}),
		})
	}
	return types.ListValueMust(elemType, elems)
}

func mustEnhancedMetricList(t *testing.T, names ...string) types.List {
	t.Helper()
	elemType := types.ObjectType{AttrTypes: awsCloudWatchScrapeJobEnhancedMetricTFModel{}.attrTypes()}
	elems := make([]attr.Value, len(names))
	for i, name := range names {
		elems[i] = types.ObjectValueMust(elemType.AttrTypes, map[string]attr.Value{
			"name": types.StringValue(name),
		})
	}
	return types.ListValueMust(elemType, elems)
}

func mustServiceList(t *testing.T, services ...awsCloudWatchScrapeJobServiceTFModel) types.List {
	t.Helper()
	elemType := types.ObjectType{AttrTypes: awsCloudWatchScrapeJobServiceTFModel{}.attrTypes()}
	elems := make([]attr.Value, len(services))
	for i, service := range services {
		elems[i] = types.ObjectValueMust(elemType.AttrTypes, map[string]attr.Value{
			"name":                          service.Name,
			"metric":                        service.Metrics,
			"enhanced_metric":               service.EnhancedMetrics,
			"scrape_interval_seconds":       types.Int64Value(300),
			"resource_discovery_tag_filter": types.ListValueMust(types.ObjectType{AttrTypes: awsCloudWatchScrapeJobTagFilterTFModel{}.attrTypes()}, []attr.Value{}),
			"tags_to_add_to_metrics":        types.SetValueMust(types.StringType, []attr.Value{}),
		})
	}
	return types.ListValueMust(elemType, elems)
}

func TestUnitAWSCloudWatchScrapeJobServiceAtLeastOneMetricOrEnhancedMetricValidator(t *testing.T) {
	ctx := context.Background()

	testCases := map[string]struct {
		services  []awsCloudWatchScrapeJobServiceTFModel
		wantError bool
	}{
		"metric only": {
			services: []awsCloudWatchScrapeJobServiceTFModel{
				{
					Name:            types.StringValue("AWS/EC2"),
					Metrics:         mustMetricList(t, "CPUUtilization"),
					EnhancedMetrics: mustEnhancedMetricList(t),
				},
			},
			wantError: false,
		},
		"enhanced_metric only": {
			services: []awsCloudWatchScrapeJobServiceTFModel{
				{
					Name:            types.StringValue("AWS/EC2"),
					Metrics:         mustMetricList(t),
					EnhancedMetrics: mustEnhancedMetricList(t, "CPUCreditBalance"),
				},
			},
			wantError: false,
		},
		"both metric and enhanced_metric": {
			services: []awsCloudWatchScrapeJobServiceTFModel{
				{
					Name:            types.StringValue("AWS/EC2"),
					Metrics:         mustMetricList(t, "CPUUtilization"),
					EnhancedMetrics: mustEnhancedMetricList(t, "CPUCreditBalance"),
				},
			},
			wantError: false,
		},
		"neither metric nor enhanced_metric": {
			services: []awsCloudWatchScrapeJobServiceTFModel{
				{
					Name:            types.StringValue("AWS/EC2"),
					Metrics:         mustMetricList(t),
					EnhancedMetrics: mustEnhancedMetricList(t),
				},
			},
			wantError: true,
		},
		"one valid one invalid service": {
			services: []awsCloudWatchScrapeJobServiceTFModel{
				{
					Name:            types.StringValue("AWS/EC2"),
					Metrics:         mustMetricList(t, "CPUUtilization"),
					EnhancedMetrics: mustEnhancedMetricList(t),
				},
				{
					Name:            types.StringValue("AWS/RDS"),
					Metrics:         mustMetricList(t),
					EnhancedMetrics: mustEnhancedMetricList(t),
				},
			},
			wantError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			req := validator.ListRequest{
				ConfigValue: mustServiceList(t, tc.services...),
			}
			resp := &validator.ListResponse{}

			awsCloudWatchScrapeJobServiceAtLeastOneMetricOrEnhancedMetricValidator{}.ValidateList(ctx, req, resp)

			if tc.wantError && !resp.Diagnostics.HasError() {
				t.Errorf("expected an error, got none")
			}
			if !tc.wantError && resp.Diagnostics.HasError() {
				t.Errorf("expected no error, got: %s", resp.Diagnostics)
			}
			if tc.wantError && resp.Diagnostics.HasError() && len(resp.Diagnostics.Errors()) != 1 {
				t.Errorf("expected exactly 1 error, got %d: %s", len(resp.Diagnostics.Errors()), resp.Diagnostics)
			}
		})
	}
}

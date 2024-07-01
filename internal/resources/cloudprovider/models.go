package cloudprovider

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type awsCWScrapeJobTFModel struct {
	ID                   types.String `tfsdk:"id"`
	StackID              types.String `tfsdk:"stack_id"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	AWSAccountResourceID types.String `tfsdk:"aws_account_resource_id"`
	Regions              types.Set    `tfsdk:"regions"`
	// TODO(tristan): if the grafana provider is updated to use the Terraform v6 plugin protocol,
	// we can consider adding additional support to use Set Nested Attributes, instead of Blocks.
	// See https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attribute-types
	ServiceConfigurationBlocks []awsCWScrapeJobServiceConfigTFModel `tfsdk:"service_configuration"`
}
type awsCWScrapeJobServiceConfigTFModel struct {
	Name                        types.String                     `tfsdk:"name"`
	Metrics                     []awsCWScrapeJobMetricTFModel    `tfsdk:"metric"`
	ScrapeIntervalSeconds       types.Int64                      `tfsdk:"scrape_interval_seconds"`
	ResourceDiscoveryTagFilters []awsCWScrapeJobTagFilterTFModel `tfsdk:"resource_discovery_tag_filter"`
	TagsToAddToMetrics          types.Set                        `tfsdk:"tags_to_add_to_metrics"`
	IsCustomNamespace           types.Bool                       `tfsdk:"is_custom_namespace"`
}
type awsCWScrapeJobMetricTFModel struct {
	Name       types.String `tfsdk:"name"`
	Statistics types.Set    `tfsdk:"statistics"`
}
type awsCWScrapeJobTagFilterTFModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type awsCWScrapeJobNoDuplicateServiceConfigNamesValidator struct{}

func (v awsCWScrapeJobNoDuplicateServiceConfigNamesValidator) Description(ctx context.Context) string {
	return "No duplicate service configuration names are allowed."
}

func (v awsCWScrapeJobNoDuplicateServiceConfigNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "No duplicate service configuration names are allowed."
}

func (v awsCWScrapeJobNoDuplicateServiceConfigNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	seen := map[string]struct{}{}
	elems := make([]awsCWScrapeJobServiceConfigTFModel, len(req.ConfigValue.Elements()))
	diags := req.ConfigValue.ElementsAs(ctx, &elems, false)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	for _, elem := range elems {
		name := elem.Name.ValueString()
		if _, ok := seen[name]; ok {
			resp.Diagnostics.AddError("Duplicate service configuration name", fmt.Sprintf("Service configuration name %q is duplicated.", name))
		}
		seen[name] = struct{}{}
	}
}

type awsCWScrapeJobNoDuplicateMetricNamesValidator struct{}

func (v awsCWScrapeJobNoDuplicateMetricNamesValidator) Description(ctx context.Context) string {
	return "Metric names must be unique (case-insensitive) within the same service configuration."
}

func (v awsCWScrapeJobNoDuplicateMetricNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "Metric names must be unique (case-insensitive) within the same service configuration."
}

func (v awsCWScrapeJobNoDuplicateMetricNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	seen := map[string]struct{}{}
	elems := make([]awsCWScrapeJobMetricTFModel, len(req.ConfigValue.Elements()))
	diags := req.ConfigValue.ElementsAs(ctx, &elems, false)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	for _, elem := range elems {
		name := elem.Name.ValueString()
		if _, ok := seen[name]; ok {
			resp.Diagnostics.AddError("Duplicate metric name for service configuration", fmt.Sprintf("Metric name %q is duplicated within the service configuration.", name))
		}
		seen[name] = struct{}{}
	}
}

func convertScrapeJobClientModelToTFModel(ctx context.Context, scrapeJobData cloudproviderapi.AWSCloudWatchScrapeJob) (*awsCWScrapeJobTFModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := &awsCWScrapeJobTFModel{
		ID:                   types.StringValue(resourceAWSCloudWatchScrapeJobTerraformID.Make(scrapeJobData.StackID, scrapeJobData.Name)),
		StackID:              types.StringValue(scrapeJobData.StackID),
		Name:                 types.StringValue(scrapeJobData.Name),
		Enabled:              types.BoolValue(scrapeJobData.Enabled),
		AWSAccountResourceID: types.StringValue(scrapeJobData.AWSAccountResourceID),
	}

	regions, diags := types.SetValueFrom(ctx, basetypes.StringType{}, &scrapeJobData.Regions)
	diags.Append(diags...)
	if diags.HasError() {
		return nil, conversionDiags
	}
	converted.Regions = regions

	for _, serviceConfigData := range scrapeJobData.ServiceConfigurations {
		serviceConfig := awsCWScrapeJobServiceConfigTFModel{
			Name:                  types.StringValue(serviceConfigData.Name),
			ScrapeIntervalSeconds: types.Int64Value(serviceConfigData.ScrapeIntervalSeconds),
			IsCustomNamespace:     types.BoolValue(serviceConfigData.IsCustomNamespace),
		}

		metricsData := make([]awsCWScrapeJobMetricTFModel, len(serviceConfigData.Metrics))
		for i, metricData := range serviceConfigData.Metrics {
			metricsData[i] = awsCWScrapeJobMetricTFModel{
				Name: types.StringValue(metricData.Name),
			}
			statistics, diags := types.SetValueFrom(ctx, basetypes.StringType{}, &metricData.Statistics)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return nil, conversionDiags
			}
			metricsData[i].Statistics = statistics
		}
		serviceConfig.Metrics = metricsData

		tagFiltersData := make([]awsCWScrapeJobTagFilterTFModel, len(serviceConfigData.ResourceDiscoveryTagFilters))
		for i, tagFilterData := range serviceConfigData.ResourceDiscoveryTagFilters {
			tagFiltersData[i] = awsCWScrapeJobTagFilterTFModel{
				Key:   types.StringValue(tagFilterData.Key),
				Value: types.StringValue(tagFilterData.Value),
			}
		}
		serviceConfig.ResourceDiscoveryTagFilters = tagFiltersData

		tagsToAdd, diags := types.SetValueFrom(ctx, basetypes.StringType{}, &serviceConfigData.TagsToAddToMetrics)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return nil, conversionDiags
		}
		serviceConfig.TagsToAddToMetrics = tagsToAdd

		converted.ServiceConfigurationBlocks = append(converted.ServiceConfigurationBlocks, serviceConfig)
	}

	return converted, conversionDiags
}

package cloudprovider

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type awsCloudWatchScrapeJobTFResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	StackID               types.String `tfsdk:"stack_id"`
	Name                  types.String `tfsdk:"name"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	AWSAccountResourceID  types.String `tfsdk:"aws_account_resource_id"`
	RegionsSubsetOverride types.Set    `tfsdk:"regions_subset_override"`
	ExportTags            types.Bool   `tfsdk:"export_tags"`
	DisabledReason        types.String `tfsdk:"disabled_reason"`
	// TODO(tristan): if the grafana provider is updated to use the Terraform v6 plugin protocol,
	// we can consider adding additional support to use Set Nested Attributes, instead of Blocks.
	// See https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attribute-types
	Services         types.List `tfsdk:"service"`
	CustomNamespaces types.List `tfsdk:"custom_namespace"`
	StaticLabels     types.Map  `tfsdk:"static_labels"`
}

type awsCloudWatchScrapeJobTFDataSourceModel struct {
	ID                        types.String `tfsdk:"id"`
	StackID                   types.String `tfsdk:"stack_id"`
	Name                      types.String `tfsdk:"name"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	AWSAccountResourceID      types.String `tfsdk:"aws_account_resource_id"`
	Regions                   types.Set    `tfsdk:"regions"`
	RegionsSubsetOverrideUsed types.Bool   `tfsdk:"regions_subset_override_used"`
	RoleARN                   types.String `tfsdk:"role_arn"`
	ExportTags                types.Bool   `tfsdk:"export_tags"`
	DisabledReason            types.String `tfsdk:"disabled_reason"`
	// TODO(tristan): if the grafana provider is updated to use the Terraform v6 plugin protocol,
	// we can consider adding additional support to use Set Nested Attributes, instead of Blocks.
	// See https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attribute-types
	Services         types.List `tfsdk:"service"`
	CustomNamespaces types.List `tfsdk:"custom_namespace"`
	StaticLabels     types.Map  `tfsdk:"static_labels"`
}

type awsCloudWatchScrapeJobServiceTFModel struct {
	Name                        types.String `tfsdk:"name"`
	Metrics                     types.List   `tfsdk:"metric"`
	EnhancedMetrics             types.List   `tfsdk:"enhanced_metric"`
	ScrapeIntervalSeconds       types.Int64  `tfsdk:"scrape_interval_seconds"`
	ResourceDiscoveryTagFilters types.List   `tfsdk:"resource_discovery_tag_filter"`
	TagsToAddToMetrics          types.Set    `tfsdk:"tags_to_add_to_metrics"`
}

func (m awsCloudWatchScrapeJobServiceTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"metric": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsCloudWatchScrapeJobMetricTFModel{}.attrTypes(),
			},
		},
		"enhanced_metric": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsCloudWatchScrapeJobEnhancedMetricTFModel{}.attrTypes(),
			},
		},
		"scrape_interval_seconds": types.Int64Type,
		"resource_discovery_tag_filter": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsCloudWatchScrapeJobTagFilterTFModel{}.attrTypes(),
			},
		},
		"tags_to_add_to_metrics": types.SetType{
			ElemType: types.StringType,
		},
	}
}

type awsCloudWatchScrapeJobCustomNamespaceTFModel struct {
	Name                  types.String `tfsdk:"name"`
	Metrics               types.List   `tfsdk:"metric"`
	ScrapeIntervalSeconds types.Int64  `tfsdk:"scrape_interval_seconds"`
}

func (m awsCloudWatchScrapeJobCustomNamespaceTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"metric": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsCloudWatchScrapeJobMetricTFModel{}.attrTypes(),
			},
		},
		"scrape_interval_seconds": types.Int64Type,
	}
}

type awsCloudWatchScrapeJobMetricTFModel struct {
	Name       types.String `tfsdk:"name"`
	Statistics types.Set    `tfsdk:"statistics"`
}

func (m awsCloudWatchScrapeJobMetricTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"statistics": types.SetType{
			ElemType: types.StringType,
		},
	}
}

type awsCloudWatchScrapeJobEnhancedMetricTFModel struct {
	Name types.String `tfsdk:"name"`
}

func (m awsCloudWatchScrapeJobEnhancedMetricTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
	}
}

type awsCloudWatchScrapeJobTagFilterTFModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func (m awsCloudWatchScrapeJobTagFilterTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
	}
}

type awsCloudWatchScrapeJobNoDuplicateServiceNamesValidator struct{}

func (v awsCloudWatchScrapeJobNoDuplicateServiceNamesValidator) Description(ctx context.Context) string {
	return "No duplicate service names are allowed."
}

func (v awsCloudWatchScrapeJobNoDuplicateServiceNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "No duplicate service names are allowed."
}

func (v awsCloudWatchScrapeJobNoDuplicateServiceNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	var services []awsCloudWatchScrapeJobServiceTFModel
	diags := req.ConfigValue.ElementsAs(ctx, &services, true)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	seen := map[string]struct{}{}
	for _, service := range services {
		name := service.Name.ValueString()
		if _, ok := seen[name]; ok {
			resp.Diagnostics.AddError("Duplicate service name", fmt.Sprintf("Service name %q is duplicated.", name))
		}
		seen[name] = struct{}{}
	}
}

type awsCloudWatchScrapeJobNoDuplicateCustomNamespaceNamesValidator struct{}

func (v awsCloudWatchScrapeJobNoDuplicateCustomNamespaceNamesValidator) Description(ctx context.Context) string {
	return "No duplicate custom namespace names are allowed."
}

func (v awsCloudWatchScrapeJobNoDuplicateCustomNamespaceNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "No duplicate custom namespace names are allowed."
}

func (v awsCloudWatchScrapeJobNoDuplicateCustomNamespaceNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	var customNamespaces []awsCloudWatchScrapeJobCustomNamespaceTFModel
	diags := req.ConfigValue.ElementsAs(ctx, &customNamespaces, true)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	seen := map[string]struct{}{}
	for _, customNamespace := range customNamespaces {
		name := customNamespace.Name.ValueString()
		if _, ok := seen[name]; ok {
			resp.Diagnostics.AddError("Duplicate custom namespace name", fmt.Sprintf("Custom namespace name %q is duplicated.", name))
		}
		seen[name] = struct{}{}
	}
}

type awsCloudWatchScrapeJobNoDuplicateMetricNamesValidator struct{}

func (v awsCloudWatchScrapeJobNoDuplicateMetricNamesValidator) Description(ctx context.Context) string {
	return "Metric names must be unique (case-insensitive) within the same service or custom namespace."
}

func (v awsCloudWatchScrapeJobNoDuplicateMetricNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "Metric names must be unique (case-insensitive) within the same service or custom namespace."
}

func (v awsCloudWatchScrapeJobNoDuplicateMetricNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	seen := map[string]struct{}{}
	elems := make([]awsCloudWatchScrapeJobMetricTFModel, len(req.ConfigValue.Elements()))
	diags := req.ConfigValue.ElementsAs(ctx, &elems, true)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	for _, elem := range elems {
		name := elem.Name.ValueString()
		if _, ok := seen[name]; ok {
			resp.Diagnostics.AddError("Duplicate metric name for service or custom namespace", fmt.Sprintf("Metric name %q is duplicated within the service or custom namespace.", name))
		}
		seen[name] = struct{}{}
	}
}

type awsCloudWatchScrapeJobNoDuplicateEnhancedMetricNamesValidator struct{}

func (v awsCloudWatchScrapeJobNoDuplicateEnhancedMetricNamesValidator) Description(ctx context.Context) string {
	return "Enhanced metric names must be unique (case-insensitive) within the same service."
}

func (v awsCloudWatchScrapeJobNoDuplicateEnhancedMetricNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "Enhanced metric names must be unique (case-insensitive) within the same service."
}

func (v awsCloudWatchScrapeJobNoDuplicateEnhancedMetricNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	seen := map[string]struct{}{}
	elems := make([]awsCloudWatchScrapeJobEnhancedMetricTFModel, len(req.ConfigValue.Elements()))
	diags := req.ConfigValue.ElementsAs(ctx, &elems, true)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	for _, elem := range elems {
		name := elem.Name.ValueString()
		if _, ok := seen[name]; ok {
			resp.Diagnostics.AddError("Duplicate enhanced metric name for service", fmt.Sprintf("Enhanced metric name %q is duplicated within the service.", name))
		}
		seen[name] = struct{}{}
	}
}

// toClientModel converts a awsCloudWatchScrapeJobTFModel instance to a cloudproviderapi.AWSCloudWatchScrapeJobRequest instance.
func (tfData awsCloudWatchScrapeJobTFResourceModel) toClientModel(ctx context.Context) (cloudproviderapi.AWSCloudWatchScrapeJobRequest, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := cloudproviderapi.AWSCloudWatchScrapeJobRequest{
		Name:                 tfData.Name.ValueString(),
		Enabled:              tfData.Enabled.ValueBool(),
		AWSAccountResourceID: tfData.AWSAccountResourceID.ValueString(),
		ExportTags:           tfData.ExportTags.ValueBool(),
	}

	diags := tfData.RegionsSubsetOverride.ElementsAs(ctx, &converted.RegionsSubsetOverride, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
	}

	var services []awsCloudWatchScrapeJobServiceTFModel
	diags = tfData.Services.ElementsAs(ctx, &services, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
	}
	converted.Services = make([]cloudproviderapi.AWSCloudWatchService, len(services))
	for i, service := range services {
		converted.Services[i] = cloudproviderapi.AWSCloudWatchService{
			Name:                  service.Name.ValueString(),
			ScrapeIntervalSeconds: service.ScrapeIntervalSeconds.ValueInt64(),
		}

		diags = service.TagsToAddToMetrics.ElementsAs(ctx, &converted.Services[i].TagsToAddToMetrics, false)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
		}

		var metrics []awsCloudWatchScrapeJobMetricTFModel
		diags = service.Metrics.ElementsAs(ctx, &metrics, false)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
		}
		converted.Services[i].Metrics = make([]cloudproviderapi.AWSCloudWatchMetric, len(metrics))
		for j, metric := range metrics {
			converted.Services[i].Metrics[j] = cloudproviderapi.AWSCloudWatchMetric{
				Name: metric.Name.ValueString(),
			}

			diags = metric.Statistics.ElementsAs(ctx, &converted.Services[i].Metrics[j].Statistics, false)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
			}
		}

		var enhancedMetrics []awsCloudWatchScrapeJobEnhancedMetricTFModel
		diags = service.EnhancedMetrics.ElementsAs(ctx, &enhancedMetrics, false)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
		}
		converted.Services[i].EnhancedMetrics = make([]cloudproviderapi.AWSEnhancedMetric, len(enhancedMetrics))
		for j, metric := range enhancedMetrics {
			converted.Services[i].EnhancedMetrics[j] = cloudproviderapi.AWSEnhancedMetric{
				Name: metric.Name.ValueString(),
			}
		}

		var tagFilters []awsCloudWatchScrapeJobTagFilterTFModel
		diags = service.ResourceDiscoveryTagFilters.ElementsAs(ctx, &tagFilters, false)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
		}
		converted.Services[i].ResourceDiscoveryTagFilters = make([]cloudproviderapi.AWSCloudWatchTagFilter, len(tagFilters))
		for j, tagFilter := range tagFilters {
			converted.Services[i].ResourceDiscoveryTagFilters[j] = cloudproviderapi.AWSCloudWatchTagFilter{
				Key:   tagFilter.Key.ValueString(),
				Value: tagFilter.Value.ValueString(),
			}
		}
	}

	var customNamepsaces []awsCloudWatchScrapeJobCustomNamespaceTFModel
	diags = tfData.CustomNamespaces.ElementsAs(ctx, &customNamepsaces, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
	}
	converted.CustomNamespaces = make([]cloudproviderapi.AWSCloudWatchCustomNamespace, len(customNamepsaces))
	for i, customNamespace := range customNamepsaces {
		converted.CustomNamespaces[i] = cloudproviderapi.AWSCloudWatchCustomNamespace{
			Name:                  customNamespace.Name.ValueString(),
			ScrapeIntervalSeconds: customNamespace.ScrapeIntervalSeconds.ValueInt64(),
		}

		var metrics []awsCloudWatchScrapeJobMetricTFModel
		diags = customNamespace.Metrics.ElementsAs(ctx, &metrics, false)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
		}
		converted.CustomNamespaces[i].Metrics = make([]cloudproviderapi.AWSCloudWatchMetric, len(metrics))
		for j, metric := range metrics {
			converted.CustomNamespaces[i].Metrics[j] = cloudproviderapi.AWSCloudWatchMetric{
				Name: metric.Name.ValueString(),
			}

			diags = metric.Statistics.ElementsAs(ctx, &converted.CustomNamespaces[i].Metrics[j].Statistics, false)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
			}
		}
	}

	diags = tfData.StaticLabels.ElementsAs(ctx, &converted.StaticLabels, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
	}

	return converted, conversionDiags
}

// generateCloudWatchScrapeJobTFResourceModel generates a new awsCloudWatchScrapeJobTFResourceModel based on the provided cloudproviderapi.AWSCloudWatchScrapeJobResponse
func generateCloudWatchScrapeJobTFResourceModel(ctx context.Context, stackID string, scrapeJobData cloudproviderapi.AWSCloudWatchScrapeJobResponse) (awsCloudWatchScrapeJobTFResourceModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := awsCloudWatchScrapeJobTFResourceModel{
		ID:                   types.StringValue(resourceAWSCloudWatchScrapeJobTerraformID.Make(stackID, scrapeJobData.Name)),
		StackID:              types.StringValue(stackID),
		Name:                 types.StringValue(scrapeJobData.Name),
		Enabled:              types.BoolValue(scrapeJobData.Enabled),
		AWSAccountResourceID: types.StringValue(scrapeJobData.AWSAccountResourceID),
		ExportTags:           types.BoolValue(scrapeJobData.ExportTags),
		DisabledReason:       types.StringValue(scrapeJobData.DisabledReason),
	}

	regionsSubsetOverride := types.SetValueMust(basetypes.StringType{}, []attr.Value{})
	if scrapeJobData.RegionsSubsetOverrideUsed {
		regions, diags := types.SetValueFrom(ctx, basetypes.StringType{}, scrapeJobData.Regions)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return awsCloudWatchScrapeJobTFResourceModel{}, conversionDiags
		}
		regionsSubsetOverride = regions
	}
	converted.RegionsSubsetOverride = regionsSubsetOverride

	services, diags := convertAWSCloudWatchServicesClientToTFModel(ctx, scrapeJobData.Services)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCloudWatchScrapeJobTFResourceModel{}, conversionDiags
	}
	converted.Services = services

	customNamespaces, diags := convertAWSCloudWatchCustomNamespacesClientToTFModel(ctx, scrapeJobData.CustomNamespaces)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCloudWatchScrapeJobTFResourceModel{}, conversionDiags
	}
	converted.CustomNamespaces = customNamespaces

	staticLabelsMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	if scrapeJobData.StaticLabels != nil {
		staticLabelsMap, diags = types.MapValueFrom(ctx, basetypes.StringType{}, scrapeJobData.StaticLabels)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return awsCloudWatchScrapeJobTFResourceModel{}, conversionDiags
		}
	}
	converted.StaticLabels = staticLabelsMap

	return converted, conversionDiags
}

// generateCloudWatchScrapeJobTFDataSourceModel generates a new awsCloudWatchScrapeJobTFDataSourceModel based on the provided cloudproviderapi.AWSCloudWatchScrapeJobResponse
func generateAWSCloudWatchScrapeJobDataSourceTFModel(ctx context.Context, stackID string, scrapeJobData cloudproviderapi.AWSCloudWatchScrapeJobResponse) (awsCloudWatchScrapeJobTFDataSourceModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := awsCloudWatchScrapeJobTFDataSourceModel{
		ID:                        types.StringValue(resourceAWSCloudWatchScrapeJobTerraformID.Make(stackID, scrapeJobData.Name)),
		StackID:                   types.StringValue(stackID),
		Name:                      types.StringValue(scrapeJobData.Name),
		Enabled:                   types.BoolValue(scrapeJobData.Enabled),
		AWSAccountResourceID:      types.StringValue(scrapeJobData.AWSAccountResourceID),
		RoleARN:                   types.StringValue(scrapeJobData.RoleARN),
		RegionsSubsetOverrideUsed: types.BoolValue(scrapeJobData.RegionsSubsetOverrideUsed),
		ExportTags:                types.BoolValue(scrapeJobData.ExportTags),
		DisabledReason:            types.StringValue(scrapeJobData.DisabledReason),
	}

	regions, diags := types.SetValueFrom(ctx, basetypes.StringType{}, scrapeJobData.Regions)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCloudWatchScrapeJobTFDataSourceModel{}, conversionDiags
	}
	converted.Regions = regions

	services, diags := convertAWSCloudWatchServicesClientToTFModel(ctx, scrapeJobData.Services)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCloudWatchScrapeJobTFDataSourceModel{}, conversionDiags
	}
	converted.Services = services

	customNamespaces, diags := convertAWSCloudWatchCustomNamespacesClientToTFModel(ctx, scrapeJobData.CustomNamespaces)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCloudWatchScrapeJobTFDataSourceModel{}, conversionDiags
	}
	converted.CustomNamespaces = customNamespaces

	staticLabelsMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	if scrapeJobData.StaticLabels != nil {
		staticLabelsMap, diags = types.MapValueFrom(ctx, basetypes.StringType{}, scrapeJobData.StaticLabels)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return awsCloudWatchScrapeJobTFDataSourceModel{}, conversionDiags
		}
	}
	converted.StaticLabels = staticLabelsMap

	return converted, conversionDiags
}

func convertAWSCloudWatchServicesClientToTFModel(ctx context.Context, services []cloudproviderapi.AWSCloudWatchService) (types.List, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	servicesTF := make([]awsCloudWatchScrapeJobServiceTFModel, len(services))
	servicesListObjType := types.ObjectType{AttrTypes: awsCloudWatchScrapeJobServiceTFModel{}.attrTypes()}

	for i, service := range services {
		serviceTF := awsCloudWatchScrapeJobServiceTFModel{
			Name:                  types.StringValue(service.Name),
			ScrapeIntervalSeconds: types.Int64Value(service.ScrapeIntervalSeconds),
		}

		metricsTF := make([]awsCloudWatchScrapeJobMetricTFModel, len(service.Metrics))
		for j, metric := range service.Metrics {
			metricsTF[j] = awsCloudWatchScrapeJobMetricTFModel{
				Name: types.StringValue(metric.Name),
			}
			statistics, diags := types.SetValueFrom(ctx, basetypes.StringType{}, services[i].Metrics[j].Statistics)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return types.ListNull(servicesListObjType), conversionDiags
			}
			metricsTF[j].Statistics = statistics
		}
		metricsTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCloudWatchScrapeJobMetricTFModel{}.attrTypes()}, metricsTF)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(servicesListObjType), conversionDiags
		}
		serviceTF.Metrics = metricsTFList

		enhancedMetricsTF := make([]awsCloudWatchScrapeJobEnhancedMetricTFModel, len(service.EnhancedMetrics))
		for j, metric := range service.EnhancedMetrics {
			enhancedMetricsTF[j] = awsCloudWatchScrapeJobEnhancedMetricTFModel{
				Name: types.StringValue(metric.Name),
			}
		}
		enhancedMetricsTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCloudWatchScrapeJobEnhancedMetricTFModel{}.attrTypes()}, enhancedMetricsTF)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(servicesListObjType), conversionDiags
		}
		serviceTF.EnhancedMetrics = enhancedMetricsTFList

		tagFiltersTF := make([]awsCloudWatchScrapeJobTagFilterTFModel, len(service.ResourceDiscoveryTagFilters))
		for j, tagFilter := range service.ResourceDiscoveryTagFilters {
			tagFiltersTF[j] = awsCloudWatchScrapeJobTagFilterTFModel{
				Key:   types.StringValue(tagFilter.Key),
				Value: types.StringValue(tagFilter.Value),
			}
		}
		tagFiltersTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCloudWatchScrapeJobTagFilterTFModel{}.attrTypes()}, tagFiltersTF)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(servicesListObjType), conversionDiags
		}
		serviceTF.ResourceDiscoveryTagFilters = tagFiltersTFList

		tagsToAdd, diags := types.SetValueFrom(ctx, basetypes.StringType{}, services[i].TagsToAddToMetrics)
		if tagsToAdd.IsNull() {
			tagsToAdd = types.SetValueMust(basetypes.StringType{}, []attr.Value{})
		}
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(servicesListObjType), conversionDiags
		}
		serviceTF.TagsToAddToMetrics = tagsToAdd

		servicesTF[i] = serviceTF
	}

	servicesTFList, diags := types.ListValueFrom(ctx, servicesListObjType, servicesTF)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return types.ListNull(servicesListObjType), conversionDiags
	}

	return servicesTFList, conversionDiags
}

func convertAWSCloudWatchCustomNamespacesClientToTFModel(ctx context.Context, customNamespaces []cloudproviderapi.AWSCloudWatchCustomNamespace) (types.List, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	customNamespacesTF := make([]awsCloudWatchScrapeJobCustomNamespaceTFModel, len(customNamespaces))
	customNamspacesListObjType := types.ObjectType{AttrTypes: awsCloudWatchScrapeJobCustomNamespaceTFModel{}.attrTypes()}

	for i, customNamespace := range customNamespaces {
		customNamespaceTF := awsCloudWatchScrapeJobCustomNamespaceTFModel{
			Name:                  types.StringValue(customNamespace.Name),
			ScrapeIntervalSeconds: types.Int64Value(customNamespace.ScrapeIntervalSeconds),
		}

		metricsTF := make([]awsCloudWatchScrapeJobMetricTFModel, len(customNamespace.Metrics))
		for j, metric := range customNamespace.Metrics {
			metricsTF[j] = awsCloudWatchScrapeJobMetricTFModel{
				Name: types.StringValue(metric.Name),
			}
			statistics, diags := types.SetValueFrom(ctx, basetypes.StringType{}, customNamespaces[i].Metrics[j].Statistics)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return types.ListNull(customNamspacesListObjType), conversionDiags
			}
			metricsTF[j].Statistics = statistics
		}
		metricsTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCloudWatchScrapeJobMetricTFModel{}.attrTypes()}, metricsTF)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(customNamspacesListObjType), conversionDiags
		}
		customNamespaceTF.Metrics = metricsTFList

		customNamespacesTF[i] = customNamespaceTF
	}

	customNamespacesTFList, diags := types.ListValueFrom(ctx, customNamspacesListObjType, customNamespacesTF)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return types.ListNull(customNamspacesListObjType), conversionDiags
	}

	return customNamespacesTFList, conversionDiags
}

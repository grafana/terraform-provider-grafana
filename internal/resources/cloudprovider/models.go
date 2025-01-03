package cloudprovider

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type awsCWScrapeJobTFResourceModel struct {
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
	StaticLabels     types.List `tfsdk:"static_label"`
}

type awsCWScrapeJobTFDataSourceModel struct {
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
	StaticLabels     types.List `tfsdk:"static_label"`
}

type awsCWScrapeJobStaticLabelTFModel struct {
	Label types.String `tfsdk:"label"`
	Value types.String `tfsdk:"value"`
}

func (tf awsCWScrapeJobStaticLabelTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"label": types.StringType,
		"value": types.StringType,
	}
}

type awsCWScrapeJobServiceTFModel struct {
	Name                        types.String `tfsdk:"name"`
	Metrics                     types.List   `tfsdk:"metric"`
	ScrapeIntervalSeconds       types.Int64  `tfsdk:"scrape_interval_seconds"`
	ResourceDiscoveryTagFilters types.List   `tfsdk:"resource_discovery_tag_filter"`
	TagsToAddToMetrics          types.Set    `tfsdk:"tags_to_add_to_metrics"`
}

func (m awsCWScrapeJobServiceTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"metric": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsCWScrapeJobMetricTFModel{}.attrTypes(),
			},
		},
		"scrape_interval_seconds": types.Int64Type,
		"resource_discovery_tag_filter": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsCWScrapeJobTagFilterTFModel{}.attrTypes(),
			},
		},
		"tags_to_add_to_metrics": types.SetType{
			ElemType: types.StringType,
		},
	}
}

type awsCWScrapeJobCustomNamespaceTFModel struct {
	Name                  types.String `tfsdk:"name"`
	Metrics               types.List   `tfsdk:"metric"`
	ScrapeIntervalSeconds types.Int64  `tfsdk:"scrape_interval_seconds"`
}

func (m awsCWScrapeJobCustomNamespaceTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"metric": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsCWScrapeJobMetricTFModel{}.attrTypes(),
			},
		},
		"scrape_interval_seconds": types.Int64Type,
	}
}

type awsCWScrapeJobMetricTFModel struct {
	Name       types.String `tfsdk:"name"`
	Statistics types.Set    `tfsdk:"statistics"`
}

func (m awsCWScrapeJobMetricTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"statistics": types.SetType{
			ElemType: types.StringType,
		},
	}
}

type awsCWScrapeJobTagFilterTFModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func (m awsCWScrapeJobTagFilterTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
	}
}

type awsCWScrapeJobNoDuplicateServiceNamesValidator struct{}

func (v awsCWScrapeJobNoDuplicateServiceNamesValidator) Description(ctx context.Context) string {
	return "No duplicate service names are allowed."
}

func (v awsCWScrapeJobNoDuplicateServiceNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "No duplicate service names are allowed."
}

func (v awsCWScrapeJobNoDuplicateServiceNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	var services []awsCWScrapeJobServiceTFModel
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

type awsCWScrapeJobNoDuplicateCustomNamespaceNamesValidator struct{}

func (v awsCWScrapeJobNoDuplicateCustomNamespaceNamesValidator) Description(ctx context.Context) string {
	return "No duplicate custom namespace names are allowed."
}

func (v awsCWScrapeJobNoDuplicateCustomNamespaceNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "No duplicate custom namespace names are allowed."
}

func (v awsCWScrapeJobNoDuplicateCustomNamespaceNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	var customNamespaces []awsCWScrapeJobCustomNamespaceTFModel
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

type awsCWScrapeJobNoDuplicateMetricNamesValidator struct{}

func (v awsCWScrapeJobNoDuplicateMetricNamesValidator) Description(ctx context.Context) string {
	return "Metric names must be unique (case-insensitive) within the same service or custom namespace."
}

func (v awsCWScrapeJobNoDuplicateMetricNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "Metric names must be unique (case-insensitive) within the same service or custom namespace."
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
			resp.Diagnostics.AddError("Duplicate metric name for service or custom namespace", fmt.Sprintf("Metric name %q is duplicated within the service or custom namespace.", name))
		}
		seen[name] = struct{}{}
	}
}

// toClientModel converts a awsCWScrapeJobTFModel instance to a cloudproviderapi.AWSCloudWatchScrapeJobRequest instance.
func (tfData awsCWScrapeJobTFResourceModel) toClientModel(ctx context.Context) (cloudproviderapi.AWSCloudWatchScrapeJobRequest, diag.Diagnostics) {
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

	var services []awsCWScrapeJobServiceTFModel
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

		var metrics []awsCWScrapeJobMetricTFModel
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

		var tagFilters []awsCWScrapeJobTagFilterTFModel
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

	var customNamepsaces []awsCWScrapeJobCustomNamespaceTFModel
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

		var metrics []awsCWScrapeJobMetricTFModel
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

	var staticLabels []awsCWScrapeJobStaticLabelTFModel
	diags = tfData.StaticLabels.ElementsAs(ctx, &staticLabels, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSCloudWatchScrapeJobRequest{}, conversionDiags
	}
	converted.StaticLabels = make(map[string]string, len(services))
	for _, label := range staticLabels {
		converted.StaticLabels[label.Label.ValueString()] = label.Value.ValueString()
	}

	return converted, conversionDiags
}

// generateCloudWatchScrapeJobTFResourceModel generates a new awsCWScrapeJobTFResourceModel based on the provided cloudproviderapi.AWSCloudWatchScrapeJobResponse
func generateCloudWatchScrapeJobTFResourceModel(ctx context.Context, stackID string, scrapeJobData cloudproviderapi.AWSCloudWatchScrapeJobResponse) (awsCWScrapeJobTFResourceModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := awsCWScrapeJobTFResourceModel{
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
			return awsCWScrapeJobTFResourceModel{}, conversionDiags
		}
		regionsSubsetOverride = regions
	}
	converted.RegionsSubsetOverride = regionsSubsetOverride

	services, diags := convertServicesClientToTFModel(ctx, scrapeJobData.Services)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCWScrapeJobTFResourceModel{}, conversionDiags
	}
	converted.Services = services

	customNamespaces, diags := convertCustomNamespacesClientToTFModel(ctx, scrapeJobData.CustomNamespaces)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCWScrapeJobTFResourceModel{}, conversionDiags
	}
	converted.CustomNamespaces = customNamespaces

	staticLabels := make([]awsCWScrapeJobStaticLabelTFModel, 0, len(scrapeJobData.StaticLabels))
	for key, value := range scrapeJobData.StaticLabels {
		staticLabels = append(staticLabels, awsCWScrapeJobStaticLabelTFModel{
			Label: types.StringValue(key),
			Value: types.StringValue(value),
		})
	}

	staticLabelsList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCWScrapeJobStaticLabelTFModel{}.attrTypes()}, staticLabels)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCWScrapeJobTFResourceModel{}, conversionDiags
	}
	converted.StaticLabels = staticLabelsList

	return converted, conversionDiags
}

// generateCloudWatchScrapeJobTFDataSourceModel generates a new awsCWScrapeJobTFDataSourceModel based on the provided cloudproviderapi.AWSCloudWatchScrapeJobResponse
func generateCloudWatchScrapeJobDataSourceTFModel(ctx context.Context, stackID string, scrapeJobData cloudproviderapi.AWSCloudWatchScrapeJobResponse) (awsCWScrapeJobTFDataSourceModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := awsCWScrapeJobTFDataSourceModel{
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
		return awsCWScrapeJobTFDataSourceModel{}, conversionDiags
	}
	converted.Regions = regions

	services, diags := convertServicesClientToTFModel(ctx, scrapeJobData.Services)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCWScrapeJobTFDataSourceModel{}, conversionDiags
	}
	converted.Services = services

	customNamespaces, diags := convertCustomNamespacesClientToTFModel(ctx, scrapeJobData.CustomNamespaces)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCWScrapeJobTFDataSourceModel{}, conversionDiags
	}
	converted.CustomNamespaces = customNamespaces

	staticLabels := make([]awsCWScrapeJobStaticLabelTFModel, 0, len(scrapeJobData.StaticLabels))
	for key, value := range scrapeJobData.StaticLabels {
		staticLabels = append(staticLabels, awsCWScrapeJobStaticLabelTFModel{
			Label: types.StringValue(key),
			Value: types.StringValue(value),
		})
	}

	staticLabelsList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCWScrapeJobStaticLabelTFModel{}.attrTypes()}, staticLabels)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsCWScrapeJobTFDataSourceModel{}, conversionDiags
	}
	converted.StaticLabels = staticLabelsList

	return converted, conversionDiags
}

func convertServicesClientToTFModel(ctx context.Context, services []cloudproviderapi.AWSCloudWatchService) (types.List, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	servicesTF := make([]awsCWScrapeJobServiceTFModel, len(services))
	servicesListObjType := types.ObjectType{AttrTypes: awsCWScrapeJobServiceTFModel{}.attrTypes()}

	for i, service := range services {
		serviceTF := awsCWScrapeJobServiceTFModel{
			Name:                  types.StringValue(service.Name),
			ScrapeIntervalSeconds: types.Int64Value(service.ScrapeIntervalSeconds),
		}

		metricsTF := make([]awsCWScrapeJobMetricTFModel, len(service.Metrics))
		for j, metric := range service.Metrics {
			metricsTF[j] = awsCWScrapeJobMetricTFModel{
				Name: types.StringValue(metric.Name),
			}
			statistics, diags := types.SetValueFrom(ctx, basetypes.StringType{}, services[i].Metrics[j].Statistics)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return types.ListNull(servicesListObjType), conversionDiags
			}
			metricsTF[j].Statistics = statistics
		}
		metricsTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCWScrapeJobMetricTFModel{}.attrTypes()}, metricsTF)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(servicesListObjType), conversionDiags
		}
		serviceTF.Metrics = metricsTFList

		tagFiltersTF := make([]awsCWScrapeJobTagFilterTFModel, len(service.ResourceDiscoveryTagFilters))
		for j, tagFilter := range service.ResourceDiscoveryTagFilters {
			tagFiltersTF[j] = awsCWScrapeJobTagFilterTFModel{
				Key:   types.StringValue(tagFilter.Key),
				Value: types.StringValue(tagFilter.Value),
			}
		}
		tagFiltersTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCWScrapeJobTagFilterTFModel{}.attrTypes()}, tagFiltersTF)
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

func convertCustomNamespacesClientToTFModel(ctx context.Context, customNamespaces []cloudproviderapi.AWSCloudWatchCustomNamespace) (types.List, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	customNamespacesTF := make([]awsCWScrapeJobCustomNamespaceTFModel, len(customNamespaces))
	customNamspacesListObjType := types.ObjectType{AttrTypes: awsCWScrapeJobCustomNamespaceTFModel{}.attrTypes()}

	for i, customNamespace := range customNamespaces {
		customNamespaceTF := awsCWScrapeJobCustomNamespaceTFModel{
			Name:                  types.StringValue(customNamespace.Name),
			ScrapeIntervalSeconds: types.Int64Value(customNamespace.ScrapeIntervalSeconds),
		}

		metricsTF := make([]awsCWScrapeJobMetricTFModel, len(customNamespace.Metrics))
		for j, metric := range customNamespace.Metrics {
			metricsTF[j] = awsCWScrapeJobMetricTFModel{
				Name: types.StringValue(metric.Name),
			}
			statistics, diags := types.SetValueFrom(ctx, basetypes.StringType{}, customNamespaces[i].Metrics[j].Statistics)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return types.ListNull(customNamspacesListObjType), conversionDiags
			}
			metricsTF[j].Statistics = statistics
		}
		metricsTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsCWScrapeJobMetricTFModel{}.attrTypes()}, metricsTF)
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

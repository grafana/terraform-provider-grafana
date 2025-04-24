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

type awsRMScrapeJobTFResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	StackID               types.String `tfsdk:"stack_id"`
	Name                  types.String `tfsdk:"name"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	AWSAccountResourceID  types.String `tfsdk:"aws_account_resource_id"`
	RegionsSubsetOverride types.Set    `tfsdk:"regions_subset_override"`
	DisabledReason        types.String `tfsdk:"disabled_reason"`
	// TODO(tristan): if the grafana provider is updated to use the Terraform v6 plugin protocol,
	// we can consider adding additional support to use Set Nested Attributes, instead of Blocks.
	// See https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attribute-types
	Services     types.List `tfsdk:"service"`
	StaticLabels types.Map  `tfsdk:"static_labels"`
}

type awsRMScrapeJobServiceTFModel struct {
	Name                        types.String `tfsdk:"name"`
	ScrapeIntervalSeconds       types.Int64  `tfsdk:"scrape_interval_seconds"`
	ResourceDiscoveryTagFilters types.List   `tfsdk:"resource_discovery_tag_filter"`
}

func (m awsRMScrapeJobServiceTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":                    types.StringType,
		"scrape_interval_seconds": types.Int64Type,
		"resource_discovery_tag_filter": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: awsRMScrapeJobTagFilterTFModel{}.attrTypes(),
			},
		},
	}
}

type awsRMScrapeJobTagFilterTFModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func (m awsRMScrapeJobTagFilterTFModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
	}
}

type awsRMScrapeJobNoDuplicateServiceNamesValidator struct{}

func (v awsRMScrapeJobNoDuplicateServiceNamesValidator) Description(ctx context.Context) string {
	return "No duplicate service names are allowed."
}

func (v awsRMScrapeJobNoDuplicateServiceNamesValidator) MarkdownDescription(ctx context.Context) string {
	return "No duplicate service names are allowed."
}

func (v awsRMScrapeJobNoDuplicateServiceNamesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	var services []awsRMScrapeJobServiceTFModel
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

// toClientModel converts a awsRMScrapeJobTFModel instance to a cloudproviderapi.AWSResourceMetadataScrapeJobRequest instance.
func (tfData awsRMScrapeJobTFResourceModel) toClientModel(ctx context.Context) (cloudproviderapi.AWSResourceMetadataScrapeJobRequest, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := cloudproviderapi.AWSResourceMetadataScrapeJobRequest{
		Name:                 tfData.Name.ValueString(),
		Enabled:              tfData.Enabled.ValueBool(),
		AWSAccountResourceID: tfData.AWSAccountResourceID.ValueString(),
	}

	diags := tfData.RegionsSubsetOverride.ElementsAs(ctx, &converted.RegionsSubsetOverride, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSResourceMetadataScrapeJobRequest{}, conversionDiags
	}

	var services []awsRMScrapeJobServiceTFModel
	diags = tfData.Services.ElementsAs(ctx, &services, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSResourceMetadataScrapeJobRequest{}, conversionDiags
	}
	converted.Services = make([]cloudproviderapi.AWSResourceMetadataService, len(services))
	for i, service := range services {
		converted.Services[i] = cloudproviderapi.AWSResourceMetadataService{
			Name:                  service.Name.ValueString(),
			ScrapeIntervalSeconds: service.ScrapeIntervalSeconds.ValueInt64(),
		}

		var tagFilters []awsRMScrapeJobTagFilterTFModel
		diags = service.ResourceDiscoveryTagFilters.ElementsAs(ctx, &tagFilters, false)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return cloudproviderapi.AWSResourceMetadataScrapeJobRequest{}, conversionDiags
		}
		converted.Services[i].ResourceDiscoveryTagFilters = make([]cloudproviderapi.AWSResourceMetadataTagFilter, len(tagFilters))
		for j, tagFilter := range tagFilters {
			converted.Services[i].ResourceDiscoveryTagFilters[j] = cloudproviderapi.AWSResourceMetadataTagFilter{
				Key:   tagFilter.Key.ValueString(),
				Value: tagFilter.Value.ValueString(),
			}
		}
	}

	diags = tfData.StaticLabels.ElementsAs(ctx, &converted.StaticLabels, false)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return cloudproviderapi.AWSResourceMetadataScrapeJobRequest{}, conversionDiags
	}

	return converted, conversionDiags
}

// generateResourceMetadataScrapeJobTFResourceModel generates a new awsRMScrapeJobTFResourceModel based on the provided cloudproviderapi.AWSResourceMetadataScrapeJobResponse
func generateResourceMetadataScrapeJobTFResourceModel(ctx context.Context, stackID string, scrapeJobData cloudproviderapi.AWSResourceMetadataScrapeJobResponse) (awsRMScrapeJobTFResourceModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := awsRMScrapeJobTFResourceModel{
		ID:                   types.StringValue(resourceAWSResourceMetadataScrapeJobTerraformID.Make(stackID, scrapeJobData.Name)),
		StackID:              types.StringValue(stackID),
		Name:                 types.StringValue(scrapeJobData.Name),
		Enabled:              types.BoolValue(scrapeJobData.Enabled),
		AWSAccountResourceID: types.StringValue(scrapeJobData.AWSAccountResourceID),
		DisabledReason:       types.StringValue(scrapeJobData.DisabledReason),
	}

	regionsSubsetOverride := types.SetValueMust(basetypes.StringType{}, []attr.Value{})
	if scrapeJobData.RegionsSubsetOverrideUsed {
		regions, diags := types.SetValueFrom(ctx, basetypes.StringType{}, scrapeJobData.Regions)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return awsRMScrapeJobTFResourceModel{}, conversionDiags
		}
		regionsSubsetOverride = regions
	}
	converted.RegionsSubsetOverride = regionsSubsetOverride

	services, diags := convertRMServicesClientToTFModel(ctx, scrapeJobData.Services)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return awsRMScrapeJobTFResourceModel{}, conversionDiags
	}
	converted.Services = services

	staticLabelsMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	if scrapeJobData.StaticLabels != nil {
		staticLabelsMap, diags = types.MapValueFrom(ctx, basetypes.StringType{}, scrapeJobData.StaticLabels)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return awsRMScrapeJobTFResourceModel{}, conversionDiags
		}
	}
	converted.StaticLabels = staticLabelsMap

	return converted, conversionDiags
}

func convertRMServicesClientToTFModel(ctx context.Context, services []cloudproviderapi.AWSResourceMetadataService) (types.List, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	servicesTF := make([]awsRMScrapeJobServiceTFModel, len(services))
	servicesListObjType := types.ObjectType{AttrTypes: awsRMScrapeJobServiceTFModel{}.attrTypes()}

	for i, service := range services {
		serviceTF := awsRMScrapeJobServiceTFModel{
			Name:                  types.StringValue(service.Name),
			ScrapeIntervalSeconds: types.Int64Value(service.ScrapeIntervalSeconds),
		}

		tagFiltersTF := make([]awsRMScrapeJobTagFilterTFModel, len(service.ResourceDiscoveryTagFilters))
		for j, tagFilter := range service.ResourceDiscoveryTagFilters {
			tagFiltersTF[j] = awsRMScrapeJobTagFilterTFModel{
				Key:   types.StringValue(tagFilter.Key),
				Value: types.StringValue(tagFilter.Value),
			}
		}
		tagFiltersTFList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: awsRMScrapeJobTagFilterTFModel{}.attrTypes()}, tagFiltersTF)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(servicesListObjType), conversionDiags
		}
		serviceTF.ResourceDiscoveryTagFilters = tagFiltersTFList

		servicesTF[i] = serviceTF
	}

	servicesTFList, diags := types.ListValueFrom(ctx, servicesListObjType, servicesTF)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return types.ListNull(servicesListObjType), conversionDiags
	}

	return servicesTFList, conversionDiags
}

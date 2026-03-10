package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudproviderapi"
)

var (
	//nolint:gosec
	resourceAzureCredentialTerraformName = "grafana_cloud_provider_azure_credential"
	resourceAzureCredentialTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("resource_id"))
)

type resourceAzureCredentialModel struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	TenantID                   types.String `tfsdk:"tenant_id"`
	ClientID                   types.String `tfsdk:"client_id"`
	StackID                    types.String `tfsdk:"stack_id"`
	ClientSecret               types.String `tfsdk:"client_secret"`
	ResourceID                 types.String `tfsdk:"resource_id"`
	ResourceTagFilters         types.List   `tfsdk:"resource_discovery_tag_filter"`
	AutoDiscoveryConfiguration types.List   `tfsdk:"auto_discovery_configuration"`
	ResourceTagsToAddToMetrics types.Set    `tfsdk:"resource_tags_to_add_to_metrics"`
}

type TagFilter struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func (tf TagFilter) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
	}
}

type azureAutoDiscoveryConfigurationModel struct {
	SubscriptionID             types.String `tfsdk:"subscription_id"`
	ResourceTypeConfigurations types.List   `tfsdk:"resource_type_configurations"`
}

func (tf azureAutoDiscoveryConfigurationModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"subscription_id": types.StringType,
		"resource_type_configurations": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: azureResourceTypeConfigurationModel{}.attrTypes(),
			},
		},
	}
}

type azureResourceTypeConfigurationModel struct {
	ResourceTypeName    types.String `tfsdk:"resource_type_name"`
	MetricConfiguration types.List   `tfsdk:"metric_configuration"`
}

func (tf azureResourceTypeConfigurationModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"resource_type_name": types.StringType,
		"metric_configuration": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: azureMetricConfigurationModel{}.attrTypes(),
			},
		},
	}
}

type azureMetricConfigurationModel struct {
	Name         types.String `tfsdk:"name"`
	Dimensions   types.List   `tfsdk:"dimensions"`
	Aggregations types.List   `tfsdk:"aggregations"`
}

func (tf azureMetricConfigurationModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":         types.StringType,
		"dimensions":   types.ListType{ElemType: types.StringType},
		"aggregations": types.ListType{ElemType: types.StringType},
	}
}

type resourceAzureCredential struct {
	client *cloudproviderapi.Client
}

func makeResourceAzureCredential() *common.Resource {
	return common.NewResource(
		common.CategoryCloudProvider,
		resourceAzureCredentialTerraformName,
		resourceAzureCredentialTerraformID,
		&resourceAzureCredential{},
	)
}

func (r *resourceAzureCredential) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}

	r.client = client
}

func (r *resourceAzureCredential) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAzureCredentialTerraformName
}

func (r *resourceAzureCredential) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
This resource allows you to autodiscover resources in your Azure tenant and scrape Azure Monitor metrics for those resources in Grafana Cloud without needing to run your own infrastructure.

See the [Grafana Provider configuration docs](https://registry.terraform.io/providers/grafana/grafana/latest/docs#managing-cloud-provider)
for information on authentication and required access policy scopes.

* [Official Grafana Cloud documentation](https://grafana.com/docs/grafana-cloud/monitor-infrastructure/monitor-cloud-provider/azure/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ resource_id }}\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#usestateforunknown
					// for details on how this works.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"stack_id": schema.StringAttribute{
				Description: "The StackID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: "The ID given by the Grafana Cloud Provider API to this Azure Credential resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#usestateforunknown
					// for details on how this works.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Azure Credential.",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID of the Azure Credential.",
				Required:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID of the Azure Credential.",
				Required:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The client secret of the Azure Credential.",
				Required:    true,
				Sensitive:   true,
			},
			"resource_tags_to_add_to_metrics": schema.SetAttribute{
				Description: "The list of resource tags to add to metrics.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"resource_discovery_tag_filter": schema.ListNestedBlock{
				Description: "The list of tag filters to apply to resources.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The key of the tag filter.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the tag filter.",
							Required:    true,
						},
					},
				},
			},
			"auto_discovery_configuration": schema.ListNestedBlock{
				Description: "The list of auto discovery configurations.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"subscription_id": schema.StringAttribute{
							Description: "The subscription ID of the Azure account.",
							Required:    true,
						},
						"resource_type_configurations": schema.ListAttribute{
							Description: "The list of resource type configurations.",
							Required:    true,
							ElementType: types.ObjectType{AttrTypes: azureResourceTypeConfigurationModel{}.attrTypes()},
						},
					},
				},
			},
		},
	}
}

func (r *resourceAzureCredential) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Invalid ID: %s", req.ID))
		return
	}
	stackID := parts[0]
	resourceID := parts[1]

	credentials, err := r.client.GetAzureCredential(
		ctx,
		stackID,
		resourceID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Azure credential", err.Error())
		return
	}

	tagFilters, diags := r.convertTagFilters(ctx, credentials.ResourceTagFilters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	autoconfiguration, diags := r.convertAutoDiscoveryConfigurations(ctx, credentials.AutoDiscoveryConfiguration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTagsToAddToMetrics, diags := types.SetValueFrom(ctx, basetypes.StringType{}, credentials.ResourceTagsToAddToMetrics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &resourceAzureCredentialModel{
		ID:                         types.StringValue(req.ID),
		Name:                       types.StringValue(credentials.Name),
		TenantID:                   types.StringValue(credentials.TenantID),
		ClientID:                   types.StringValue(credentials.ClientID),
		StackID:                    types.StringValue(stackID),
		ResourceID:                 types.StringValue(resourceID),
		ClientSecret:               types.StringValue(""), // We don't import the client secret
		ResourceTagFilters:         tagFilters,
		AutoDiscoveryConfiguration: autoconfiguration,
		ResourceTagsToAddToMetrics: resourceTagsToAddToMetrics,
	})
}

func (r *resourceAzureCredential) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAzureCredentialModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var requestTagFilters []cloudproviderapi.TagFilter

	var tagFilters []TagFilter
	diags = data.ResourceTagFilters.ElementsAs(ctx, &tagFilters, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, tagFilter := range tagFilters {
		requestTagFilters = append(requestTagFilters, cloudproviderapi.TagFilter{
			Key:   tagFilter.Key.ValueString(),
			Value: tagFilter.Value.ValueString(),
		})
	}

	var requestAutoDiscoveryConfiguration []cloudproviderapi.AutoDiscoveryConfiguration

	requestAutoDiscoveryConfiguration, diags = r.convertAutoDiscoveryConfigurationsToRequest(ctx, data.AutoDiscoveryConfiguration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	azureCredential := cloudproviderapi.AzureCredential{
		Name:                       data.Name.ValueString(),
		TenantID:                   data.TenantID.ValueString(),
		ClientID:                   data.ClientID.ValueString(),
		ClientSecret:               data.ClientSecret.ValueString(),
		ResourceTagFilters:         requestTagFilters,
		AutoDiscoveryConfiguration: requestAutoDiscoveryConfiguration,
	}

	diags = data.ResourceTagsToAddToMetrics.ElementsAs(ctx, &azureCredential.ResourceTagsToAddToMetrics, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credential, err := r.client.CreateAzureCredential(
		ctx,
		data.StackID.ValueString(),
		azureCredential,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Azure credential", err.Error())
		return
	}

	resp.State.Set(ctx, &resourceAzureCredentialModel{
		ID:                         types.StringValue(resourceAzureCredentialTerraformID.Make(data.StackID.ValueString(), credential.ID)),
		Name:                       data.Name,
		TenantID:                   data.TenantID,
		ClientID:                   data.ClientID,
		StackID:                    data.StackID,
		ClientSecret:               data.ClientSecret,
		ResourceID:                 types.StringValue(credential.ID),
		ResourceTagFilters:         data.ResourceTagFilters,
		AutoDiscoveryConfiguration: data.AutoDiscoveryConfiguration,
		ResourceTagsToAddToMetrics: data.ResourceTagsToAddToMetrics,
	})
}

func (r *resourceAzureCredential) convertTagFilters(ctx context.Context, apiTagFilters []cloudproviderapi.TagFilter) (types.List, diag.Diagnostics) {
	tagFiltersTF := make([]TagFilter, len(apiTagFilters))
	conversionDiags := diag.Diagnostics{}
	tagFilterListObjType := types.ObjectType{AttrTypes: TagFilter{}.attrTypes()}

	for i, apiTagFilter := range apiTagFilters {
		tagFiltersTF[i] = TagFilter{
			Key:   types.StringValue(apiTagFilter.Key),
			Value: types.StringValue(apiTagFilter.Value),
		}
	}

	tagFiltersTFList, diags := types.ListValueFrom(ctx, tagFilterListObjType, tagFiltersTF)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return types.ListNull(tagFilterListObjType), conversionDiags
	}
	return tagFiltersTFList, conversionDiags
}

func (r *resourceAzureCredential) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceAzureCredentialModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credential, err := r.client.GetAzureCredential(
		ctx,
		data.StackID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Azure credential", err.Error())
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("name"), credential.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("tenant_id"), credential.TenantID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_id"), credential.ClientID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagFilters, diags := r.convertTagFilters(ctx, credential.ResourceTagFilters)
	resp.Diagnostics.Append(diags...)
	diags = resp.State.SetAttribute(ctx, path.Root("resource_discovery_tag_filter"), tagFilters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	autoconfiguration, diags := r.convertAutoDiscoveryConfigurations(ctx, credential.AutoDiscoveryConfiguration)
	resp.Diagnostics.Append(diags...)
	diags = resp.State.SetAttribute(ctx, path.Root("auto_discovery_configuration"), autoconfiguration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("resource_tags_to_add_to_metrics"), credential.ResourceTagsToAddToMetrics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceAzureCredential) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData resourceAzureCredentialModel
	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credential := cloudproviderapi.AzureCredential{}
	credential.Name = planData.Name.ValueString()
	credential.TenantID = planData.TenantID.ValueString()
	credential.ClientID = planData.ClientID.ValueString()
	credential.ClientSecret = planData.ClientSecret.ValueString()

	var tagFilters []TagFilter
	diags = planData.ResourceTagFilters.ElementsAs(ctx, &tagFilters, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credential.ResourceTagFilters = make([]cloudproviderapi.TagFilter, len(tagFilters))
	for i, tagFilter := range tagFilters {
		credential.ResourceTagFilters[i] = cloudproviderapi.TagFilter{
			Key:   tagFilter.Key.ValueString(),
			Value: tagFilter.Value.ValueString(),
		}
	}

	var autoDiscoveryConfigurations []cloudproviderapi.AutoDiscoveryConfiguration
	autoDiscoveryConfigurations, diags = r.convertAutoDiscoveryConfigurationsToRequest(ctx, planData.AutoDiscoveryConfiguration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credential.AutoDiscoveryConfiguration = autoDiscoveryConfigurations

	diags = planData.ResourceTagsToAddToMetrics.ElementsAs(ctx, &credential.ResourceTagsToAddToMetrics, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentialResponse, err := r.client.UpdateAzureCredential(
		ctx,
		planData.StackID.ValueString(),
		planData.ResourceID.ValueString(),
		credential,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Azure credential", err.Error())
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("name"), credentialResponse.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("tenant_id"), credentialResponse.TenantID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_id"), credentialResponse.ClientID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_secret"), planData.ClientSecret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	convertedTagFilters, diags := r.convertTagFilters(ctx, credential.ResourceTagFilters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("resource_discovery_tag_filter"), convertedTagFilters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	convertedAutoDiscoveryConfigurations, diags := r.convertAutoDiscoveryConfigurations(ctx, credential.AutoDiscoveryConfiguration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("auto_discovery_configuration"), convertedAutoDiscoveryConfigurations)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("resource_tags_to_add_to_metrics"), planData.ResourceTagsToAddToMetrics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceAzureCredential) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceAzureCredentialModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAzureCredential(
		ctx,
		data.StackID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete Azure credential", err.Error())
		return
	}

	resp.State.Set(ctx, nil)
}

func (r *resourceAzureCredential) convertAutoDiscoveryConfigurations(ctx context.Context, configurations []cloudproviderapi.AutoDiscoveryConfiguration) (types.List, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	autoDiscoveryConfigListObjType := types.ObjectType{AttrTypes: azureAutoDiscoveryConfigurationModel{}.attrTypes()}

	autoDiscoveryConfigsTF := make([]azureAutoDiscoveryConfigurationModel, len(configurations))
	for i, config := range configurations {
		resourceTypeConfigsTF := make([]azureResourceTypeConfigurationModel, len(config.ResourceTypeConfigurations))
		for j, resourceTypeConfig := range config.ResourceTypeConfigurations {
			metricConfigsTF := make([]azureMetricConfigurationModel, len(resourceTypeConfig.MetricConfiguration))
			for k, metricConfig := range resourceTypeConfig.MetricConfiguration {
				metricConfigsTFDimensions, diags := types.ListValueFrom(ctx, types.StringType, metricConfig.Dimensions)
				conversionDiags.Append(diags...)
				if conversionDiags.HasError() {
					return types.ListNull(autoDiscoveryConfigListObjType), conversionDiags
				}

				metricConfigsTFAggregations, diags := types.ListValueFrom(ctx, types.StringType, metricConfig.Aggregations)
				conversionDiags.Append(diags...)
				if conversionDiags.HasError() {
					return types.ListNull(autoDiscoveryConfigListObjType), conversionDiags
				}
				metricConfigsTF[k] = azureMetricConfigurationModel{
					Name:         types.StringValue(metricConfig.Name),
					Dimensions:   metricConfigsTFDimensions,
					Aggregations: metricConfigsTFAggregations,
				}
			}
			resourceTypeConfigsTFMetricConfiguration, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: azureMetricConfigurationModel{}.attrTypes()}, metricConfigsTF)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return types.ListNull(autoDiscoveryConfigListObjType), conversionDiags
			}
			resourceTypeConfigsTF[j] = azureResourceTypeConfigurationModel{
				ResourceTypeName:    types.StringValue(resourceTypeConfig.ResourceTypeName),
				MetricConfiguration: resourceTypeConfigsTFMetricConfiguration,
			}
		}

		autoDiscoveryConfigsTFResourceTypeConfigurations, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: azureResourceTypeConfigurationModel{}.attrTypes()}, resourceTypeConfigsTF)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return types.ListNull(autoDiscoveryConfigListObjType), conversionDiags
		}
		autoDiscoveryConfigsTF[i] = azureAutoDiscoveryConfigurationModel{
			SubscriptionID:             types.StringValue(config.SubscriptionID),
			ResourceTypeConfigurations: autoDiscoveryConfigsTFResourceTypeConfigurations,
		}
	}

	autoDiscoveryConfigsTFList, diags := types.ListValueFrom(ctx, autoDiscoveryConfigListObjType, autoDiscoveryConfigsTF)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return types.ListNull(autoDiscoveryConfigListObjType), conversionDiags
	}
	return autoDiscoveryConfigsTFList, conversionDiags
}

func (r *resourceAzureCredential) convertAutoDiscoveryConfigurationsToRequest(ctx context.Context, configuration types.List) ([]cloudproviderapi.AutoDiscoveryConfiguration, diag.Diagnostics) {
	var autoDiscoveryConfigs []cloudproviderapi.AutoDiscoveryConfiguration
	var diags diag.Diagnostics

	var autoDiscoveryConfigModels []azureAutoDiscoveryConfigurationModel
	diags = configuration.ElementsAs(ctx, &autoDiscoveryConfigModels, false)
	if diags.HasError() {
		return nil, diags
	}

	for _, configModel := range autoDiscoveryConfigModels {
		var resourceTypeConfigs []cloudproviderapi.ResourceTypeConfiguration

		var resourceTypeConfigModels []azureResourceTypeConfigurationModel
		diags = configModel.ResourceTypeConfigurations.ElementsAs(ctx, &resourceTypeConfigModels, false)
		if diags.HasError() {
			return nil, diags
		}

		for _, resourceTypeConfigModel := range resourceTypeConfigModels {
			var metricConfigs []cloudproviderapi.MetricConfiguration

			var metricConfigModels []azureMetricConfigurationModel
			diags = resourceTypeConfigModel.MetricConfiguration.ElementsAs(ctx, &metricConfigModels, false)
			if diags.HasError() {
				return nil, diags
			}

			for _, metricConfigModel := range metricConfigModels {
				var dimensions []string
				diags = metricConfigModel.Dimensions.ElementsAs(ctx, &dimensions, false)
				if diags.HasError() {
					return nil, diags
				}

				var aggregations []string
				diags = metricConfigModel.Aggregations.ElementsAs(ctx, &aggregations, false)
				if diags.HasError() {
					return nil, diags
				}

				metricConfigs = append(metricConfigs, cloudproviderapi.MetricConfiguration{
					Name:         metricConfigModel.Name.ValueString(),
					Dimensions:   dimensions,
					Aggregations: aggregations,
				})
			}

			resourceTypeConfigs = append(resourceTypeConfigs, cloudproviderapi.ResourceTypeConfiguration{
				ResourceTypeName:    resourceTypeConfigModel.ResourceTypeName.ValueString(),
				MetricConfiguration: metricConfigs,
			})
		}

		autoDiscoveryConfigs = append(autoDiscoveryConfigs, cloudproviderapi.AutoDiscoveryConfiguration{
			SubscriptionID:             configModel.SubscriptionID.ValueString(),
			ResourceTypeConfigurations: resourceTypeConfigs,
		})
	}

	return autoDiscoveryConfigs, diags
}

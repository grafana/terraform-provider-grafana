// This file contains

package provider

import (
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/asserts"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloudprovider"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/connections"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/fleetmanagement"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/frontendo11y"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/k6"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/oncall"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/syntheticmonitoring"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSources() []*common.DataSource {
	var resources []*common.DataSource
	resources = append(resources, cloud.DataSources...)
	resources = append(resources, grafana.DataSources...)
	resources = append(resources, machinelearning.DataSources...)
	resources = append(resources, oncall.DataSources...)
	resources = append(resources, slo.DataSources...)
	resources = append(resources, k6.DataSources...)
	resources = append(resources, syntheticmonitoring.DataSources...)
	resources = append(resources, cloudprovider.DataSources...)
	resources = append(resources, connections.DataSources...)
	resources = append(resources, fleetmanagement.DataSources...)
	resources = append(resources, frontendo11y.DataSources...)
	resources = append(resources, asserts.DataSources...)
	return resources
}

func legacySDKDataSources() map[string]*schema.Resource {
	result := make(map[string]*schema.Resource)
	for _, d := range DataSources() {
		schema := d.Schema
		if schema == nil {
			continue
		}
		result[d.Name] = schema
	}
	return result
}

func pluginFrameworkDataSources() []func() datasource.DataSource {
	var dataSources []func() datasource.DataSource
	for _, d := range DataSources() {
		schema := d.PluginFrameworkSchema
		if schema == nil {
			continue
		}
		dataSources = append(dataSources, func() datasource.DataSource { return schema })
	}
	return dataSources
}

func Resources() []*common.Resource {
	var resources []*common.Resource
	resources = append(resources, cloud.Resources...)
	resources = append(resources, grafana.Resources...)
	resources = append(resources, oncall.Resources...)
	resources = append(resources, machinelearning.Resources...)
	resources = append(resources, slo.Resources...)
	resources = append(resources, k6.Resources...)
	resources = append(resources, syntheticmonitoring.Resources...)
	resources = append(resources, cloudprovider.Resources...)
	resources = append(resources, connections.Resources...)
	resources = append(resources, fleetmanagement.Resources...)
	resources = append(resources, frontendo11y.Resources...)
	resources = append(resources, asserts.Resources...)
	return resources
}

func AppPlatformResources() []appplatform.NamedResource {
	return []appplatform.NamedResource{
		appplatform.Dashboard(),
		appplatform.Playlist(),
		appplatform.AlertEnrichment(),
		appplatform.AppO11yConfigResource(),
		appplatform.K8sO11yConfigResource(),
	}
}

func ResourcesMap() map[string]*common.Resource {
	result := make(map[string]*common.Resource)
	for _, r := range Resources() {
		result[r.Name] = r
	}
	return result
}

func legacySDKResources() map[string]*schema.Resource {
	result := make(map[string]*schema.Resource)
	for _, r := range Resources() {
		schema := r.Schema
		if schema == nil {
			continue
		}
		result[r.Name] = schema
	}
	return result
}

func pluginFrameworkResources() []func() resource.Resource {
	var resources []func() resource.Resource

	for _, r := range Resources() {
		resourceSchema := r.PluginFrameworkSchema
		if resourceSchema == nil {
			continue
		}
		resources = append(resources, func() resource.Resource { return resourceSchema })
	}

	for _, r := range AppPlatformResources() {
		resources = append(resources, func() resource.Resource { return r.Resource })
	}

	return resources
}

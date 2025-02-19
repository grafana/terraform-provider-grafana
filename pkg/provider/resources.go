// This file contains

package provider

import (
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudprovider"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/connections"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/fleetmanagement"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/oncall"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/syntheticmonitoring"
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
	resources = append(resources, syntheticmonitoring.DataSources...)
	resources = append(resources, cloudprovider.DataSources...)
	resources = append(resources, connections.DataSources...)
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
	resources = append(resources, machinelearning.Resources...)
	resources = append(resources, oncall.Resources...)
	resources = append(resources, slo.Resources...)
	resources = append(resources, syntheticmonitoring.Resources...)
	resources = append(resources, cloudprovider.Resources...)
	resources = append(resources, connections.Resources...)
	resources = append(resources, fleetmanagement.Resources...)
	return resources
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
	return resources
}

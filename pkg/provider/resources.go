// This file contains

package provider

import (
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloudprovider"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/oncall"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/syntheticmonitoring"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Resources() []*common.Resource {
	var resources []*common.Resource
	resources = append(resources, cloud.Resources...)
	resources = append(resources, cloudprovider.Resources...)
	resources = append(resources, grafana.Resources...)
	resources = append(resources, machinelearning.Resources...)
	resources = append(resources, oncall.Resources...)
	resources = append(resources, slo.Resources...)
	resources = append(resources, syntheticmonitoring.Resources...)
	return resources
}

func resourceMap() map[string]*schema.Resource {
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

func mergeResourceMaps(maps ...map[string]*schema.Resource) map[string]*schema.Resource {
	result := make(map[string]*schema.Resource)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
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

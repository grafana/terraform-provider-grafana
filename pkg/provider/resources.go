// This file contains

package provider

import (
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/oncall"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/syntheticmonitoring"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Resources() []*common.Resource {
	var resources []*common.Resource
	resources = append(resources, cloud.Resources...)
	resources = append(resources, machinelearning.Resources...)
	resources = append(resources, oncall.Resources...)
	resources = append(resources, slo.Resources...)
	resources = append(resources, syntheticmonitoring.Resources...)
	return resources
}

func resourceMap() map[string]*schema.Resource {
	result := make(map[string]*schema.Resource)
	for _, r := range Resources() {
		result[r.Name] = r.Schema
	}

	// TODO: Migrate to common.Resource instances (in Resources function)
	return mergeResourceMaps(
		result,
		grafana.ResourcesMap,
	)
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

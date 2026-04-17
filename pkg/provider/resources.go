package provider

import (
	"reflect"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	appplatformgeneric "github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/generic"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/asserts"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloudintegrations"
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
	var dataSources []*common.DataSource
	dataSources = append(dataSources, cloud.DataSources...)
	dataSources = append(dataSources, grafana.DataSources...)
	dataSources = append(dataSources, machinelearning.DataSources...)
	dataSources = append(dataSources, oncall.DataSources...)
	dataSources = append(dataSources, slo.DataSources...)
	dataSources = append(dataSources, k6.DataSources...)
	dataSources = append(dataSources, syntheticmonitoring.DataSources...)
	dataSources = append(dataSources, cloudprovider.DataSources...)
	dataSources = append(dataSources, connections.DataSources...)
	dataSources = append(dataSources, fleetmanagement.DataSources...)
	dataSources = append(dataSources, frontendo11y.DataSources...)
	dataSources = append(dataSources, asserts.DataSources...)
	return dataSources
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
		if d.PluginFrameworkSchema == nil {
			continue
		}
		// Capture a reflect.Value of the template so each factory call returns a
		// fresh copy (preserving initialized fields like resourceType while resetting
		// client/config to nil so Configure runs correctly for each new provider).
		tmpl := reflect.ValueOf(d.PluginFrameworkSchema)
		dataSources = append(dataSources, func() datasource.DataSource {
			newPtr := reflect.New(tmpl.Elem().Type())
			newPtr.Elem().Set(tmpl.Elem())
			return newPtr.Interface().(datasource.DataSource)
		})
	}
	return dataSources
}

func Resources() []*common.Resource {
	var resources []*common.Resource
	resources = append(resources, cloud.Resources...)
	resources = append(resources, grafana.Resources...)
	resources = append(resources, cloudintegrations.Resources...)
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
		appplatformgeneric.GenericResource(),
		appplatform.Dashboard(),
		appplatform.DashboardV2(),
		appplatform.DashboardV2Stable(),
		appplatform.PlaylistV0Alpha1(),
		appplatform.PlaylistV1(),
		appplatform.AlertEnrichment(),
		appplatform.AlertRule(),
		appplatform.InhibitionRule(),
		appplatform.RecordingRule(),
		appplatform.AppO11yConfigResource(),
		appplatform.K8sO11yConfigResource(),
		appplatform.DbO11yConfigResource(),
		appplatform.Repository(),
		appplatform.Connection(),
		appplatform.Keeper(),
		appplatform.SecureValue(),
		appplatform.KeeperActivation(),
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
		if r.PluginFrameworkSchema == nil {
			continue
		}
		// Capture a reflect.Value of the template so each factory call returns a
		// fresh copy (preserving initialized fields like resourceType while resetting
		// client/config to nil so Configure runs correctly for each new provider).
		tmpl := reflect.ValueOf(r.PluginFrameworkSchema)
		resources = append(resources, func() resource.Resource {
			newPtr := reflect.New(tmpl.Elem().Type())
			newPtr.Elem().Set(tmpl.Elem())
			return newPtr.Interface().(resource.Resource)
		})
	}

	for _, r := range AppPlatformResources() {
		resources = append(resources, func() resource.Resource { return r.Resource })
	}

	return resources
}

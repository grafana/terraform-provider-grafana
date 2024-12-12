package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ResourceCategory string

var (
	CategoryAlerting            ResourceCategory = "Alerting"
	CategoryCloud               ResourceCategory = "Cloud"
	CategoryGrafanaEnterprise   ResourceCategory = "Grafana Enterprise"
	CategoryGrafanaOSS          ResourceCategory = "Grafana OSS"
	CategoryMachineLearning     ResourceCategory = "Machine Learning"
	CategoryOnCall              ResourceCategory = "OnCall"
	CategorySLO                 ResourceCategory = "SLO"
	CategorySyntheticMonitoring ResourceCategory = "Synthetic Monitoring"
	CategoryCloudProvider       ResourceCategory = "Cloud Provider"
	CategoryConnections         ResourceCategory = "Connections"
	CategoryFleetManagement     ResourceCategory = "Fleet Management"
)

type ResourceCommon struct {
	Name     string
	Schema   *schema.Resource // Legacy SDKv2 schema
	Category ResourceCategory
}

// DataSource represents a Terraform data source, implemented either with the SDKv2 or Terraform Plugin Framework.
type DataSource struct {
	ResourceCommon
	PluginFrameworkSchema datasource.DataSourceWithConfigure
}

func NewLegacySDKDataSource(category ResourceCategory, name string, schema *schema.Resource) *DataSource {
	d := &DataSource{
		ResourceCommon: ResourceCommon{
			Name:     name,
			Schema:   schema,
			Category: category,
		},
	}
	return d
}

func NewDataSource(category ResourceCategory, name string, schema datasource.DataSourceWithConfigure) *DataSource {
	d := &DataSource{
		ResourceCommon: ResourceCommon{
			Name:     name,
			Category: category,
		},
		PluginFrameworkSchema: schema,
	}
	return d
}

// ResourceListIDsFunc is a function that returns a list of resource IDs.
// This is used to generate TF config from existing resources.
// The data arg can be used to pass information between different listers. For example, the list of stacks will be used when listing stack plugins.
type ResourceListIDsFunc func(ctx context.Context, client *Client, data any) ([]string, error)

// Resource represents a Terraform resource, implemented either with the SDKv2 or Terraform Plugin Framework.
type Resource struct {
	ResourceCommon
	IDType                *ResourceID
	PluginFrameworkSchema resource.ResourceWithConfigure

	// Generation configuration
	ListIDsFunc                ResourceListIDsFunc
	PreferredResourceNameField string // This field will be used as the resource name instead of the ID. This is useful if the ID is not ideal for humans (ex: UUID or numeric). The field value should uniquely identify the resource.
}

func NewLegacySDKResource(category ResourceCategory, name string, idType *ResourceID, schema *schema.Resource) *Resource {
	r := &Resource{
		ResourceCommon: ResourceCommon{
			Name:     name,
			Schema:   schema,
			Category: category,
		},
		IDType: idType,
	}
	return r
}

func NewResource(category ResourceCategory, name string, idType *ResourceID, schema resource.ResourceWithConfigure) *Resource {
	r := &Resource{
		ResourceCommon: ResourceCommon{
			Name:     name,
			Category: category,
		},
		IDType:                idType,
		PluginFrameworkSchema: schema,
	}
	return r
}

func (r *Resource) WithLister(lister ResourceListIDsFunc) *Resource {
	r.ListIDsFunc = lister
	return r
}

func (r *Resource) WithPreferredResourceNameField(fieldName string) *Resource {
	r.PreferredResourceNameField = fieldName
	return r
}

func (r *Resource) ImportExample() string {
	exampleFromFields := func(fields []ResourceIDField) string {
		fieldTemplates := make([]string, len(fields))
		for i := range fields {
			fieldTemplates[i] = fmt.Sprintf("{{ %s }}", fields[i].Name)
		}
		return fmt.Sprintf(`terraform import %s.name %q
`, r.Name, strings.Join(fieldTemplates, ResourceIDSeparator))
	}

	id := r.IDType
	example := exampleFromFields(id.RequiredFields())
	if len(id.expectedFields) != len(id.RequiredFields()) {
		example += exampleFromFields(id.expectedFields)
	}

	return example
}

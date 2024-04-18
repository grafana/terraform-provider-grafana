package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceListIDsFunc is a function that returns a list of resource IDs.
// This is used to generate TF config from existing resources.
// The data arg can be used to pass information between different listers. For example, the list of stacks will be used when listing stack plugins.
type ResourceListIDsFunc func(ctx context.Context, client *Client, data any) ([]string, error)

// Resource represents a Terraform resource, implemented either with the SDKv2 or Terraform Plugin Framework.
type Resource struct {
	Name                  string
	IDType                *ResourceID
	ListIDsFunc           ResourceListIDsFunc
	Schema                *schema.Resource
	PluginFrameworkSchema resource.ResourceWithConfigure
}

func NewLegacySDKResource(name string, idType *ResourceID, schema *schema.Resource) *Resource {
	r := &Resource{
		Name:   name,
		IDType: idType,
		Schema: schema,
	}
	return r
}

func NewResource(name string, idType *ResourceID, schema resource.ResourceWithConfigure) *Resource {
	r := &Resource{
		Name:                  name,
		IDType:                idType,
		PluginFrameworkSchema: schema,
	}
	return r
}

func (r *Resource) WithLister(lister ResourceListIDsFunc) *Resource {
	r.ListIDsFunc = lister
	return r
}

func (r *Resource) ImportExample() string {
	exampleFromFields := func(fields []ResourceIDField) string {
		fieldTemplates := make([]string, len(fields))
		for i := range fields {
			fieldTemplates[i] = fmt.Sprintf("{{ %s }}", fields[i].Name)
		}
		return fmt.Sprintf(`terraform import %s.name %q
`, r.Name, strings.Join(fieldTemplates, defaultSeparator))
	}

	id := r.IDType
	example := exampleFromFields(id.RequiredFields())
	if len(id.expectedFields) != len(id.RequiredFields()) {
		example += exampleFromFields(id.expectedFields)
	}

	return example
}

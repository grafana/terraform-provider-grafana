package common

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ResourceListIDsFunc func(ctx context.Context, cache *sync.Map, client *Client) ([]string, error)
type Resource struct {
	Name        string
	IDType      *ResourceID
	ListIDsFunc ResourceListIDsFunc
	Schema      *schema.Resource
}

func NewResource(name string, idType *ResourceID, schema *schema.Resource) *Resource {
	return NewResourceWithLister(name, idType, nil, schema)
}

func NewResourceWithLister(name string, idType *ResourceID, lister ResourceListIDsFunc, schema *schema.Resource) *Resource {
	r := &Resource{
		Name:        name,
		IDType:      idType,
		ListIDsFunc: lister,
		Schema:      schema,
	}
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

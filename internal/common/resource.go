package common

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Resource struct {
	Name   string
	IDType *ResourceID
	Schema *schema.Resource
}

func NewResource(name string, idType *ResourceID, schema *schema.Resource) *Resource {
	r := &Resource{
		Name:   name,
		IDType: idType,
		Schema: schema,
	}
	return r
}

func (r *Resource) ImportExample() string {
	id := r.IDType
	fields := make([]string, len(id.expectedFields))
	for i := range fields {
		fields[i] = fmt.Sprintf("{{ %s }}", id.expectedFields[i].Name)
	}
	return fmt.Sprintf(`terraform import %s.name %q
`, r.Name, strings.Join(fields, defaultSeparator))
}

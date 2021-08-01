package grafana

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// schemaDiffFloat32 is a SchemaDiffSuppressFunc for diffing float32 values.
// schema.TypeFloat uses float64, which is a problem for API types that use
// float32. Terraform automatically converts float32 to float64 which changes
// the precision and causes incorrect diffs.
//
// For example, synthetic_monitoring.Probe.Latitude is float32. Attempting to
// set grafanacloud_synthetic_monitoring_probe.latitude to 27.98606 results in
// 27.986059188842773. The solution is to diff old and new values as float32.
func schemaDiffFloat32(k, old string, nw string, d *schema.ResourceData) bool {
	old32, _ := strconv.ParseFloat(old, 32)
	nw32, _ := strconv.ParseFloat(nw, 32)
	return old32 == nw32
}

// datasourceSchemaFromResourceSchema is a recursive func that
// converts an existing Resource schema to a Datasource schema.
// All schema elements are copied, but certain attributes are ignored or changed:
// - all attributes have Computed = true
// - all attributes have ForceNew, Required = false
// - Validation funcs and attributes (e.g. MaxItems) are not copied
func datasourceSchemaFromResourceSchema(rs map[string]*schema.Schema) map[string]*schema.Schema {
	ds := make(map[string]*schema.Schema, len(rs))
	for k, v := range rs {
		dv := &schema.Schema{
			Computed:    true,
			ForceNew:    false,
			Required:    false,
			Description: v.Description,
			Type:        v.Type,
		}
		switch v.Type {
		case schema.TypeSet:
			dv.Set = v.Set
			fallthrough
		case schema.TypeList:
			// List & Set types are generally used for 2 cases:
			// - a list/set of simple primitive values (e.g. list of strings)
			// - a sub resource
			if elem, ok := v.Elem.(*schema.Resource); ok {
				// handle the case where the Element is a sub-resource
				dv.Elem = &schema.Resource{
					Schema: datasourceSchemaFromResourceSchema(elem.Schema),
				}
			} else {
				// handle simple primitive case
				dv.Elem = v.Elem
			}
		default:
			// Elem of all other types are copied as-is
			dv.Elem = v.Elem
		}
		ds[k] = dv
	}
	return ds
}

// fixDatasourceSchemaFlags is a convenience func that toggles the Computed,
// Optional + Required flags on a schema element. This is useful when the schema
// has been generated (using `datasourceSchemaFromResourceSchema` above for
// example) and therefore the attribute flags were not set appropriately when
// first added to the schema definition. Currently only supports top-level
// schema elements.
func fixDatasourceSchemaFlags(schema map[string]*schema.Schema, required bool, keys ...string) {
	for _, v := range keys {
		schema[v].Computed = false
		schema[v].Optional = !required
		schema[v].Required = required
	}
}

func addRequiredFieldsToSchema(schema map[string]*schema.Schema, keys ...string) {
	fixDatasourceSchemaFlags(schema, true, keys...)
}

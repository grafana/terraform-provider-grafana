package postprocessing

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func traversal(root string, attrs ...string) hcl.Traversal {
	tr := hcl.Traversal{hcl.TraverseRoot{Name: root}}
	for _, attr := range attrs {
		tr = append(tr, hcl.TraverseAttr{Name: attr})
	}
	return tr
}

func attributeToMap(attr *hclwrite.Attribute) (map[string]any, error) {
	var err error

	// Convert jsonencode to raw json
	s := strings.TrimPrefix(string(attr.Expr().BuildTokens(nil).Bytes()), " ")

	if strings.HasPrefix(s, "jsonencode(") {
		return nil, nil // Figure out how to handle those
	}

	if !strings.HasPrefix(s, "\"") {
		// if expr is not a string, assume it's already converted, return (idempotency
		return nil, nil
	}
	s, err = strconv.Unquote(s)
	if err != nil {
		return nil, err
	}
	s = strings.ReplaceAll(s, "$${", "${") // These are escaped interpolations

	var dashboardMap map[string]any
	err = json.Unmarshal([]byte(s), &dashboardMap)
	if err != nil {
		return nil, err
	}

	return dashboardMap, nil
}

func extractJSONEncode(value string) (string, error) {
	if !strings.HasPrefix(value, "jsonencode(") {
		return "", nil
	}
	value = strings.TrimPrefix(value, "jsonencode(")
	value = strings.TrimSuffix(value, ")")

	b, err := json.MarshalIndent(value, "", "  ")
	return string(b), err
}

// BELOW IS FROM https://github.com/hashicorp/terraform/blob/main/internal/configs/hcl2shim/values.go

// UnknownVariableValue is a sentinel value that can be used
// to denote that the value of a variable is unknown at this time.
// RawConfig uses this information to build up data about
// unknown keys.
const unknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// hcl2ValueFromConfigValue is the opposite of configValueFromHCL2: it takes
// a value as would be returned from the old interpolator and turns it into
// a cty.Value so it can be used within, for example, an HCL2 EvalContext.
func hcl2ValueFromConfigValue(v any) cty.Value {
	if v == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	if v == unknownVariableValue {
		return cty.DynamicVal
	}

	switch tv := v.(type) {
	case bool:
		return cty.BoolVal(tv)
	case string:
		return cty.StringVal(tv)
	case int:
		return cty.NumberIntVal(int64(tv))
	case float64:
		return cty.NumberFloatVal(tv)
	case []any:
		vals := make([]cty.Value, len(tv))
		for i, ev := range tv {
			vals[i] = hcl2ValueFromConfigValue(ev)
		}
		return cty.TupleVal(vals)
	case map[string]any:
		vals := map[string]cty.Value{}
		for k, ev := range tv {
			vals[k] = hcl2ValueFromConfigValue(ev)
		}
		return cty.ObjectVal(vals)
	default:
		// HCL/HIL should never generate anything that isn't caught by
		// the above, so if we get here something has gone very wrong.
		panic(fmt.Errorf("can't convert %#v to cty.Value", v))
	}
}

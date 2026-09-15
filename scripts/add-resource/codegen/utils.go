package codegen

import (
	"strings"

	"cuelang.org/go/cue"
	"github.com/huandu/xstrings"
)

func SnakeCase(input string) string {
	return xstrings.ToSnakeCase(input)
}

func getTypePrefix(val cue.Value) string {
	if attr := val.Attribute("grafana_app_sdk"); attr.Err() == nil {
		prefix, ok, err := attr.Lookup(0, "prefix")
		if ok && err == nil {
			return exportField(prefix)
		}
	}
	return ""
}

// exportField makes a field name exported
func exportField(field string) string {
	if len(field) > 0 {
		return strings.ToUpper(field[:1]) + field[1:]
	}
	return strings.ToUpper(field)
}

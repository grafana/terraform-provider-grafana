package fleetmanagement

import (
	"bytes"
	"context"
	"fmt"

	"github.com/grafana/river/parser"
	"github.com/grafana/river/printer"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ basetypes.StringValuable                   = PipelineConfigValue{}
	_ basetypes.StringValuableWithSemanticEquals = PipelineConfigValue{}
	// NOTE: Validation is done at resource level via ConfigValidators() to access config_type field
	yamlParser = Parser()
)

type PipelineConfigValue struct {
	basetypes.StringValue
}

func NewPipelineConfigValue(value string) PipelineConfigValue {
	return PipelineConfigValue{
		StringValue: basetypes.NewStringValue(value),
	}
}

func (v PipelineConfigValue) Equal(o attr.Value) bool {
	other, ok := o.(PipelineConfigValue)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

func (v PipelineConfigValue) Type(ctx context.Context) attr.Type {
	return PipelineConfigType
}

func (v PipelineConfigValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(PipelineConfigValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Expected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)

		return false, diags
	}

	oldStr := v.ValueString()
	newStr := newValue.ValueString()

	// Try Alloy semantic equality first
	if equal, err := riverEqual(oldStr, newStr); err == nil {
		return equal, diags
	}

	// OTel Collectors will fail the Alloy semantic equality check, fall back to YAML semantic equality
	if equal, err := yamlEqual(oldStr, newStr); err == nil {
		return equal, diags
	}

	return oldStr == newStr, diags
}

func riverEqual(contents1 string, contents2 string) (bool, error) {
	parsed1, err := parseRiver(contents1)
	if err != nil {
		return false, err
	}

	parsed2, err := parseRiver(contents2)
	if err != nil {
		return false, err
	}

	return parsed1 == parsed2, nil
}

func parseRiver(contents string) (string, error) {
	file, err := parser.ParseFile("", []byte(contents))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, file); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func yamlEqual(contents1, contents2 string) (bool, error) {
	parsed1, err := parseYAML(contents1)
	if err != nil {
		return false, err
	}

	parsed2, err := parseYAML(contents2)
	if err != nil {
		return false, err
	}

	return parsed1 == parsed2, nil
}

func parseYAML(contents string) (string, error) {
	data, err := yamlParser.Unmarshal([]byte(contents))
	if err != nil {
		return "", fmt.Errorf("invalid YAML syntax: %w", err)
	}

	out, err := yamlParser.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML content: %w", err)
	}
	return string(out), nil
}

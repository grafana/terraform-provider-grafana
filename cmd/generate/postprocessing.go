package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func stripDefaults(fpath string, extraFieldsToRemove map[string]string) error {
	file, err := readHCLFile(fpath)
	if err != nil {
		return err
	}

	hasChanges := false
	for _, block := range file.Body().Blocks() {
		if s := stripDefaultsFromBlock(block, extraFieldsToRemove); s {
			hasChanges = true
		}
	}
	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		return os.WriteFile(fpath, file.Bytes(), 0600)
	}
	return nil
}

func wrapJSONFieldsInFunction(fpath string) error {
	file, err := readHCLFile(fpath)
	if err != nil {
		return err
	}

	hasChanges := false
	// Find json attributes and use jsonencode
	for _, block := range file.Body().Blocks() {
		for key, attr := range block.Body().Attributes() {
			asMap, err := attributeToMap(attr)
			if err != nil || asMap == nil {
				continue
			}
			tokens := hclwrite.TokensForValue(HCL2ValueFromConfigValue(asMap))
			block.Body().SetAttributeRaw(key, hclwrite.TokensForFunctionCall("jsonencode", tokens))
			hasChanges = true
		}
	}

	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		return os.WriteFile(fpath, file.Bytes(), 0600)
	}
	return nil
}

func abstractDashboards(fpath string) error {
	fDir := filepath.Dir(fpath)
	outPath := filepath.Join(fDir, "files")

	file, err := readHCLFile(fpath)
	if err != nil {
		return err
	}

	hasChanges := false
	dashboardJsons := map[string][]byte{}
	for _, block := range file.Body().Blocks() {
		labels := block.Labels()
		if len(labels) == 0 || labels[0] != "grafana_dashboard" {
			continue
		}

		dashboard, err := attributeToJSON(block.Body().GetAttribute("config_json"))
		if err != nil {
			return err
		}

		if dashboard == nil {
			continue
		}

		writeTo := filepath.Join(outPath, fmt.Sprintf("%s.json", block.Labels()[1]))

		// Replace $${ with ${ in the json. No need to escape in the json file
		dashboard = []byte(strings.ReplaceAll(string(dashboard), "$${", "${"))
		dashboardJsons[writeTo] = dashboard

		// Hacky relative path with interpolation
		relativePath := strings.ReplaceAll(writeTo, fDir, "")
		pathWithInterpolation := hclwrite.Tokens{
			{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)},
			{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte(`${`)},
			{Type: hclsyntax.TokenIdent, Bytes: []byte(`path.module`)},
			{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte(`}`)},
			{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(relativePath)},
			{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)},
		}

		block.Body().SetAttributeRaw(
			"config_json",
			hclwrite.TokensForFunctionCall("file", pathWithInterpolation),
		)

		hasChanges = true
	}
	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		os.Mkdir(outPath, 0755)
		for writeTo, dashboard := range dashboardJsons {
			err := os.WriteFile(writeTo, dashboard, 0600)
			if err != nil {
				panic(err)
			}
		}
		return os.WriteFile(fpath, file.Bytes(), 0600)
	}
	return nil
}

func attributeToMap(attr *hclwrite.Attribute) (map[string]interface{}, error) {
	s := string(attr.Expr().BuildTokens(nil).Bytes())
	s = strings.TrimPrefix(s, " ")
	if !strings.HasPrefix(s, "\"") {
		// if expr is not a string, assume it's already converted, return (idempotency
		return nil, nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return nil, err
	}
	s = strings.ReplaceAll(s, "$${", "${") // These are escaped interpolations

	var jsonMap map[string]interface{}
	err = json.Unmarshal([]byte(s), &jsonMap)
	if err != nil {
		return nil, err
	}

	return jsonMap, nil
}

func attributeToJSON(attr *hclwrite.Attribute) ([]byte, error) {
	jsonMap, err := attributeToMap(attr)
	if err != nil {
		return nil, err
	}

	jsonMarshalled, err := json.MarshalIndent(jsonMap, "", "\t")
	if err != nil {
		return nil, err
	}

	return jsonMarshalled, nil
}

func readHCLFile(fpath string) (*hclwrite.File, error) {
	src, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	file, diags := hclwrite.ParseConfig(src, fpath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, errors.New(diags.Error())
	}

	return file, nil
}

func stripDefaultsFromBlock(block *hclwrite.Block, extraFieldsToRemove map[string]string) bool {
	hasChanges := false
	for _, innblock := range block.Body().Blocks() {
		if s := stripDefaultsFromBlock(innblock, extraFieldsToRemove); s {
			hasChanges = true
		}
		if len(innblock.Body().Attributes()) == 0 && len(innblock.Body().Blocks()) == 0 {
			if rm := block.Body().RemoveBlock(innblock); rm {
				hasChanges = true
			}
		}
	}
	for name, attribute := range block.Body().Attributes() {
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " null" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " {}" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " []" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		for key, value := range extraFieldsToRemove {
			if name == key && string(attribute.Expr().BuildTokens(nil).Bytes()) == value {
				if rm := block.Body().RemoveAttribute(name); rm != nil {
					hasChanges = true
				}
			}
		}
	}
	return hasChanges
}

// BELOW IS FROM https://github.com/hashicorp/terraform/blob/main/internal/configs/hcl2shim/values.go

// UnknownVariableValue is a sentinel value that can be used
// to denote that the value of a variable is unknown at this time.
// RawConfig uses this information to build up data about
// unknown keys.
const UnknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// HCL2ValueFromConfigValue is the opposite of configValueFromHCL2: it takes
// a value as would be returned from the old interpolator and turns it into
// a cty.Value so it can be used within, for example, an HCL2 EvalContext.
func HCL2ValueFromConfigValue(v interface{}) cty.Value {
	if v == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	if v == UnknownVariableValue {
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
	case []interface{}:
		vals := make([]cty.Value, len(tv))
		for i, ev := range tv {
			vals[i] = HCL2ValueFromConfigValue(ev)
		}
		return cty.TupleVal(vals)
	case map[string]interface{}:
		vals := map[string]cty.Value{}
		for k, ev := range tv {
			vals[k] = HCL2ValueFromConfigValue(ev)
		}
		return cty.ObjectVal(vals)
	default:
		// HCL/HIL should never generate anything that isn't caught by
		// the above, so if we get here something has gone very wrong.
		panic(fmt.Errorf("can't convert %#v to cty.Value", v))
	}
}

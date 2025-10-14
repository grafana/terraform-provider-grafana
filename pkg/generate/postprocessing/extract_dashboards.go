package postprocessing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	tfjson "github.com/hashicorp/terraform-json"
)

func ExtractDashboards(fpath string, plannedState *tfjson.Plan) error {
	fDir := filepath.Dir(fpath)
	outPath := filepath.Join(fDir, "dashboards")

	return postprocessFile(fpath, func(file *hclwrite.File) error {
		dashboardJsons := map[string][]byte{}
		for _, block := range file.Body().Blocks() {
			labels := block.Labels()
			if len(labels) == 0 || labels[0] != "grafana_dashboard" {
				continue
			}

			var dashboardValue string
			for _, r := range plannedState.PlannedValues.RootModule.Resources {
				if r.Type != "grafana_dashboard" {
					continue
				}
				if r.Name != labels[1] {
					continue
				}
				dashboardValue = r.AttributeValues["config_json"].(string)
			}

			// Skip dashboards that have 10 or fewer attributes (counted by commas)
			// They are fine as inline JSON
			if strings.Count(dashboardValue, ",") <= 10 {
				continue
			}

			writeTo := filepath.Join(outPath, fmt.Sprintf("%s.json", block.Labels()[1]))
			dashboardJsons[writeTo] = []byte(dashboardValue)

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
		}

		if len(dashboardJsons) == 0 {
			return nil
		}

		if err := os.MkdirAll(outPath, 0755); err != nil {
			return err
		}
		for writeTo, dashboard := range dashboardJsons {
			dashboardFile, err := os.Create(writeTo)
			if err != nil {
				return err
			}

			// Parse the JSON to format it nicely
			var dashboardInterface any
			if err := json.Unmarshal(dashboard, &dashboardInterface); err != nil {
				return err
			}
			dashboard, err := json.MarshalIndent(dashboardInterface, "", "    ")
			if err != nil {
				return err
			}

			if _, err := dashboardFile.Write(dashboard); err != nil {
				return err
			}

			if err := dashboardFile.Close(); err != nil {
				return err
			}
		}

		return nil
	})
}

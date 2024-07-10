package postprocessing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func AbstractDashboards(fpath string) error {
	fDir := filepath.Dir(fpath)
	outPath := filepath.Join(fDir, "files")

	return postprocessFile(fpath, func(file *hclwrite.File) error {
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
		}

		if len(dashboardJsons) == 0 {
			return nil
		}

		if err := os.Mkdir(outPath, 0755); err != nil {
			return err
		}
		for writeTo, dashboard := range dashboardJsons {
			err := os.WriteFile(writeTo, dashboard, 0600)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

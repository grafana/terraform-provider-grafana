package codegen

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"github.com/grafana/codejen"
)

var _ codejen.OneToOne[cue.Value] = &GoAccTestGenerator{}

type GoAccTestGenerator struct {
	name           string
	kindName       string
	version        string
	appName        string
	serviceName    string
	groupOverride  string
	outputDir      string
	templatesDir   string
	grafanaVersion string
	isEnterprise   bool
}

type accTestTemplateData struct {
	Version        string
	KindImportPath string
	VarPrefix      string
	TFResourceType string
	Name           string
	LowerName      string
	KindName       string
	CheckFunc      string
	VersionArg     string
}

func (jenny *GoAccTestGenerator) JennyName() string {
	return "GoAccTestGenerator"
}

func (jenny *GoAccTestGenerator) Generate(_ cue.Value) (*codejen.File, error) {
	apiGroup := jenny.appName
	if jenny.groupOverride != "" {
		apiGroup = strings.SplitN(jenny.groupOverride, ".", 2)[0]
	}

	serviceName := jenny.serviceName
	if serviceName == "" {
		serviceName = jenny.appName
	}

	lowerName := strings.ToLower(jenny.name)
	// varPrefix is lowerCamelCase of jenny.name (e.g. "Check" → "check", "InhibitionRule" → "inhibitionRule")
	varPrefix := strings.ToLower(jenny.name[:1]) + jenny.name[1:]
	tfResourceType := fmt.Sprintf("grafana_apps_%s_%s_%s", apiGroup, lowerName, jenny.version)
	kindImportPath := fmt.Sprintf("github.com/grafana/grafana/apps/%s/pkg/apis/%s/%s",
		serviceName, apiGroup, jenny.version)

	checkFunc := "CheckOSSTestsEnabled"
	if jenny.isEnterprise {
		checkFunc = "CheckEnterpriseTestsEnabled"
	}

	versionArg := ""
	if jenny.grafanaVersion != "" {
		versionArg = fmt.Sprintf(", %q", jenny.grafanaVersion)
	}

	tmplPath := filepath.Join(jenny.templatesDir, "acc_test.tmpl")
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("reading acc test template: %w", err)
	}

	tmpl, err := template.New("acc_test").Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing acc test template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, accTestTemplateData{
		Version:        jenny.version,
		KindImportPath: kindImportPath,
		VarPrefix:      varPrefix,
		TFResourceType: tfResourceType,
		Name:           jenny.name,
		LowerName:      lowerName,
		KindName:       jenny.kindName,
		CheckFunc:      checkFunc,
		VersionArg:     versionArg,
	}); err != nil {
		return nil, fmt.Errorf("executing acc test template: %w", err)
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		// Return an unformatted source so the user can see what went wrong.
		formatted = []byte(buf.String())
	}

	outputPath := filepath.Join(jenny.outputDir, fmt.Sprintf("%s_resource_acc_test.go", lowerName))
	return codejen.NewFile(outputPath, formatted, jenny), nil
}

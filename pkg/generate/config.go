package generate

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type OutputFormat string

const (
	OutputFormatJSON       OutputFormat = "json"
	OutputFormatHCL        OutputFormat = "hcl"
	OutputFormatCrossplane OutputFormat = "crossplane"
)

var OutputFormats = []OutputFormat{OutputFormatJSON, OutputFormatHCL, OutputFormatCrossplane}

type GrafanaConfig struct {
	URL                 string
	Auth                string
	IsGrafanaCloudStack bool
	SMURL               string
	SMAccessToken       string
	OnCallURL           string
	OnCallAccessToken   string
}

type CloudConfig struct {
	AccessPolicyToken         string
	Org                       string
	CreateStackServiceAccount bool
	StackServiceAccountName   string
}

type TerraformInstallConfig struct {
	InstallDir string
	Version    *version.Version
	PluginDir  string
}

type Config struct {
	// IncludeResources is a list of patterns to filter resources by.
	// If a resource name matches any of the patterns, it will be included in the output.
	// Patterns are in the form of `resourceType.resourceName` and support * as a wildcard.
	IncludeResources []string
	// OutputDir is the directory to write the generated files to.
	OutputDir string
	// Clobber will overwrite existing files in the output directory.
	Clobber           bool
	OutputCredentials bool
	Format            OutputFormat
	ProviderVersion   string
	Grafana           *GrafanaConfig
	Cloud             *CloudConfig

	TerraformInstallConfig TerraformInstallConfig
	Terraform              *tfexec.Terraform
}

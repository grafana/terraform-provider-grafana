package generate

type OutputFormat string

const (
	OutputFormatJSON       OutputFormat = "json"
	OutputFormatHCL        OutputFormat = "hcl"
	OutputFormatCrossplane OutputFormat = "crossplane"
)

var OutputFormats = []OutputFormat{OutputFormatJSON, OutputFormatHCL, OutputFormatCrossplane}

type GrafanaConfig struct {
	URL  string
	Auth string
}

type CloudConfig struct {
	AccessPolicyToken         string
	Org                       string
	CreateStackServiceAccount bool
	StackServiceAccountName   string
}

type Config struct {
	OutputDir       string
	Clobber         bool
	Format          OutputFormat
	ProviderVersion string
	Grafana         *GrafanaConfig
	Cloud           *CloudConfig
}

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v4/pkg/generate"

	"github.com/fatih/color"
	goVersion "github.com/hashicorp/go-version"
	"github.com/urfave/cli/v2"
)

var version = "" // set by ldflags

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	msg := "WARNING: This tool is highly experimental and comes with no support or guarantees."
	lines := strings.Repeat("-", len(msg))
	color.New(color.FgRed, color.Bold).Fprintf(os.Stderr, "%[2]s\n%[1]s\n%[2]s\n", msg, lines)
	app := &cli.App{
		Name:      "terraform-provider-grafana-generate",
		Usage:     "Generate `terraform-provider-grafana` resources from your Grafana instance or Grafana Cloud account.",
		UsageText: "terraform-provider-grafana-generate [options]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "output-dir",
				Aliases:  []string{"o"},
				Usage:    "Output directory for generated resources",
				Required: true,
				EnvVars:  []string{"TFGEN_OUTPUT_DIR"},
			},
			&cli.BoolFlag{
				Name:    "clobber",
				Aliases: []string{"c"},
				Usage:   "Delete all files in the output directory before generating resources",
				EnvVars: []string{"TFGEN_CLOBBER"},
			},
			&cli.StringFlag{
				Name:    "output-format",
				Aliases: []string{"f"},
				Usage: fmt.Sprintf("Output format for generated resources. "+
					"Supported formats are: %v", generate.OutputFormats),
				Value:   string(generate.OutputFormatHCL),
				EnvVars: []string{"TFGEN_OUTPUT_FORMAT"},
			},
			&cli.StringFlag{
				Name:    "terraform-provider-version",
				Usage:   "Version of the Grafana provider to generate resources for. Defaults to the release version (same as the generator version).",
				EnvVars: []string{"TFGEN_TERRAFORM_PROVIDER_VERSION"},
				Value:   version,
			},
			&cli.StringSliceFlag{
				Name: "include-resources",
				Usage: `List of resources to include in the "resourceType.resourceName" format. If not set, all resources will be included
This supports a glob format. Examples:
  * Generate all dashboards and folders: --resource-names 'grafana_dashboard.*' --resource-names 'grafana_folder.*'
  * Generate all resources with "hello" in their ID (this is usually the resource UIDs): --resource-names '*.*hello*'
  * Generate all resources (same as default behaviour): --resource-names '*.*'
`,
				EnvVars:  []string{"TFGEN_INCLUDE_RESOURCES"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:    "output-credentials",
				Usage:   "Output credentials in the generated resources",
				EnvVars: []string{"TFGEN_OUTPUT_CREDENTIALS"},
				Value:   false,
			},
			&cli.StringFlag{
				Name:     "terraform-install-dir",
				Usage:    `Directory to install Terraform to. If not set, a temporary directory will be created.`,
				EnvVars:  []string{"TFGEN_TERRAFORM_INSTALL_DIR"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "terraform-install-version",
				Usage:    `Version of Terraform to install. If not set, the latest version _tested in this tool_ will be installed.`,
				EnvVars:  []string{"TFGEN_TERRAFORM_INSTALL_VERSION"},
				Required: false,
			},

			// Grafana OSS flags
			&cli.StringFlag{
				Name:     "grafana-url",
				Usage:    "URL of the Grafana instance to generate resources from",
				Category: "Grafana",
				EnvVars:  []string{"TF_GEN_GRAFANA_URL"},
			},
			&cli.StringFlag{
				Name:     "grafana-auth",
				Usage:    "Service account token or username:password for the Grafana instance",
				Category: "Grafana",
				EnvVars:  []string{"TFGEN_GRAFANA_AUTH"},
			},
			&cli.BoolFlag{
				Name:     "grafana-is-cloud-stack",
				Usage:    "Indicates that the Grafana instance is a Grafana Cloud stack",
				Category: "Grafana",
				EnvVars:  []string{"TFGEN_GRAFANA_IS_CLOUD_STACK"},
			},
			&cli.StringFlag{
				Name:     "synthetic-monitoring-url",
				Usage:    "URL of the Synthetic Monitoring instance to generate resources from",
				Category: "Grafana",
				EnvVars:  []string{"TFGEN_SYNTHETIC_MONITORING_URL"},
			},
			&cli.StringFlag{
				Name:     "synthetic-monitoring-access-token",
				Usage:    "API token for the Synthetic Monitoring instance",
				Category: "Grafana",
				EnvVars:  []string{"TFGEN_SYNTHETIC_MONITORING_ACCESS_TOKEN"},
			},
			&cli.StringFlag{
				Name:     "oncall-url",
				Usage:    "URL of the OnCall instance to generate resources from",
				Category: "Grafana",
				EnvVars:  []string{"TFGEN_ONCALL_URL"},
			},
			&cli.StringFlag{
				Name:     "oncall-access-token",
				Usage:    "API token for the OnCall instance",
				Category: "Grafana",
				EnvVars:  []string{"TFGEN_ONCALL_ACCESS_TOKEN"},
			},

			// Grafana Cloud flags
			&cli.StringFlag{
				Name:     "cloud-access-policy-token",
				Usage:    "Access policy token for Grafana Cloud",
				Category: "Grafana Cloud",
				EnvVars:  []string{"TFGEN_CLOUD_ACCESS_POLICY_TOKEN"},
			},
			&cli.StringFlag{
				Name:     "cloud-org",
				Usage:    "Organization ID or name for Grafana Cloud",
				Category: "Grafana Cloud",
				EnvVars:  []string{"TFGEN_CLOUD_ORG"},
			},
			&cli.BoolFlag{
				Name:     "cloud-create-stack-service-account",
				Usage:    "Create a service account for each Grafana Cloud stack, allowing generation and management of resources in that stack.",
				Category: "Grafana Cloud",
				EnvVars:  []string{"TFGEN_CLOUD_CREATE_STACK_SERVICE_ACCOUNT"},
			},
			&cli.StringFlag{
				Name:     "cloud-stack-service-account-name",
				Usage:    "Name of the service account to create for each Grafana Cloud stack.",
				Category: "Grafana Cloud",
				EnvVars:  []string{"TFGEN_CLOUD_STACK_SERVICE_ACCOUNT_NAME"},
				Value:    "tfgen-management",
			},
		},
		InvalidFlagAccessHandler: func(ctx *cli.Context, s string) {
			panic(fmt.Errorf("invalid flag access: %s", s))
		},
		Action: func(ctx *cli.Context) error {
			cfg, err := parseFlags(ctx)
			if err != nil {
				return fmt.Errorf("failed to parse flags: %w", err)
			}
			result := generate.Generate(ctx.Context, cfg)
			return errors.Join(result.Errors...)
		},
	}

	return app.Run(os.Args)
}

func parseFlags(ctx *cli.Context) (*generate.Config, error) {
	config := &generate.Config{
		OutputDir:         ctx.String("output-dir"),
		Clobber:           ctx.Bool("clobber"),
		Format:            generate.OutputFormat(ctx.String("output-format")),
		ProviderVersion:   ctx.String("terraform-provider-version"),
		OutputCredentials: ctx.Bool("output-credentials"),
		Grafana: &generate.GrafanaConfig{
			URL:                 ctx.String("grafana-url"),
			Auth:                ctx.String("grafana-auth"),
			IsGrafanaCloudStack: ctx.Bool("grafana-is-cloud-stack"),
			SMURL:               ctx.String("synthetic-monitoring-url"),
			SMAccessToken:       ctx.String("synthetic-monitoring-access-token"),
			OnCallURL:           ctx.String("oncall-url"),
			OnCallAccessToken:   ctx.String("oncall-access-token"),
		},
		Cloud: &generate.CloudConfig{
			AccessPolicyToken:         ctx.String("cloud-access-policy-token"),
			Org:                       ctx.String("cloud-org"),
			CreateStackServiceAccount: ctx.Bool("cloud-create-stack-service-account"),
			StackServiceAccountName:   ctx.String("cloud-stack-service-account-name"),
		},
		IncludeResources: ctx.StringSlice("include-resources"),
		TerraformInstallConfig: generate.TerraformInstallConfig{
			InstallDir: ctx.String("terraform-install-dir"),
		},
	}
	var err error
	if tfVersion := ctx.String("terraform-install-version"); tfVersion != "" {
		config.TerraformInstallConfig.Version, err = goVersion.NewVersion(ctx.String("terraform-install-version"))
		if err != nil {
			return nil, fmt.Errorf("terraform-install-version must be a valid version: %w", err)
		}
	}

	if config.ProviderVersion == "" {
		return nil, fmt.Errorf("terraform-provider-version must be set")
	}

	// Validate flags
	err = newFlagValidations().
		atLeastOne("grafana-url", "cloud-access-policy-token").
		conflicting(
			[]string{"grafana-url", "grafana-auth", "synthetic-monitoring-url", "synthetic-monitoring-access-token", "oncall-url", "oncall-access-token"},
			[]string{"cloud-access-policy-token", "cloud-org", "cloud-create-stack-service-account", "cloud-stack-service-account-name"},
		).
		requiredWhenSet("grafana-url", "grafana-auth").
		requiredWhenSet("cloud-access-policy-token", "cloud-org").
		requiredWhenSet("cloud-stack-service-account-name", "cloud-create-stack-service-account").
		validate(ctx)
	if err != nil {
		return nil, err
	}

	if config.Grafana.Auth == "" {
		config.Grafana = nil
	}

	if config.Cloud.AccessPolicyToken == "" {
		config.Cloud = nil
	}

	return config, nil
}

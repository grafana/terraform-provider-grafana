package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var version = "" // set by ldflags

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
					"Supported formats are: %v", outputFormats),
				Value:   string(outputFormatHCL),
				EnvVars: []string{"TFGEN_OUTPUT_FORMAT"},
			},
			&cli.StringFlag{
				Name:    "terraform-provider-version",
				Usage:   "Version of the Grafana provider to generate resources for. Defaults to the release version (same as the generator version).",
				EnvVars: []string{"TFGEN_TERRAFORM_PROVIDER_VERSION"},
				Value:   version,
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
			return generate(ctx.Context, cfg)
		},
	}

	return app.Run(os.Args)
}

func parseFlags(ctx *cli.Context) (*config, error) {
	config := &config{
		outputDir:                      ctx.String("output-dir"),
		clobber:                        ctx.Bool("clobber"),
		format:                         outputFormat(ctx.String("output-format")),
		providerVersion:                ctx.String("terraform-provider-version"),
		grafanaURL:                     ctx.String("grafana-url"),
		grafanaAuth:                    ctx.String("grafana-auth"),
		cloudAccessPolicyToken:         ctx.String("cloud-access-policy-token"),
		cloudOrg:                       ctx.String("cloud-org"),
		cloudCreateStackServiceAccount: ctx.Bool("cloud-create-stack-service-account"),
		cloudStackServiceAccountName:   ctx.String("cloud-stack-service-account-name"),
	}

	if config.providerVersion == "" {
		return nil, fmt.Errorf("terraform-provider-version must be set")
	}

	// Validate flags
	err := newFlagValidations().
		atLeastOne("grafana-url", "cloud-access-policy-token").
		conflicting(
			[]string{"grafana-url", "grafana-auth"},
			[]string{"cloud-access-policy-token", "cloud-org", "cloud-create-stack-service-account", "cloud-stack-service-account-name"},
		).
		requiredWhenSet("grafana-url", "grafana-auth").
		requiredWhenSet("cloud-access-policy-token", "cloud-org").
		requiredWhenSet("cloud-stack-service-account-name", "cloud-create-stack-service-account").
		validate(ctx)
	if err != nil {
		return nil, err
	}

	return config, nil
}

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

type outputFormat string

const (
	outputFormatJSON       outputFormat = "json"
	outputFormatHCL        outputFormat = "hcl"
	outputFormatCrossplane outputFormat = "crossplane"
)

var outputFormats = []outputFormat{outputFormatJSON, outputFormatHCL, outputFormatCrossplane}

func main() {
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
			},
			&cli.BoolFlag{
				Name:    "clobber",
				Aliases: []string{"c"},
				Usage:   "Delete all files in the output directory before generating resources",
			},
			&cli.StringFlag{
				Name: "format",
				Usage: fmt.Sprintf("Output format for generated resources. "+
					"Supported formats are: %v", outputFormats),
				Value: string(outputFormatHCL),
			},

			// Grafana OSS flags
			&cli.StringFlag{
				Name:     "grafana-url",
				Usage:    "URL of the Grafana instance to generate resources from",
				Category: "Grafana",
			},
			&cli.StringFlag{
				Name:     "grafana-auth",
				Usage:    "Service account token or username:password for the Grafana instance",
				Category: "Grafana",
			},

			// Grafana Cloud flags
			&cli.StringFlag{
				Name:     "grafana-cloud-access-policy-token",
				Usage:    "Access policy token for Grafana Cloud",
				Category: "Grafana Cloud",
			},
			&cli.StringFlag{
				Name:     "grafana-cloud-org",
				Usage:    "Organization ID or name for Grafana Cloud",
				Category: "Grafana Cloud",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(ctx *cli.Context) error {
	clobber := ctx.Bool("clobber")
	outputDir := ctx.String("output-dir")
	if _, err := os.Stat(outputDir); err == nil && clobber {
		log.Printf("Deleting all files in %s", outputDir)
		if err := os.RemoveAll(outputDir); err != nil {
			return cli.Exit(fmt.Sprintf("Failed to delete %s: %s", outputDir, err), 1)
		}
	} else if err == nil && !clobber {
		return cli.Exit(fmt.Sprintf("Output dir %q already exists. Use --clobber to delete it", outputDir), 1)
	}

	log.Printf("Generating resources to %s", outputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return cli.Exit(fmt.Sprintf("Failed to create output directory %s: %s", outputDir, err), 1)
	}

	grafanaAuth := ctx.String("grafana-auth")
	cloudAccessPolicyToken := ctx.String("grafana-cloud-access-policy-token")
	switch {
	case grafanaAuth != "" && cloudAccessPolicyToken != "":
		return cli.Exit("Cannot specify both Grafana and Grafana Cloud credentials", 1)
	case cloudAccessPolicyToken != "":
		org := ctx.String("grafana-cloud-org")
		if org == "" {
			return cli.Exit("Must specify --grafana-cloud-org when using Grafana Cloud credentials", 1)
		}
		return generateCloudResources(cloudAccessPolicyToken, org)
	case grafanaAuth != "":
		url := ctx.String("grafana-url")
		if url == "" {
			return cli.Exit("Must specify --grafana-url when using Grafana credentials", 1)
		}
		return generateGrafanaResources(url, grafanaAuth)
	default:
		return cli.Exit("Must specify either Grafana or Grafana Cloud credentials", 1)
	}
}

// Command teststack provisions and tears down ephemeral Grafana Cloud stacks
// for use by the cloud-instance acceptance test matrix in CI.
//
// Subcommands:
//
//	up      - Create a stack, install requested features, emit env vars (KEY=VALUE)
//	down    - Delete a stack by slug
//	cleanup - Delete stacks matching a prefix that are older than --age
//
// The output of "up" is a key=value file (stdout or --output path) suitable for
// piping into $GITHUB_ENV.
//
// Required environment variables for all subcommands:
//
//	GRAFANA_CLOUD_ACCESS_POLICY_TOKEN - org-level CAP token with stacks:read,write,
//	                                    stack-service-accounts:write, subscriptions:read,
//	                                    orgs:read, metrics:write, logs:write, traces:write
//	GRAFANA_CLOUD_ORG                 - org slug, used when listing/creating stacks
//
// Optional environment variables:
//
//	GRAFANA_CLOUD_API_URL             - gcom base URL (default https://grafana.com)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "up":
		err = cmdUp(ctx, args)
	case "down":
		err = cmdDown(ctx, args)
	case "cleanup":
		err = cmdCleanup(ctx, args)
	case "-h", "--help", "help":
		cancel()
		usage()
		return
	default:
		cancel()
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", cmd)
		usage()
		os.Exit(2)
	}

	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `teststack: provision ephemeral Grafana Cloud stacks for CI

Usage:
  teststack up      [flags]   Create a stack, install features, emit env vars
  teststack down    [flags]   Delete a stack
  teststack cleanup [flags]   Delete leaked CI stacks older than --age

Run "teststack <subcommand> -h" for flags.

Required env:
  GRAFANA_CLOUD_ACCESS_POLICY_TOKEN
  GRAFANA_CLOUD_ORG

Optional env:
  GRAFANA_CLOUD_API_URL (default https://grafana.com)
`)
}

// Shared flags helper used by subcommands.
type commonFlags struct {
	region   string
	cloudAPI string
}

func registerCommon(fs *flag.FlagSet) *commonFlags {
	c := &commonFlags{}
	fs.StringVar(&c.region, "region", "us", "Grafana Cloud region slug for new stacks. The shared long-lived test stack lives in us-central-0; per-region features like Fleet Management OTel support follow.")
	fs.StringVar(&c.cloudAPI, "cloud-api-url", envOr("GRAFANA_CLOUD_API_URL", "https://grafana.com"), "gcom API base URL")
	return c
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("environment variable %s is required", key)
	}
	return v, nil
}

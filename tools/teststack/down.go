package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func cmdDown(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("down", flag.ContinueOnError)
	common := registerCommon(fs)

	var slug string
	fs.StringVar(&slug, "slug", "", "Stack slug to delete (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if slug == "" {
		return fmt.Errorf("--slug is required")
	}

	capToken, err := mustEnv("GRAFANA_CLOUD_ACCESS_POLICY_TOKEN")
	if err != nil {
		return err
	}

	client, err := newGcomClient(common.cloudAPI, capToken)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "teststack down: deleting stack %q\n", slug)
	if err := deleteStack(ctx, client, slug); err != nil {
		return err
	}
	return nil
}

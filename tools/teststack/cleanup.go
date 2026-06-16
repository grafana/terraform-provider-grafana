package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

func cmdCleanup(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	common := registerCommon(fs)

	var (
		prefix   string
		age      time.Duration
		dryRun   bool
		maxItems int
	)
	fs.StringVar(&prefix, "prefix", "tftest", "Only consider stacks whose slug starts with this prefix")
	fs.DurationVar(&age, "age", 2*time.Hour, "Delete stacks older than this duration")
	fs.BoolVar(&dryRun, "dry-run", false, "List candidates without deleting")
	fs.IntVar(&maxItems, "max", 0, "If > 0, cap the number of deletions per invocation (safety)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	capToken, err := mustEnv("GRAFANA_CLOUD_ACCESS_POLICY_TOKEN")
	if err != nil {
		return err
	}

	client, err := newGcomClient(common.cloudAPI, capToken)
	if err != nil {
		return err
	}

	stacks, err := listStacksByPrefix(ctx, client, prefix)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-age)
	candidates := make([]string, 0, len(stacks))
	for _, s := range stacks {
		// CreatedAt is RFC3339 (validated against gcom OpenAPI).
		created, parseErr := time.Parse(time.RFC3339, s.CreatedAt)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "teststack cleanup: skipping %q with unparseable createdAt=%q: %v\n", s.Slug, s.CreatedAt, parseErr)
			continue
		}
		if created.Before(cutoff) {
			candidates = append(candidates, s.Slug)
		}
	}

	if len(candidates) == 0 {
		fmt.Fprintf(os.Stderr, "teststack cleanup: no stacks older than %s with prefix %q\n", age, prefix)
		return nil
	}

	if maxItems > 0 && len(candidates) > maxItems {
		fmt.Fprintf(os.Stderr, "teststack cleanup: capping deletions to %d (found %d candidates)\n", maxItems, len(candidates))
		candidates = candidates[:maxItems]
	}

	fmt.Fprintf(os.Stderr, "teststack cleanup: %d candidate(s) to delete (dry-run=%v)\n", len(candidates), dryRun)
	var errs []error
	for _, slug := range candidates {
		fmt.Fprintf(os.Stderr, "  - %s\n", slug)
		if dryRun {
			continue
		}
		if err := deleteStack(ctx, client, slug); err != nil {
			errs = append(errs, fmt.Errorf("delete %q: %w", slug, err))
			fmt.Fprintf(os.Stderr, "    delete failed: %v\n", err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("cleanup completed with %d failure(s); first: %w", len(errs), errs[0])
	}
	return nil
}

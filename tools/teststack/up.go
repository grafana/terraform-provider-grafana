package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

func cmdUp(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("up", flag.ContinueOnError)
	common := registerCommon(fs)

	var (
		prefix       string
		features     string
		outputPath   string
		smPubKey     string
		envFile      string
		jsonExport   bool
		readyTimeout time.Duration
	)
	fs.StringVar(&prefix, "prefix", "", "Slug for the new stack; also used as the unique identifier (required)")
	fs.StringVar(&features, "features", "basic", "Comma-separated features to install on the stack")
	fs.StringVar(&outputPath, "output", "", "Path to write KEY=VALUE env vars to (defaults to stdout)")
	fs.StringVar(&smPubKey, "sm-publisher-token", os.Getenv("GRAFANA_CLOUD_ACCESS_POLICY_TOKEN"), "CAP token with metrics/logs/traces:write used for SM install (default: $GRAFANA_CLOUD_ACCESS_POLICY_TOKEN)")
	fs.StringVar(&envFile, "github-env-mask", os.Getenv("GITHUB_ENV"), "If set, mark sensitive values as masked in GitHub Actions before writing")
	fs.BoolVar(&jsonExport, "json", false, "Output JSON instead of KEY=VALUE")
	fs.DurationVar(&readyTimeout, "ready-timeout", 10*time.Minute, "How long to wait for the stack to be active and healthy")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if prefix == "" {
		return fmt.Errorf("--prefix is required")
	}

	capToken, err := mustEnv("GRAFANA_CLOUD_ACCESS_POLICY_TOKEN")
	if err != nil {
		return err
	}

	featureSet, err := parseFeatures(features)
	if err != nil {
		return err
	}

	client, err := newGcomClient(common.cloudAPI, capToken)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "teststack up: creating stack slug=%s region=%s features=%s\n", prefix, common.region, sortedList(featureSet))

	stackCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	info, err := createStack(stackCtx, client, prefix, common.region)
	if err != nil {
		// Best-effort cleanup so a partially-created stack doesn't leak.
		if info != nil {
			_ = deleteStack(context.Background(), client, info.Slug)
		}
		return err
	}

	// Roll back the stack if any subsequent step fails so we never leak a
	// partially-configured stack on the way out.
	var rollback bool
	defer func() {
		if rollback {
			fmt.Fprintf(os.Stderr, "teststack up: rolling back stack %q after failure\n", info.Slug)
			_ = deleteStack(context.Background(), client, info.Slug)
		}
	}()

	if err := waitStackHealthy(stackCtx, info, 5*time.Minute); err != nil {
		rollback = true
		return fmt.Errorf("stack health: %w", err)
	}

	saID, saToken, err := createAdminSA(stackCtx, client, info.Slug, "teststack-admin")
	if err != nil {
		rollback = true
		return err
	}
	info.AdminSAID = saID
	info.AdminSAToken = saToken

	out := map[string]string{
		"GRAFANA_URL":      info.URL,
		"GRAFANA_AUTH":     info.AdminSAToken,
		"GRAFANA_STACK_ID": fmt.Sprintf("%d", info.ID),

		// CheckCloudInstanceTestsEnabled requires these to be non-empty even
		// when not actually used. Fill with placeholders by default; feature
		// installers below overwrite the real ones.
		"GRAFANA_K6_ACCESS_TOKEN":             "skipped-by-shard",
		"GRAFANA_SM_ACCESS_TOKEN":             "skipped-by-shard",
		"GRAFANA_ONCALL_ACCESS_TOKEN":         info.AdminSAToken,
		"GRAFANA_FLEET_MANAGEMENT_AUTH":       "skipped-by-shard",
		"GRAFANA_FLEET_MANAGEMENT_URL":        "https://skipped.example",
		"GRAFANA_CLOUD_PROVIDER_URL":          "https://skipped.example",
		"GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN": "skipped-by-shard",

		// teststack-internal: used by the matching `down` invocation in the
		// teardown step so the workflow doesn't have to track the slug.
		"TESTSTACK_SLUG": info.Slug,
	}

	if featureSet[featureK6] {
		fmt.Fprintf(os.Stderr, "teststack up: installing k6\n")
		k6Token, k6URL, err := installK6(stackCtx, capToken, info, common.cloudAPI)
		if err != nil {
			rollback = true
			return err
		}
		out["GRAFANA_K6_ACCESS_TOKEN"] = k6Token
		out["GRAFANA_K6_URL"] = k6URL
	}

	if featureSet[featureSM] {
		fmt.Fprintf(os.Stderr, "teststack up: installing synthetic monitoring\n")
		if smPubKey == "" {
			rollback = true
			return fmt.Errorf("sm feature requested but no SM publisher token available")
		}
		smToken, smURL, err := installSM(stackCtx, smPubKey, info)
		if err != nil {
			rollback = true
			return err
		}
		out["GRAFANA_SM_ACCESS_TOKEN"] = smToken
		out["GRAFANA_SM_URL"] = smURL
	}

	if featureSet[featureFleet] {
		fmt.Fprintf(os.Stderr, "teststack up: configuring fleet management\n")
		fleetAuth, fleetURL, err := installFleet(stackCtx, client, info)
		if err != nil {
			rollback = true
			return err
		}
		out["GRAFANA_FLEET_MANAGEMENT_AUTH"] = fleetAuth
		out["GRAFANA_FLEET_MANAGEMENT_URL"] = fleetURL
	}

	if featureSet[featureIntegrations] {
		if err := installCloudIntegrations(stackCtx, info); err != nil {
			rollback = true
			return err
		}
	}

	// featureAssertions, featureOncall, featureMLOSS, featureSLO are all
	// available by default on every Grafana Cloud stack and need no further
	// provisioning. The relevant tokens are derived from GRAFANA_AUTH.

	return writeOutput(out, outputPath, jsonExport)
}

// sortedList returns a deterministic, comma-separated rendering of the set's
// keys for logging.
func sortedList(set map[string]bool) string {
	keys := make([]string, 0, len(set))
	for k, v := range set {
		if v {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func writeOutput(out map[string]string, path string, jsonOut bool) error {
	var w io.Writer = os.Stdout
	if path != "" {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	if jsonOut {
		return writeJSON(w, out)
	}
	return writeKV(w, out)
}

func writeKV(w io.Writer, out map[string]string) error {
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// Defensive: env values can't contain newlines, but if they do, mask
		// them so $GITHUB_ENV doesn't get corrupted.
		v := strings.ReplaceAll(out[k], "\n", "\\n")
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w io.Writer, out map[string]string) error {
	return jsonEncode(w, out)
}

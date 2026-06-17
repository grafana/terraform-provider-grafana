package main

import (
	"fmt"
	"strings"
)

// Known feature identifiers accepted by --features. Each non-basic feature
// has its own file in this package mirroring the corresponding
// internal/resources/<name>/ package, so each Grafana team can own the
// teststack setup for their own product via .github/CODEOWNERS.
//
// To add a new feature:
//  1. Add a const below and include it in the parseFeatures switch.
//  2. Create tools/teststack/<feature>.go with the install/setup function.
//  3. Wire the install function into up.go.
//  4. Add a CODEOWNERS entry in .github/CODEOWNERS.in.
const (
	featureBasic        = "basic"
	featureK6           = "k6"
	featureSM           = "sm"
	featureOncall       = "oncall"
	featureFleet        = "fleet"
	featureAssertions   = "assertions"
	featureMLOSS        = "mloss"
	featureSLO          = "slo"
	featureIntegrations = "integrations"
)

// parseFeatures splits a comma-separated list and returns a set.
func parseFeatures(spec string) (map[string]bool, error) {
	out := map[string]bool{featureBasic: true}
	for _, raw := range strings.Split(spec, ",") {
		f := strings.TrimSpace(raw)
		if f == "" {
			continue
		}
		switch f {
		case featureBasic, featureK6, featureSM, featureOncall, featureFleet,
			featureAssertions, featureMLOSS, featureSLO, featureIntegrations:
			out[f] = true
		default:
			return nil, fmt.Errorf("unknown feature %q (allowed: basic,k6,sm,oncall,fleet,assertions,mloss,slo,integrations)", f)
		}
	}
	return out, nil
}

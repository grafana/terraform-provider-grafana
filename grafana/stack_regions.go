package grafana

import (
	"fmt"
	"strings"
)

var (
	stackRegions = StackRegions{
		"us":                  "GCP US Central",
		"us-azure":            "Azure US Central",
		"eu":                  "GCP Belgium",
		"au":                  "GCP Australia",
		"prod-ap-southeast-0": "GCP Singapore",
		"prod-gb-south-0":     "GCP UK",
		"prod-eu-west-3":      "Azure Netherlands",
		"prod-ap-south-0":     "GCP India",
		"prod-sa-east-0":      "GCP Brazil",
	}
)

type StackRegions map[string]string

// Slugs returns the list of all available stack regions.
func (sr StackRegions) Slugs() []string {
	slugs := make([]string, 0)
	for slug := range sr {
		slugs = append(slugs, slug)
	}
	return slugs
}

// DescriptionOptions returns a human-friendly string containing the list of
// the region slugs and names.
func (sr StackRegions) DescriptionOptions() string {
	options := make([]string, 0)
	for slug, name := range sr {
		options = append(options, fmt.Sprintf("%s (%s)", slug, name))
	}
	return strings.Join(options, ", ")
}

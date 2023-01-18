package grafana

import (
	"fmt"
	"strings"
)

var (
	stackRegions = stackRegionList{
		{slug: "us", name: "GCP US Central"},
		{slug: "us-azure", name: "Azure US Central"},
		{slug: "eu", name: "GCP Belgium"},
		{slug: "au", name: "GCP Australia"},
		{slug: "prod-ap-southeast-0", name: "GCP Singapore"},
		{slug: "prod-gb-south-0", name: "GCP UK"},
		{slug: "prod-eu-west-3", name: "Azure Netherlands"},
		{slug: "prod-ap-south-0", name: "GCP India"},
		{slug: "prod-sa-east-0", name: "GCP Brazil"},
	}
)

type stackRegion struct {
	slug string
	name string
}
type stackRegionList []stackRegion

// slugs returns the list of all available stack regions.
func (l stackRegionList) slugs() []string {
	slugs := make([]string, 0)
	for _, region := range l {
		slugs = append(slugs, region.slug)
	}
	return slugs
}

// descriptionOptions returns a human-friendly string containing the list of
// the region slugs and names.
func (l stackRegionList) descriptionOptions() string {
	options := make([]string, 0)
	for _, region := range l {
		options = append(options, fmt.Sprintf("%s (%s)", region.slug, region.name))
	}
	return strings.Join(options, ", ")
}

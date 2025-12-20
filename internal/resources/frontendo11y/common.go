package frontendo11y

import (
	"fmt"
	"time"
)

// TODO: make faroEndpointUrl visible in gcom response
var faroEndpointUrlsRegionExceptions = map[string]string{
	"au":       "https://faro-api-prod-au-southeast-0.grafana.net/faro",
	"eu":       "https://faro-api-prod-eu-west-0.grafana.net/faro",
	"us-azure": "https://faro-api-prod-us-central-7.grafana.net/faro",
	"us":       "https://faro-api-prod-us-central-0.grafana.net/faro",
}

type faroEndpointUrlsRegionCutoff struct {
	cutoffDate      time.Time
	faroEndpointURL string // URL to use after the cutoff date
}

var faroEndpointUrlsAfterCutoff = map[string]faroEndpointUrlsRegionCutoff{
	"prod-us-east-0": {
		cutoffDate:      time.Date(2024, 12, 18, 0, 0, 0, 0, time.UTC),
		faroEndpointURL: "https://faro-api-prod-us-east-2.grafana.net/faro",
	},
}

// getFrontendO11yAPIURLForRegion gets the frontend o11y API URL given a region slug and a created_at date
func getFrontendO11yAPIURLForRegion(regionSlug string, createdAt time.Time) string {
	if cutoffInfo, ok := faroEndpointUrlsAfterCutoff[regionSlug]; ok {
		// if createdAt is after the cutoffInfo date then we have to use the new region endpoint
		if createdAt.After(cutoffInfo.cutoffDate) {
			return cutoffInfo.faroEndpointURL
		}
	}

	if url, ok := faroEndpointUrlsRegionExceptions[regionSlug]; ok {
		return url
	}

	return fmt.Sprintf("https://faro-api-%s.grafana.net/faro", regionSlug)
}

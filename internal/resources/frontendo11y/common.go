package frontendo11y

import "fmt"

// TODO: make faroEndpointUrl visible in gcom response
var faroEndpointUrlsRegionExceptions = map[string]string{
	"au":             "https://faro-api-prod-au-southeast-0.grafana.net/faro",
	"eu":             "https://faro-api-prod-eu-west-0.grafana.net/faro",
	"prod-us-east-0": "https://faro-api-prod-us-east-2.grafana.net/faro",
	"us-azure":       "https://faro-api-prod-us-central-7.grafana.net/faro",
	"us":             "https://faro-api-prod-us-central-0.grafana.net/faro",
}

// getFrontendO11yAPIURLForRegion gets the frontend o11y API URL given a region slug
func getFrontendO11yAPIURLForRegion(regionSlug string) string {
	if url, ok := faroEndpointUrlsRegionExceptions[regionSlug]; ok {
		return url
	}

	return fmt.Sprintf("https://faro-api-%s.grafana.net/faro", regionSlug)
}

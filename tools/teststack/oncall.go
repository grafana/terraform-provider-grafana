package main

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

// getOnCallURL returns the per-stack OnCall API URL from gcom. OnCall is
// a regional service: the URL host varies by region (e.g.
// https://oncall-prod-eu-west-0.grafana.net), so we can't hard-code it.
// The provider requires `oncall_url` to be set before it builds the OnCall
// client, so the caller exports this value as GRAFANA_ONCALL_URL.
//
// The lookup goes through the gcom InstancesAPI which needs the org-level
// CAP token, hence living in tools/teststack. The OnCall test package
// could technically discover the URL itself if it had CAP access, but
// today only the bootstrap step has that.
func getOnCallURL(ctx context.Context, client *gcom.APIClient, info *stackInfo) (string, error) {
	conn, _, err := client.InstancesAPI.GetConnections(ctx, info.Slug).Execute()
	if err != nil {
		return "", fmt.Errorf("get instance connections for OnCall URL: %w", gcomErr(err))
	}
	if conn.OncallApiUrl.IsSet() {
		if v := conn.OncallApiUrl.Get(); v != nil && *v != "" {
			return *v, nil
		}
	}
	return "", fmt.Errorf("stack %q has no OnCall API URL configured", info.Slug)
}

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

// installFleet returns basic-auth credentials and a URL for Grafana Fleet
// Management. Fleet management is auto-provisioned with a stack; the auth
// format per the provider docs is `{fleet_management_user_id}:{cap_token}`
// (see docs/index.md). The user ID comes from the gcom "agent management"
// fields (legacy name for the same backend); the password is the org-level
// Cloud Access Policy token that authenticates the caller.
//
// The auth string uses the CAP token, which is org-scoped and not available
// to individual tests, so this lives in tools/teststack rather than the
// fleetmanagement test package.
func installFleet(ctx context.Context, client *gcom.APIClient, capToken string, info *stackInfo) (auth, apiURL string, err error) {
	stack, _, err := client.InstancesAPI.GetInstance(ctx, info.Slug).Execute()
	if err != nil {
		return "", "", fmt.Errorf("get instance for fleet management URL: %w", gcomErr(err))
	}

	apiURL = strings.TrimSpace(stack.AgentManagementInstanceUrl)
	if apiURL == "" {
		return "", "", fmt.Errorf("stack %q has no fleet management URL configured", info.Slug)
	}

	fleetUserID := stack.AgentManagementInstanceId
	if fleetUserID == 0 {
		return "", "", fmt.Errorf("stack %q has no fleet management user id configured", info.Slug)
	}

	auth = fmt.Sprintf("%d:%s", int64(fleetUserID), capToken)
	return auth, apiURL, nil
}

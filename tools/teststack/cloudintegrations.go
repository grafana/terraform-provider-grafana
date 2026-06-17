package main

import "context"

// installCloudIntegrations is a no-op stub: cloud integrations are enabled
// per-stack via the easystart plugin. Installation happens transparently
// when the tests call the integrations API, so no upfront provisioning is
// needed.
//
// This file exists so the cloud-integrations team owns the teststack code
// for their feature via CODEOWNERS, even though the implementation is
// currently empty.
func installCloudIntegrations(ctx context.Context, info *stackInfo) error {
	_ = ctx
	_ = info
	return nil
}

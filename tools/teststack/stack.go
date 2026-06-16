package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

// stackInfo carries everything we need about a freshly provisioned stack.
type stackInfo struct {
	ID         int64
	Slug       string
	URL        string
	RegionSlug string

	// HmInstancePromID and HlInstanceID are needed by some feature installers
	// (e.g. SM install). Filled in after creation.
	HmInstancePromID int64
	HlInstanceID     int64

	// Admin SA + token created on the stack for use as GRAFANA_AUTH.
	AdminSAID    int64
	AdminSAToken string
}

// createStack creates a Grafana Cloud stack via gcom and waits for it to
// become active. The slug must be globally unique within the org's region.
//
// The caller is responsible for deleting the stack via deleteStack on
// success or failure; createStack does not roll back automatically.
func createStack(ctx context.Context, client *gcom.APIClient, slug, region string) (*stackInfo, error) {
	labels := map[string]string{
		"managed-by": "teststack",
		"purpose":    "ci",
	}
	req := gcom.StackCreateRequestV1{
		Name:             slug,
		Slug:             slug,
		Region:           region,
		Description:      *gcom.NewNullableString(stringPtr("Ephemeral CI stack provisioned by tools/teststack")),
		Labels:           &labels,
		DeleteProtection: *gcom.NewNullableBool(boolPtr(false)),
	}

	// Use a generous client-side timeout for the create call. gcom can take a
	// while to respond when provisioning regional microservices.
	createCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	resp, _, err := client.StacksAPI.CreateStackV1(createCtx).
		StackCreateRequestV1(req).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("create stack %q in region %q: %w", slug, region, gcomErr(err))
	}

	info := &stackInfo{
		ID:   resp.Id,
		Slug: resp.Slug,
		URL:  resp.Url,
	}

	if err := waitStackActive(ctx, client, info.Slug, 10*time.Minute); err != nil {
		return info, err
	}

	// CreateStackV1 returns the minimal StackV1 shape. We need the richer
	// FormattedApiInstance fields (RegionSlug, HmInstancePromId, HlInstanceId)
	// to drive downstream feature installers, so re-fetch via GetInstance.
	full, _, err := client.InstancesAPI.GetInstance(ctx, info.Slug).Execute()
	if err != nil {
		return info, fmt.Errorf("get instance %q after create: %w", info.Slug, gcomErr(err))
	}
	info.RegionSlug = full.RegionSlug
	info.HmInstancePromID = int64(full.HmInstancePromId)
	info.HlInstanceID = int64(full.HlInstanceId)
	if full.Url != "" {
		info.URL = full.Url
	}

	return info, nil
}

// waitStackActive polls until the stack status is "active" or ctx expires.
func waitStackActive(ctx context.Context, client *gcom.APIClient, slug string, timeout time.Duration) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return pollUntil(waitCtx, 5*time.Second, func(c context.Context) (bool, error) {
		stack, _, err := client.InstancesAPI.GetInstance(c, slug).Execute()
		if err != nil {
			return false, err
		}
		if stack.Status == "active" {
			return true, nil
		}
		return false, fmt.Errorf("stack status=%s", stack.Status)
	})
}

// createAdminSA provisions an Admin service account on the stack with a long-
// lived token (24h) intended to be used as GRAFANA_AUTH for the duration of a
// single CI shard.
func createAdminSA(ctx context.Context, client *gcom.APIClient, slug, name string) (int64, string, error) {
	saReq := gcom.PostInstanceServiceAccountsRequest{
		Name:       name,
		Role:       "Admin",
		IsDisabled: boolPtr(false),
	}
	sa, _, err := client.InstancesAPI.PostInstanceServiceAccounts(ctx, slug).
		PostInstanceServiceAccountsRequest(saReq).
		XRequestId(requestID()).
		Execute()
	if err != nil {
		return 0, "", fmt.Errorf("create admin SA on %q: %w", slug, gcomErr(err))
	}

	tokReq := gcom.PostInstanceServiceAccountTokensRequest{
		Name:          name + "-token",
		SecondsToLive: int32Ptr(int32((24 * time.Hour).Seconds())),
	}
	tok, _, err := client.InstancesAPI.PostInstanceServiceAccountTokens(ctx, slug, strconv.FormatInt(*sa.Id, 10)).
		PostInstanceServiceAccountTokensRequest(tokReq).
		XRequestId(requestID()).
		Execute()
	if err != nil {
		return 0, "", fmt.Errorf("create admin SA token on %q: %w", slug, gcomErr(err))
	}

	return *sa.Id, *tok.Key, nil
}

// deleteStack soft-deletes the stack. Idempotent: returns nil if the stack is
// already gone (404).
func deleteStack(ctx context.Context, client *gcom.APIClient, slug string) error {
	delCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	_, httpResp, err := client.InstancesAPI.DeleteInstance(delCtx, slug).
		XRequestId(requestID()).
		Execute()
	if httpResp != nil && httpResp.StatusCode == 404 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete stack %q: %w", slug, gcomErr(err))
	}
	return nil
}

// listStacksByPrefix returns all instances in the configured org whose slug
// begins with prefix. Used by the cleanup subcommand.
func listStacksByPrefix(ctx context.Context, client *gcom.APIClient, prefix string) ([]gcom.FormattedApiInstance, error) {
	resp, _, err := client.InstancesAPI.GetInstances(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", gcomErr(err))
	}
	var out []gcom.FormattedApiInstance
	for _, item := range resp.Items {
		if strings.HasPrefix(item.Slug, prefix) {
			out = append(out, item)
		}
	}
	return out, nil
}

// gcomErr improves the gcom error message: the typed error often contains a
// JSON body that's more informative than the bare error string.
func gcomErr(err error) error {
	if err == nil {
		return nil
	}
	type bodyError interface {
		Body() []byte
	}
	if be, ok := err.(bodyError); ok && len(be.Body()) > 0 {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(be.Body())))
	}
	return err
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func int32Ptr(i int32) *int32    { return &i }

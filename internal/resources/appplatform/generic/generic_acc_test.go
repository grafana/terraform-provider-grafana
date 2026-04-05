package generic_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const genericResourceName = "grafana_apps_generic_resource.test"

func genericProviderConfig(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf(`
provider "grafana" {
  org_id = %d
}
`, grafanaOrgID(t))
}

func grafanaOrgID(t *testing.T) int64 {
	t.Helper()

	orgIDStr := strings.TrimSpace(os.Getenv("GRAFANA_ORG_ID"))
	if orgIDStr == "" {
		return 1
	}

	orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse GRAFANA_ORG_ID %q: %v", orgIDStr, err)
	}

	return orgID
}

func genericResourceImportIDFunc(resourceName string) terraformresource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		apiVersion, err := stateResourceAttribute(s, resourceName, "manifest.apiVersion")
		if err != nil {
			return "", err
		}

		apiGroup, version, ok := strings.Cut(apiVersion, "/")
		if !ok || strings.TrimSpace(apiGroup) == "" || strings.TrimSpace(version) == "" {
			return "", fmt.Errorf("invalid apiVersion %q for resource %s", apiVersion, resourceName)
		}

		kind, err := stateResourceAttribute(s, resourceName, "manifest.kind")
		if err != nil {
			return "", err
		}

		// The manifest may use either metadata.name or metadata.uid as the identifier.
		name, err := stateResourceAttribute(s, resourceName, "manifest.metadata.name")
		if err != nil {
			name, err = stateResourceAttribute(s, resourceName, "manifest.metadata.uid")
			if err != nil {
				return "", fmt.Errorf("neither manifest.metadata.name nor manifest.metadata.uid found for resource %s", resourceName)
			}
		}

		return fmt.Sprintf("%s/%s/%s/%s", apiGroup, version, kind, name), nil
	}
}

func stateResourceAttribute(s *terraform.State, resourceName, attribute string) (string, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("resource not found in state: %s", resourceName)
	}

	value, ok := rs.Primary.Attributes[attribute]
	if !ok {
		return "", fmt.Errorf("attribute %s not found for resource %s", attribute, resourceName)
	}

	return value, nil
}

type genericLiveGetter[T any] func(context.Context, *common.Client, string) (T, error)
type genericNotFoundFunc func(error) bool

func genericEventually[T any](resourceName string, getter genericLiveGetter[T], check func(T) error) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		uid, err := stateResourceAttribute(s, resourceName, "manifest.metadata.name")
		if err != nil {
			uid, err = stateResourceAttribute(s, resourceName, "manifest.metadata.uid")
			if err != nil {
				return fmt.Errorf("neither manifest.metadata.name nor manifest.metadata.uid found for resource %s", resourceName)
			}
		}

		client := testutils.Provider.Meta().(*common.Client)
		deadline := time.Now().Add(30 * time.Second)
		var lastErr error

		for time.Now().Before(deadline) {
			resource, err := getter(context.Background(), client, uid)
			if err != nil {
				lastErr = err
			} else if check == nil {
				return nil
			} else if err := check(resource); err == nil {
				return nil
			} else {
				lastErr = err
			}

			time.Sleep(1 * time.Second)
		}

		if lastErr == nil {
			lastErr = fmt.Errorf("timed out waiting for %s", resourceName)
		}

		return lastErr
	}
}

func genericCheckDestroy[T any](s *terraform.State, resourceType, resourceLabel string, getter genericLiveGetter[T]) error {
	return genericCheckDestroyWithNotFound(s, resourceType, resourceLabel, getter, apierrors.IsNotFound)
}

func genericCheckDestroyWithNotFound[T any](
	s *terraform.State,
	resourceType string,
	resourceLabel string,
	getter genericLiveGetter[T],
	isNotFound genericNotFoundFunc,
) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != resourceType {
			continue
		}

		uid := r.Primary.Attributes["manifest.metadata.name"]
		if uid == "" {
			uid = r.Primary.Attributes["manifest.metadata.uid"]
		}
		if uid == "" {
			continue
		}

		if err := genericWaitForDestroyWithNotFound(context.Background(), client, uid, resourceLabel, getter, isNotFound); err != nil {
			return err
		}
	}

	return nil
}

func genericWaitForDestroyWithNotFound[T any](
	ctx context.Context,
	client *common.Client,
	uid string,
	resourceLabel string,
	getter genericLiveGetter[T],
	isNotFound genericNotFoundFunc,
) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		if _, err := getter(ctx, client, uid); err == nil {
			lastErr = fmt.Errorf("%s %s still exists", resourceLabel, uid)
			time.Sleep(1 * time.Second)
			continue
		} else if isNotFound(err) {
			return nil
		} else {
			lastErr = fmt.Errorf("error checking %s %s: %w", resourceLabel, uid, err)
			time.Sleep(1 * time.Second)
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for %s %s to be deleted", resourceLabel, uid)
	}

	return lastErr
}

func genericCheckNoStateAttributePrefix(resourceName, prefix string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok || resourceState == nil {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		for key := range resourceState.Primary.Attributes {
			if key == prefix || strings.HasPrefix(key, prefix+".") || strings.HasPrefix(key, prefix+".%") {
				return fmt.Errorf("unexpected state attribute %q for resource %s", key, resourceName)
			}
		}

		return nil
	}
}

func genericConfiguredOrgNamespace(client *common.Client) string {
	orgID := client.GrafanaOrgID
	if orgID <= 0 {
		orgID = 1
	}

	return claims.OrgNamespaceFormatter(orgID)
}

func testAccNamespace(commonClient *common.Client) (string, error) {
	switch {
	case commonClient.GrafanaStackID > 0:
		return claims.CloudNamespaceFormatter(commonClient.GrafanaStackID), nil
	case commonClient.GrafanaOrgID > 0:
		return claims.OrgNamespaceFormatter(commonClient.GrafanaOrgID), nil
	default:
		return "", fmt.Errorf("missing Grafana org or stack ID")
	}
}

func waitForProvisioningAPI(t *testing.T) {
	t.Helper()

	baseURL := strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")
	if baseURL == "" {
		t.Fatal("GRAFANA_URL must be set")
	}

	reqURL := baseURL + "/apis/provisioning.grafana.app/v0alpha1/namespaces/" + claims.OrgNamespaceFormatter(grafanaOrgID(t)) + "/repositories"
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(2 * time.Minute)
	start := time.Now()
	nextLog := 10 * time.Second
	lastResult := "no response yet"

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
		if err != nil {
			t.Fatalf("failed to create provisioning readiness request: %v", err)
		}

		setGrafanaAuth(req)

		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastResult = fmt.Sprintf("status %d", resp.StatusCode)
		} else {
			lastResult = err.Error()
		}

		if elapsed := time.Since(start); elapsed >= nextLog {
			t.Logf("waiting for provisioning API at %s (%s elapsed, last result: %s)", reqURL, elapsed.Round(time.Second), lastResult)
			nextLog += 10 * time.Second
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf("timed out waiting for provisioning API at %s (last result: %s)", reqURL, lastResult)
}

func setGrafanaAuth(req *http.Request) {
	auth := os.Getenv("GRAFANA_AUTH")
	if auth == "" {
		return
	}

	if username, password, ok := strings.Cut(auth, ":"); ok {
		req.SetBasicAuth(username, password)
		return
	}

	req.Header.Set("Authorization", "Bearer "+auth)
}

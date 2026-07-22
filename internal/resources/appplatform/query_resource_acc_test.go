package appplatform_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	queryResourceType = "grafana_apps_queries_query_v1"
	queryResourceName = queryResourceType + ".test"

	// queriesAPIGroupVersion is the API group/version probed to decide whether
	// the Query Library API is available on the target instance.
	queriesAPIGroupVersion = "queries.grafana.app/v1"
)

func TestAccQuery(t *testing.T) {
	// Query Library is a Grafana Enterprise feature. Gate on enterprise acc tests,
	// then self-skip if the queries.grafana.app API group isn't served on the
	// target instance (e.g. the queryLibrary feature toggle is off). This way the
	// test runs wherever the API is available and skips cleanly otherwise, instead
	// of being unconditionally disabled.
	testutils.CheckEnterpriseTestsEnabled(t)
	skipIfQueryLibraryUnavailable(t)

	t.Run("basic", func(t *testing.T) {
		uid := fmt.Sprintf("test-query-%s", acctest.RandString(6))

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckQueryDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccQueryConfig(uid, "Requests per second", []string{"http", "prometheus", "provisioned"}),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(queryResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.title", "Requests per second"),
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.tags.#", "3"),
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.targets.#", "1"),
						// properties_json is stored in canonical jsonencode() form
						// (compact, keys sorted). This guards the canonicalization
						// that keeps freeform JSON diff-free.
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.targets.0.properties_json", testAccQueryCanonicalProperties),
						terraformresource.TestCheckResourceAttrSet(queryResourceName, "id"),
					),
				},
				{
					// Import round-trip: verifies SpecSaver reconstructs the spec
					// to match the applied state. Regression guard for the
					// import diffs we fixed — properties_json key ordering,
					// variables_json "null", and is_locked false->null.
					ResourceName:      queryResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					// options.overwrite is stamped by the framework on import and
					// isn't part of config; siblings ignore it too.
					ImportStateVerifyIgnore: []string{
						"options.%",
						"options.overwrite",
					},
					ImportStateIdFunc: importStateIDFunc(queryResourceName),
				},
			},
		})
	})

	t.Run("update", func(t *testing.T) {
		uid := fmt.Sprintf("test-query-%s", acctest.RandString(6))

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckQueryDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccQueryConfig(uid, "Requests per second", []string{"http"}),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.title", "Requests per second"),
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.tags.#", "1"),
					),
				},
				{
					Config: testAccQueryConfig(uid, "Requests per minute", []string{"http", "prometheus", "sli"}),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.title", "Requests per minute"),
						terraformresource.TestCheckResourceAttr(queryResourceName, "spec.tags.#", "3"),
					),
				},
			},
		})
	})
}

// skipIfQueryLibraryUnavailable skips the test unless the queries.grafana.app/v1
// API group is served by the target Grafana instance (i.e. the queryLibrary
// feature is enabled). Relies on GRAFANA_URL/GRAFANA_AUTH, which
// CheckEnterpriseTestsEnabled has already verified are set.
func skipIfQueryLibraryUnavailable(t *testing.T) {
	t.Helper()

	base := strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")
	//nolint:gosec // G704: URL is built from the trusted GRAFANA_URL test env var, not user input.
	req, err := http.NewRequest(http.MethodGet, base+"/apis/"+queriesAPIGroupVersion, nil)
	if err != nil {
		t.Fatalf("building query library capability check: %s", err)
	}
	if user, pass, ok := strings.Cut(os.Getenv("GRAFANA_AUTH"), ":"); ok {
		req.SetBasicAuth(user, pass)
	} else {
		req.Header.Set("Authorization", "Bearer "+os.Getenv("GRAFANA_AUTH"))
	}

	//nolint:gosec // G704: request targets the trusted GRAFANA_URL test env var, not user input.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("checking query library availability at %s: %s", req.URL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// API group is served — run the test.
	case http.StatusNotFound:
		t.Skipf("queries.grafana.app API not available (HTTP 404); enable the queryLibrary feature toggle to run this test")
	default:
		// Any other status (401/403/5xx, ...) is a real problem, not "feature
		// off" — fail loudly instead of masking it as a skip.
		t.Fatalf("unexpected status probing %s: HTTP %d", req.URL, resp.StatusCode)
	}
}

func testAccCheckQueryDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != queryResourceType {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(appplatform.QueryKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := resource.NewNamespaced(
			resource.NewTypedClient[*appplatform.Query, *appplatform.QueryList](rcli, appplatform.QueryKind()),
			ns,
		)

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("Query %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if Query %s exists: %w", uid, err)
		}
	}
	return nil
}

// testAccQueryCanonicalProperties is the canonical (compact, keys sorted) form
// of the target's properties_json used in testAccQueryConfig — i.e. what
// jsonencode() produces and what the provider stores in state.
const testAccQueryCanonicalProperties = `{"expr":"rate(http_requests_total[$__rate_interval])","refId":"A"}`

func testAccQueryConfig(uid, title string, tags []string) string {
	tagsHCL := ""
	for i, tag := range tags {
		if i > 0 {
			tagsHCL += ", "
		}
		tagsHCL += fmt.Sprintf("%q", tag)
	}

	return fmt.Sprintf(`
resource "grafana_apps_queries_query_v1" "test" {
  metadata {
    uid = %q
  }

  options {
    overwrite = true
  }

  spec {
    title = %q
    tags  = [%s]

    targets {
      properties_json = jsonencode({
        refId = "A"
        expr  = "rate(http_requests_total[$__rate_interval])"
      })
    }
  }
}
`, uid, title, tagsHCL)
}

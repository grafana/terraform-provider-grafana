package grafana_test

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	defaultOrgIDRegexp = regexp.MustCompile(`^(0|1):[a-zA-Z0-9-_]+$`)
	// https://regex101.com/r/icTmfm/1
	nonDefaultOrgIDRegexp = regexp.MustCompile(`^([^0-1]\d*|1\d+):[a-zA-Z0-9-_]+$`)

	// providerConfigMu serializes tests that set an explicit provider config (token or basic auth)
	// so they do not run concurrently and see each other's provider config (SDK may cache/reuse server).
	// When running only these tests locally, use -parallel 1 so they run sequentially and neither blocks on the lock.
	providerConfigMu sync.Mutex
)

// #region agent log
func debugLog(loc, msg, hyp string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["sessionId"] = "4ea19e"
	f, err := os.OpenFile("/Users/arati/code/terraform-provider-grafana/.cursor/debug-4ea19e.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	rec := map[string]interface{}{"sessionId": "4ea19e", "timestamp": time.Now().UnixMilli(), "location": loc, "message": msg, "data": data, "hypothesisId": hyp}
	enc := json.NewEncoder(f)
	enc.Encode(rec)
}

// #endregion

func checkResourceIsInOrg(resourceName, orgResourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceOrgID, err := strconv.Atoi(s.RootModule().Resources[resourceName].Primary.Attributes["org_id"])
		if err != nil {
			return err
		}

		if resourceOrgID <= 1 {
			return fmt.Errorf("resource org ID %d is not greater than 1", resourceOrgID)
		}

		orgID, err := strconv.Atoi(s.RootModule().Resources[orgResourceName].Primary.ID)
		if err != nil {
			return err
		}

		if orgID != resourceOrgID {
			return fmt.Errorf("expected org ID %d, got %d", orgID, resourceOrgID)
		}

		return nil
	}
}

func grafanaTestClient() *goapi.GrafanaHTTPAPI {
	return testutils.Provider.Meta().(*common.Client).GrafanaAPI.Clone().WithOrgID(0)
}

// orgScopedTest creates a temporary org and service account token. It returns the org ID and
// token so callers can pass them in the Terraform provider config (e.g. via ConfigWithTokenProvider)
// instead of setting GRAFANA_AUTH. That keeps tests isolated: parallel tests no longer overwrite
// process-wide env and each test's provider config is explicit.
func orgScopedTest(t *testing.T) (orgID int64, token string) {
	t.Helper()

	// #region agent log
	t0 := time.Now()
	debugLog("oss_org_id_test.go:orgScopedTest", "orgScopedTest start", "H-A", map[string]interface{}{"elapsed_ms": 0})
	// #endregion

	name := acctest.RandString(10)
	globalClient := grafanaTestClient()
	org, err := globalClient.Orgs.CreateOrg(&models.CreateOrgCommand{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	// #region agent log
	debugLog("oss_org_id_test.go:orgScopedTest", "after CreateOrg", "H-A", map[string]interface{}{"elapsed_ms": time.Since(t0).Milliseconds()})
	// #endregion

	t.Cleanup(func() {
		if _, err := globalClient.Orgs.DeleteOrgByID(*org.Payload.OrgID); err != nil {
			t.Fatal(err)
		}
	})
	orgClient := grafanaTestClient().WithOrgID(*org.Payload.OrgID)
	sa, err := orgClient.ServiceAccounts.CreateServiceAccount(
		service_accounts.NewCreateServiceAccountParams().WithBody(&models.CreateServiceAccountForm{
			Name: name,
			Role: "Admin",
		},
		))
	if err != nil {
		t.Fatal(err)
	}
	// #region agent log
	debugLog("oss_org_id_test.go:orgScopedTest", "after CreateServiceAccount", "H-A", map[string]interface{}{"elapsed_ms": time.Since(t0).Milliseconds()})
	// #endregion

	saToken, err := orgClient.ServiceAccounts.CreateToken(
		service_accounts.NewCreateTokenParams().WithBody(&models.AddServiceAccountTokenCommand{
			Name: name,
		},
		).WithServiceAccountID(sa.Payload.ID),
	)
	if err != nil {
		t.Fatal(err)
	}

	// #region agent log
	debugLog("oss_org_id_test.go:orgScopedTest", "orgScopedTest done", "H-A", map[string]interface{}{"elapsed_ms": time.Since(t0).Milliseconds()})
	// #endregion

	return *org.Payload.OrgID, saToken.Payload.Key
}

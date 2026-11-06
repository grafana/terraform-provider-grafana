package assistant_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// This test invokes the real calibration agent. It creates only dedicated
// fixtures without notification destinations and removes them on completion.
func TestAccAssistantWatcher_unattended(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)
	auth := os.Getenv("GRAFANA_AUTH")
	var basic *url.Userinfo
	if user, password, ok := strings.Cut(auth, ":"); ok {
		basic = url.UserPassword(user, password)
		auth = ""
	}
	client, err := assistantapi.NewClient(os.Getenv("GRAFANA_URL"), basic, auth, nil, "terraform-watcher-acceptance", nil)
	if err != nil {
		t.Fatal(err)
	}
	var queryIDs []string
	var learningEpoch int64
	checkPreserved := func(s *terraform.State) error {
		entry := s.RootModule().Resources["grafana_assistant_watcher.test"]
		watcher, err := client.GetWatcher(context.Background(), entry.Primary.ID)
		if err != nil {
			return err
		}
		if len(watcher.Queries) == 0 {
			return fmt.Errorf("calibration saved no checks")
		}
		ids := make([]string, len(watcher.Queries))
		for i, q := range watcher.Queries {
			ids[i] = q.ID
		}
		if queryIDs == nil {
			queryIDs = ids
			learningEpoch = watcher.LearningEpoch
		} else if fmt.Sprint(queryIDs) != fmt.Sprint(ids) {
			return fmt.Errorf("query identities changed: %v -> %v", queryIDs, ids)
		}
		if learningEpoch != watcher.LearningEpoch {
			return fmt.Errorf("learning epoch changed from %d to %d", learningEpoch, watcher.LearningEpoch)
		}
		return nil
	}
	config := func(name string, started bool) string {
		return fmt.Sprintf(`
 resource "grafana_assistant_watcher" "test" {
 name = %q
 prompt = "Monitor Grafana-managed firing alerts. Validate an alerts check for current firing alerts. No notifications or investigations. An empty current result is an acceptable baseline."
 trigger_interval_seconds = 600
 labels = { purpose = "terraform-acceptance" }
 }
 resource "grafana_assistant_watcher_calibration" "test" {
 watcher_id = grafana_assistant_watcher.test.id
 started = %t
 timeout_seconds = 180
 }`, name, started)
	}
	initial := config("tf-acc-watcher-unattended", false)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: initial, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("grafana_assistant_watcher_calibration.test", "status", "completed"), checkPreserved)},
			{Config: initial, PlanOnly: true},
			{ResourceName: "grafana_assistant_watcher.test", ImportState: true, ImportStateVerify: true},
			{ResourceName: "grafana_assistant_watcher_calibration.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"timeout_seconds"}},
			{Config: config("tf-acc-watcher-renamed", false), Check: checkPreserved},
			{Config: config("tf-acc-watcher-renamed", true), Check: checkPreserved},
			{Config: config("tf-acc-watcher-renamed", false), Check: checkPreserved},
			{Config: config("tf-acc-watcher-renamed", false), PlanOnly: true},
		},
	})
}

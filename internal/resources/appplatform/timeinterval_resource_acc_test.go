package appplatform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	timeIntervalResourceType = "grafana_apps_notifications_timeinterval_v1beta1"
	timeIntervalResourceName = timeIntervalResourceType + ".test"
)

func TestAccTimeInterval(t *testing.T) {
	t.Skip("v1beta1 APIs requires Grafana >=13.0.0-23429090056; enable once a compatible instance is available in CI")
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0-23429090056")

	t.Run("basic", func(t *testing.T) {
		name := fmt.Sprintf("test-time-interval-%s", acctest.RandString(6))

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckTimeIntervalDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccTimeIntervalConfig(name, []testAccTimeInterval{
						{weekdays: []string{"monday", "tuesday"}},
					}),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.name", name),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.#", "1"),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.0.weekdays.#", "2"),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.0.weekdays.0", "monday"),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.0.weekdays.1", "tuesday"),
						terraformresource.TestCheckResourceAttrSet(timeIntervalResourceName, "metadata.uid"),
						terraformresource.TestCheckResourceAttrSet(timeIntervalResourceName, "id"),
					),
				},
				{
					ResourceName:      timeIntervalResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateIdFunc: importStateIDFunc(timeIntervalResourceName),
				},
			},
		})
	})

	t.Run("update", func(t *testing.T) {
		name := fmt.Sprintf("test-time-interval-%s", acctest.RandString(6))

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckTimeIntervalDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccTimeIntervalConfig(name, []testAccTimeInterval{
						{weekdays: []string{"monday"}},
					}),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.#", "1"),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.0.weekdays.#", "1"),
					),
				},
				{
					Config: testAccTimeIntervalConfig(name, []testAccTimeInterval{
						{weekdays: []string{"monday", "wednesday", "friday"}},
						{months: []string{"january", "february"}},
					}),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.#", "2"),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.0.weekdays.#", "3"),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.1.months.#", "2"),
						terraformresource.TestCheckResourceAttr(timeIntervalResourceName, "spec.time_intervals.1.months.0", "january"),
					),
				},
			},
		})
	})
}

func testAccCheckTimeIntervalDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != timeIntervalResourceType {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v1beta1.TimeIntervalKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := sdkresource.NewNamespaced(
			sdkresource.NewTypedClient[*v1beta1.TimeInterval, *v1beta1.TimeIntervalList](rcli, v1beta1.TimeIntervalKind()),
			ns,
		)

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("TimeInterval %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if TimeInterval %s exists: %w", uid, err)
		}
	}
	return nil
}

type testAccTimeInterval struct {
	weekdays    []string
	daysOfMonth []string
	months      []string
	years       []string
	location    string
	times       []testAccTimeRange
}

type testAccTimeRange struct {
	startTime string
	endTime   string
}

func testAccTimeIntervalConfig(name string, intervals []testAccTimeInterval) string {
	intervalsHCL := ""
	for _, iv := range intervals {
		intervalsHCL += "    time_intervals {\n"
		if len(iv.weekdays) > 0 {
			intervalsHCL += fmt.Sprintf("      weekdays = [%s]\n", formatStringList(iv.weekdays))
		}
		if len(iv.daysOfMonth) > 0 {
			intervalsHCL += fmt.Sprintf("      days_of_month = [%s]\n", formatStringList(iv.daysOfMonth))
		}
		if len(iv.months) > 0 {
			intervalsHCL += fmt.Sprintf("      months = [%s]\n", formatStringList(iv.months))
		}
		if len(iv.years) > 0 {
			intervalsHCL += fmt.Sprintf("      years = [%s]\n", formatStringList(iv.years))
		}
		if iv.location != "" {
			intervalsHCL += fmt.Sprintf("      location = %q\n", iv.location)
		}
		for _, tr := range iv.times {
			intervalsHCL += fmt.Sprintf("      times = [{ start_time = %q, end_time = %q }]\n", tr.startTime, tr.endTime)
		}
		intervalsHCL += "    }\n"
	}

	return fmt.Sprintf(`
resource "grafana_apps_notifications_timeinterval_v1beta1" "test" {
  metadata {}

  spec {
    name = %q
%s  }
}
`, name, intervalsHCL)
}

func formatStringList(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%q", s)
	}
	return result
}

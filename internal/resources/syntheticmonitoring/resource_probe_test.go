package syntheticmonitoring_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceProbe(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := acctest.RandomWithPrefix("My Probe")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_probe/resource.tf", map[string]string{
					"Mount Everest": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", randomName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "latitude", "27.986059188842773"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "longitude", "86.92262268066406"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "region", "APAC"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "public", "false"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "labels.type", "mountain"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_probe/resource_update.tf", map[string]string{
					"Mauna Loa": randomName + " Updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", randomName+" Updated"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "latitude", "19.479480743408203"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "longitude", "-155.60281372070312"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "region", "AMER"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "public", "false"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "labels.type", "volcano"),
				),
			},
		},
	})
}

// Test that a probe is recreated if deleted outside the Terraform process
func TestAccResourceProbe_recreate(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := acctest.RandomWithPrefix("My Probe")
	config := testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_probe/resource.tf", map[string]string{
		"Mount Everest": randomName,
	})

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["grafana_synthetic_monitoring_probe.main"]
					id, _ := strconv.ParseInt(rs.Primary.ID, 10, 64)
					return testutils.Provider.Meta().(*common.Client).SMAPI.DeleteProbe(context.Background(), id)
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", randomName),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "latitude", "27.986059188842773"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "longitude", "86.92262268066406"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "region", "APAC"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "public", "false"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "labels.type", "mountain"),
				),
			},
		},
	})
}

// Test that a probe that is used in a check can be deleted or recreated
func TestAccResourceProbe_recreateProbeUsedInCheck(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testSyntheticMonitoringProbeAndCheck(randomName, "test1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.first", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.second", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.first", "name", randomName+"test1-first"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.second", "name", randomName+"test1-second"),
				),
			},
			// Change the name of the probe
			{
				Config: testSyntheticMonitoringProbeAndCheck(randomName, "test2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.first", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.second", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.first", "name", randomName+"test2-first"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.second", "name", randomName+"test2-second"),
				),
			},
			// Taint a single probe and recreate
			{
				Config: testSyntheticMonitoringProbeAndCheck(randomName, "test2"),
				Taint:  []string{"grafana_synthetic_monitoring_probe.first"},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.first", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.second", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.first", "name", randomName+"test2-first"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.second", "name", randomName+"test2-second"),
				),
			},
			// Taint everything and recreate
			{
				Config: testSyntheticMonitoringProbeAndCheck(randomName, "test2"),
				Taint:  []string{"grafana_synthetic_monitoring_probe.first", "grafana_synthetic_monitoring_probe.second", "grafana_synthetic_monitoring_check.http"},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.first", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.second", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check.http", "id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.first", "name", randomName+"test2-first"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.second", "name", randomName+"test2-second"),
				),
			},
		},
	})
}

func TestAccResourceProbe_InvalidLabels(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var steps []resource.TestStep
	for _, tc := range []struct {
		cfg string
		err string
	}{
		{
			cfg: testSyntheticMonitoringProbeLabel("", "any"),
			err: `invalid label "=any": invalid label name`,
		},
		{
			cfg: testSyntheticMonitoringProbeLabel("bad-label", "any"),
			err: `invalid label "bad-label=any": invalid label name`,
		},
		{
			cfg: testSyntheticMonitoringProbeLabel("thisisempty", ""),
			err: `invalid label "thisisempty=": invalid label value`,
		},
	} {
		steps = append(steps, resource.TestStep{
			Config:      tc.cfg,
			ExpectError: regexp.MustCompile(tc.err),
		})
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps:             steps,
	})
}

func testSyntheticMonitoringProbeAndCheck(name, probeSuffix string) string {
	return fmt.Sprintf(`
resource "grafana_synthetic_monitoring_probe" "first" {
	name      = "%[1]s%[2]s-first"
	latitude  = 27.98606
	longitude = 86.92262
	region    = "APAC"
}

resource "grafana_synthetic_monitoring_probe" "second" {
	name      = "%[1]s%[2]s-second"
	latitude  = 26.98606
	longitude = 87.92262
	region    = "APAC"
}

resource "grafana_synthetic_monitoring_check" "http" {
	job     = "%[1]s"
	target  = "https://%[1]s.com"
	enabled = false
	probes = [
	  grafana_synthetic_monitoring_probe.first.id,
	  grafana_synthetic_monitoring_probe.second.id,
	]
	labels = {
	  foo = "bar"
	}
	settings {
	  http {}
	}
  }
  
`, name, probeSuffix)
}

func testSyntheticMonitoringProbeLabel(name, value string) string {
	return fmt.Sprintf(`
resource "grafana_synthetic_monitoring_probe" "main" {
	name      = "Everest"
	latitude  = 27.98606
	longitude = 86.92262
	region    = "APAC"
	labels = {
		"%s" = "%s"
	}
}
`, name, value)
}

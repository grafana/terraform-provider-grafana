package grafana

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceSyntheticMonitoringProbe(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_probe/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", "Mount Everest"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "latitude", "27.986059188842773"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "longitude", "86.92262268066406"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "region", "APAC"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "public", "false"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "labels.type", "mountain"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_probe/resource_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", "Mauna Loa"),
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
func TestAccResourceSyntheticMonitoringProbe_recreate(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_probe/resource.tf"),
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["grafana_synthetic_monitoring_probe.main"]
					id, _ := strconv.ParseInt(rs.Primary.ID, 10, 64)
					return testAccProvider.Meta().(*client).smapi.DeleteProbe(context.Background(), id)
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_probe/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", "Mount Everest"),
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

func TestAccResourceSyntheticMonitoringProbe_InvalidLabels(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	var steps []resource.TestStep
	for _, tc := range []struct {
		cfg string
		err string
	}{
		{
			cfg: testSyntheticMonitoringProbeLabel("", "any"),
			err: `invalid label name ""`,
		},
		{
			cfg: testSyntheticMonitoringProbeLabel("42", "any"),
			err: `invalid label name "42"`,
		},
		{
			cfg: testSyntheticMonitoringProbeLabel("bad-label", "any"),
			err: `invalid label name "bad-label"`,
		},
		{
			cfg: testSyntheticMonitoringProbeLabel("thisisempty", ""),
			err: `label "thisisempty" has an empty value`,
		},
		{
			cfg: testSyntheticMonitoringProbeLabel("name", ","),
			err: `label "name" has an invalid character it its value`,
		},
	} {
		steps = append(steps, resource.TestStep{
			Config:      tc.cfg,
			ExpectError: regexp.MustCompile(tc.err),
		})
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps:             steps,
	})
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

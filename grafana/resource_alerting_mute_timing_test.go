package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccMuteTiming_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">9.0.0")

	var mt gapi.MuteTiming

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testMuteTimingCheckDestroy(&mt),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccExample(t, "resources/grafana_mute_timing/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testMuteTimingCheckExists("grafana_mute_timing.my_mute_timing", &mt),
				),
			},
		},
	})
}

func testMuteTimingCheckExists(rname string, timing *gapi.MuteTiming) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rname]
		if !ok {
			return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		mt, err := client.MuteTiming(resource.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting resource: %w", err)
		}
		*timing = mt
		return nil
	}
}

func testMuteTimingCheckDestroy(timing *gapi.MuteTiming) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		mt, err := client.MuteTiming(timing.Name)
		if err == nil && mt.Name != "" {
			return fmt.Errorf("mute timing still exists on the server")
		}
		return nil
	}
}

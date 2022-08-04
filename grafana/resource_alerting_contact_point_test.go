package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccContactPoint_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=9.0.0")

	var points []gapi.ContactPoint

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testContactPointCheckDestroy(points),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccExample(t, "resources/grafana_contact_point/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", "My Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "custom.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "custom.0.type", "email"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "custom.0.disable_resolve_message", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "custom.0.settings.addresses", "one@company.org;two@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.#", "0"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_contact_point.my_contact_point",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test update content.
			{
				Config: testAccExampleWithReplace(t, "resources/grafana_contact_point/resource.tf", map[string]string{
					"company.org": "user.net",
				}),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "custom.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "custom.0.settings.addresses", "one@user.net;two@user.net"),
				),
			},
			// Test rename.
			{
				Config: testAccExampleWithReplace(t, "resources/grafana_contact_point/resource.tf", map[string]string{
					"My Contact Point": "A Different Contact Point",
				}),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", "A Different Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "custom.#", "1"),
					testContactPointCheckAllDestroy("My Contact Point"),
				),
			},
		},
	})
}

func TestAccContactPoint_compound(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=9.0.0")

	var points []gapi.ContactPoint

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testContactPointCheckDestroy(points),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccExample(t, "resources/grafana_contact_point/_acc_compound_custom_receiver.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_custom_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "name", "Compound Custom Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "custom.#", "2"),
				),
			},
			// Test update.
			{
				Config: testAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_compound_custom_receiver.tf", map[string]string{
					"discord-webhook-url": "another-url",
				}),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_custom_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "custom.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "custom.1.settings.url", "http://another-url"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_contact_point/_acc_compound_custom_receiver_added.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_custom_contact_point", &points, 3),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "custom.#", "3"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "custom.0.settings.addresses", "one@company.org;two@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "custom.2.settings.addresses", "three@company.org;four@company.org"),
				),
			},
			// TODO: Test removal of a point from the compound one.
			// Test rename.
			/*{
				Config: testAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_compound_custom_receiver.tf", map[string]string{
					"Compound Custom Contact Point": "A Different Contact Point",
				}),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_custom_contact_point", &points),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "name", "A Different Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_custom_contact_point", "custom.#", "2"),
					testContactPointCheckAllDestroy("Compound Custom Contact Point"),
				),
			},*/
		},
	})
}

func testContactPointCheckExists(rname string, pts *[]gapi.ContactPoint, expCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rname]
		if !ok {
			return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
		}

		name, ok := resource.Primary.Attributes["name"]
		if !ok {
			return fmt.Errorf("resource name not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		points, err := client.ContactPointsByName(name)
		if err != nil {
			return fmt.Errorf("error getting resource: %w", err)
		}

		if len(points) != expCount {
			return fmt.Errorf("wrong number of contact points on the server, expected %d but got %#v", expCount, points)
		}

		*pts = points
		return nil
	}
}

func testContactPointCheckDestroy(points []gapi.ContactPoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		for _, p := range points {
			_, err := client.ContactPoint(p.UID)
			if err == nil {
				return fmt.Errorf("contact point still exists on the server")
			}
		}
		points = []gapi.ContactPoint{}
		return nil
	}
}

func testContactPointCheckAllDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		points, err := client.ContactPointsByName(name)
		if err != nil {
			return fmt.Errorf("error getting resource: %w", err)
		}
		if len(points) > 0 {
			return fmt.Errorf("contact points still exist on the server: %#v", points)
		}
		return nil
	}
}

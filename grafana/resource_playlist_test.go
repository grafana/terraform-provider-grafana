package grafana

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const paylistResource = "grafana_playlist.test"

func TestAccPlaylist_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					resource.TestMatchResourceAttr(paylistResource, "id", uidRegexp),
					resource.TestCheckResourceAttr(paylistResource, "name", rName),
					resource.TestCheckResourceAttr(paylistResource, "item.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "1",
						"title": "Terraform Dashboard By Tag",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "2",
						"title": "Terraform Dashboard By ID",
					}),
				),
			},
			{
				ResourceName:      paylistResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_update(t *testing.T) {
	CheckOSSTestsEnabled(t)

	rName := acctest.RandomWithPrefix("tf-acc-test")
	updatedName := "updated name"

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
				),
			},
			{
				Config: testAccPlaylistConfigUpdate(updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					resource.TestMatchResourceAttr(paylistResource, "id", uidRegexp),
					resource.TestCheckResourceAttr(paylistResource, "name", updatedName),
					resource.TestCheckResourceAttr(paylistResource, "item.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "1",
						"title": "Terraform Dashboard By ID",
						"type":  "dashboard_by_id",
						"value": "3",
					}),
				),
			},
			{
				ResourceName:      paylistResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_disappears(t *testing.T) {
	CheckOSSTestsEnabled(t)

	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					testAccPlaylistDisappears(),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccPlaylistCheckExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[paylistResource]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", paylistResource, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi

		_, err := client.Playlist(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting playlist: %w", err)
		}

		return nil
	}
}

func testAccPlaylistDisappears() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[paylistResource]
		if !ok {
			return fmt.Errorf("resource not found: %s", paylistResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi

		return client.DeletePlaylist(rs.Primary.ID)
	}
}

func testAccPlaylistDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*client).gapi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_playlist" {
			continue
		}

		playlist, err := client.Playlist(rs.Primary.ID)

		if err != nil {
			if strings.HasPrefix(err.Error(), "status: 404") {
				continue
			}
			return err
		}

		if playlist != nil && playlist.ID != 0 {
			return fmt.Errorf("Playlist still exists: %+v", playlist)
		}
	}

	return nil
}

func testAccPlaylistConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "grafana_playlist" "test" {
	name     = %[1]q
	interval = "5m"

	item {
		order = 1
		title = "Terraform Dashboard By Tag"
	}

	item {
		order = 2
		title = "Terraform Dashboard By ID"
	}
}
`, name)
}

func testAccPlaylistConfigUpdate(name string) string {
	return fmt.Sprintf(`
resource "grafana_playlist" "test" {
	name     = %[1]q
	interval = "5m"
	
	item {
		order = 1
		title = "Terraform Dashboard By ID"
		type = "dashboard_by_id"
		value = "3"
	}
}
`, name)
}

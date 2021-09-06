package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPlaylist_basic(t *testing.T) {
	resourceName := "grafana_playlist.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(resourceName),
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile(`\d+`)),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "item.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "item.*", map[string]string{
						"order": "1",
						"title": "Terraform Dashboard By Tag",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "item.*", map[string]string{
						"order": "2",
						"title": "Terraform Dashboard By ID",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_update(t *testing.T) {
	resourceName := "grafana_playlist.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")
	updatedName := "updated name"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(resourceName),
				),
			},
			{
				Config: testAccPlaylistConfigUpdate(updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(resourceName),
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile(`\d+`)),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "item.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "item.*", map[string]string{
						"order": "1",
						"title": "Terraform Dashboard By ID",
						"type":  "dashboard_by_id",
						"value": "3",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_disappears(t *testing.T) {
	resourceName := "grafana_playlist.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(resourceName),
					testAccPlaylistDisappears(resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccPlaylistCheckExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = client.Playlist(id)

		if err != nil {
			return fmt.Errorf("error getting playlist: %w", err)
		}

		return nil
	}
}

func testAccPlaylistDisappears(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		err = client.DeletePlaylist(id)
		return err
	}
}

func testAccPlaylistDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*client).gapi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_playlist" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		playlist, err := client.Playlist(id)

		if err != nil {
			if strings.HasPrefix(err.Error(), "status: 404") {
				continue
			}
			return err
		}

		if playlist != nil {
			return fmt.Errorf("Playlist still exists")
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

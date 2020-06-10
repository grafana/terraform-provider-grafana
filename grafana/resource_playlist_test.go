package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/nytm/go-grafana-api"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPlaylist_basic(t *testing.T) {
	var playlist gapi.Playlist
	rn := "grafana_playlist.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(rn, &playlist),
					resource.TestMatchResourceAttr(rn, "id", regexp.MustCompile(`\d+`)),
					resource.TestCheckResourceAttr(rn, "name", rName),
					resource.TestCheckResourceAttr(rn, "item.#", "2"),
					resource.TestCheckResourceAttr(rn, "item.3264172724.order", "1"),
					resource.TestCheckResourceAttr(rn, "item.3264172724.title", "Terraform Dashboard By Tag"),
					resource.TestCheckResourceAttr(rn, "item.3536342863.order", "2"),
					resource.TestCheckResourceAttr(rn, "item.3536342863.title", "Terraform Dashboard By ID"),
				),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_update(t *testing.T) {
	var playlist gapi.Playlist
	rn := "grafana_playlist.test"
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
					testAccPlaylistCheckExists(rn, &playlist),
				),
			},
			{
				Config: testAccPlaylistConfigUpdate(updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(rn, &playlist),
					resource.TestMatchResourceAttr(rn, "id", regexp.MustCompile(`\d+`)),
					resource.TestCheckResourceAttr(rn, "name", updatedName),
					resource.TestCheckResourceAttr(rn, "item.#", "1"),
					resource.TestCheckResourceAttr(rn, "item.1966873348.order", "1"),
					resource.TestCheckResourceAttr(rn, "item.1966873348.title", "Terraform Dashboard By ID"),
					resource.TestCheckResourceAttr(rn, "item.1966873348.type", "dashboard_by_id"),
					resource.TestCheckResourceAttr(rn, "item.1966873348.value", "3"),
				),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_disappears(t *testing.T) {
	var playlist gapi.Playlist
	rn := "grafana_playlist.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(rn, &playlist),
					testAccPlaylistDisappears(rn),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccPlaylistCheckExists(rn string, playlist *gapi.Playlist) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		gotPlaylist, err := client.Playlist(id)
		if err != nil {
			return fmt.Errorf("error getting playlist: %s", err)
		}

		*playlist = *gotPlaylist

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

		client := testAccProvider.Meta().(*gapi.Client)
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		err = client.DeletePlaylist(id)
		return err
	}
}

func testAccPlaylistDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*gapi.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_playlist" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		gotPlaylist, err := client.Playlist(id)
		if err != nil {
			if err.Error() == "404 Not Found" {
				return nil
			}
			return err
		}
		if gotPlaylist != nil {
			return fmt.Errorf("Playlist still exists")
		}
	}
	return nil
}

func testAccPlaylistConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "grafana_playlist" "test" {
	name = "%s"
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
	name = "%s"
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

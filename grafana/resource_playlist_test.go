package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gapi "github.com/nytm/go-grafana-api"
)

const testAccPlaylistConfig_basic = `
resource "grafana_playlist" "test" {
	name = "All Dashboards"
	interval = "5m"
	item {
		title = "Test 1"
		value = grafana_dashboard.board1.dashboard_id
	}
	
	item {
		title = "Test 2"
		value = grafana_dashboard.board2.dashboard_id
	}
}

resource "grafana_dashboard" "board1" {
	config_json = <<EOT
	{
		"title": "Board 1"
	}
	EOT
}

resource "grafana_dashboard" "board2" {
	config_json = <<EOT
	{
		"title": "Board 2"
	}
	EOT
}
`

const testAccPlaylistConfig_update = `
resource "grafana_playlist" "test" {
	name = "Updated Title"
	interval = "5m"
	item {
		title = "Test 1"
		value = grafana_dashboard.board1.dashboard_id
	}
}

resource "grafana_dashboard" "board1" {
	config_json = <<EOT
	{
		"title": "Board 1"
	}
	EOT
}
`

const testAccPlaylistConfig_disappear = `
resource "grafana_playlist" "test" {
	name = "Playlist Disappear"
	interval = "5m"
	item {
		title = "Test 1"
		value = grafana_dashboard.board1.dashboard_id
	}
}

resource "grafana_dashboard" "board1" {
	config_json = <<EOT
	{
		"title": "Board 1"
	}
	EOT
}
`

func TestAccPlaylist_basic(t *testing.T) {
	var playlistId int

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPlaylistCheckDestroy(&playlistId),
		Steps: []resource.TestStep{
			// first step creates the resource
			{
				Config: testAccPlaylistConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists("grafana_playlist.test", &playlistId),
					resource.TestMatchResourceAttr("grafana_playlist.test", "item.#", regexp.MustCompile(`2`)),
				),
			},
			// update playlist
			{
				Config: testAccPlaylistConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCompareId("grafana_playlist.test", &playlistId),
					testAccPlaylistCheckExists("grafana_playlist.test", &playlistId),
					resource.TestMatchResourceAttr("grafana_playlist.test", "item.#", regexp.MustCompile(`1`)),
				),
			},
			// final step checks importing the current state we reached in the step above
			{
				ResourceName:      "grafana_playlist.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_disappear(t *testing.T) {
	var playlistId int

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPlaylistCheckDestroy(&playlistId),
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfig_disappear,
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists("grafana_playlist.test", &playlistId),
					testAccPlaylistDisappear(&playlistId),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccPlaylistCheckDestroy(playlistId *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)

		_, err := client.Playlist(*playlistId)
		if err == nil {
			return fmt.Errorf("playlist with id %d still exists (%v)", *playlistId, err)
		}

		return nil
	}
}

func testAccPlaylistCheckExists(resourceName string, playlistId *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		id, err := strconv.Atoi(resourceState.Primary.ID)
		if err != nil {
			return err
		}

		client := testAccProvider.Meta().(*gapi.Client)
		_, err = client.Playlist(id)
		if err != nil {
			return fmt.Errorf("error getting playlist: %s", err)
		}

		*playlistId = id
		return nil
	}
}

func testAccPlaylistCompareId(resourceName string, playlistId *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID != strconv.Itoa(*playlistId) {
			return fmt.Errorf("resource id != expected resource id")
		}

		return nil
	}
}

func testAccPlaylistDisappear(playlistId *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// At this point testAccPlaylistCheckExists should have been called and
		// playlsit should have been populated
		client := testAccProvider.Meta().(*gapi.Client)
		_ = client.DeletePlaylist(*playlistId)
		return nil
	}
}

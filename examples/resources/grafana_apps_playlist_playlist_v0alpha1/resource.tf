resource "grafana_apps_playlist_playlist_v0alpha1" "example" {
  metadata {
    uid = "example-playlist"
  }

  spec {
    title    = "Example Playlist"
    interval = "5m"

    items = [
      {
        type  = "dashboard_by_uid"
        value = "example-dashboard-uid"
      }
    ]
  }
}

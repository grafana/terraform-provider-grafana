resource "grafana_apps_playlist_playlist_v1" "example" {
  metadata {
    uid = "example-playlist"
  }

  spec {
    title    = "Example Playlist"
    interval = "5m"

    items = [
      {
        type  = "dashboard_by_uid"
        value = grafana_apps_dashboard_dashboard_v1.my_dashboard.metadata.uid
      }
    ]
  }
}

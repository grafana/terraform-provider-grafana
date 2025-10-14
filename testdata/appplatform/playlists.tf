resource "grafana_apps_playlist_playlist_v0alpha1" "test_playlist" {
  metadata {
    uid = "test_playlist"
  }

  spec {
    title    = "Test Playlist"
    interval = "1h"
    items = [
      {
        type  = "dashboard_by_uid"
        value = grafana_apps_dashboard_dashboard_v1alpha1.test_dashboard_one.metadata.uid
      },
      {
        type  = "dashboard_by_uid"
        value = grafana_apps_dashboard_dashboard_v1alpha1.test_dashboard_two.metadata.uid
      },
    ]
  }

  options {
    overwrite = true
  }
}

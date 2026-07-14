resource "grafana_playlist" "test" {
  name     = "My Playlist!"
  interval = "5m"

  item {
    // Order is required, and is the order in which the dashboards will be displayed
    // The block order is ignored
    order = 2
    type  = "dashboard_by_tag"
    value = "terraform"
  }

  item {
    order = 1
    type  = "dashboard_by_uid"
    value = "cIBgcSjkk"
  }
}

resource "grafana_organization_preferences" "test" {
  theme      = "light"
  timezone   = "utc"
  week_start = "sunday"
}

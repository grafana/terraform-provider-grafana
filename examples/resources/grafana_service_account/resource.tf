resource "grafana_service_account" "admin" {
  name        = "admin sa"
  role        = "Admin"
  is_disabled = false
}

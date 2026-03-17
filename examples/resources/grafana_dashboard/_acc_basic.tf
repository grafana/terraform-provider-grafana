resource "grafana_dashboard" "test" {
  config_json = jsonencode({
    title = "Terraform Acceptance Test"
    uid   = "basic"
  })
  message = "inital commit."
}

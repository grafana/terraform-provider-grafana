resource "grafana_assistant_skill" "example" {
  name  = "Deploy readiness check"
  body  = <<-EOT
  1. Check deployment pipeline status.
  2. Verify SLO error budget before promoting.
  EOT
  scope = "tenant"
}

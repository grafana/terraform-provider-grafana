provider "grafana" {
  url  = "http://grafana.example.com/"
  auth = var.grafana_auth
}

// Optional (On-premise, not supported in Grafana Cloud): Create an organization
resource "grafana_organization" "my_org" {
  name = "my_org"
}

// Create resources (optional: within the organization)
resource "grafana_folder" "my_folder" {
  org_id = grafana_organization.my_org.org_id
  title  = "Test Folder"
}

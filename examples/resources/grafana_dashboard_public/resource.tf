// Optional (On-premise, not supported in Grafana Cloud): Create an organization
resource "grafana_organization" "my_org" {
  name = "test 1"
}

// Create resources (optional: within the organization)
resource "grafana_folder" "my_folder" {
  org_id = grafana_organization.my_org.org_id
  title  = "test Folder"
}

resource "grafana_dashboard" "test_dash" {
  org_id = grafana_organization.my_org.org_id
  folder = grafana_folder.my_folder.id
  config_json = jsonencode({
    "title" : "My Terraform Dashboard",
    "uid" : "my-dashboard-uid"
  })
}

resource "grafana_dashboard_public" "my_public_dashboard" {
  org_id        = grafana_organization.my_org.org_id
  dashboard_uid = grafana_dashboard.test_dash.uid

  uid          = "my-custom-public-uid"
  access_token = "e99e4275da6f410d83760eefa934d8d2"

  time_selection_enabled = true
  is_enabled             = true
  annotations_enabled    = true
  share                  = "public"
}

// Optional (On-premise, not supported in Grafana Cloud): Create an organization
resource "grafana_organization" "my_org2" {
  name = "test 2"
}

resource "grafana_dashboard" "test_dash2" {
  org_id = grafana_organization.my_org2.org_id
  config_json = jsonencode({
    "title" : "My Terraform Dashboard2",
    "uid" : "my-dashboard-uid2"
  })
}

resource "grafana_dashboard_public" "my_public_dashboard2" {
  org_id        = grafana_organization.my_org2.org_id
  dashboard_uid = grafana_dashboard.test_dash2.uid

  share = "public"
}

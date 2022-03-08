// Step 1: Create an organization
provider "grafana" {
  alias = "base"
  url   = "http://grafana.example.com/"
  auth  = var.grafana_auth
}

resource "grafana_organization" "my_org" {
  provider = grafana.base
  name     = "my_org"
}

// Step 2: Create resources within the organization
provider "grafana" {
  alias  = "my_org"
  url    = "http://grafana.example.com/"
  auth   = var.grafana_auth
  org_id = grafana_organization.my_org.org_id
}

resource "grafana_folder" "my_folder" {
  provider = grafana.my_org

  title = "Test Folder"
}

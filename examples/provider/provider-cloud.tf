// Step 1: Create a stack
provider "grafana" {
  alias         = "cloud"
  cloud_api_key = "my-token"
}

resource "grafana_cloud_stack" "my_stack" {
  provider = grafana.cloud

  name        = "myteststack"
  slug        = "myteststack"
  region_slug = "us"
}

resource "grafana_api_key" "management" {
  provider = grafana.cloud

  cloud_stack_slug = grafana_cloud_stack.my_stack.slug
  name             = "management-key"
  role             = "Admin"
}

// Step 2: Create resources within the stack
provider "grafana" {
  alias = "my_stack"

  url  = grafana_cloud_stack.my_stack.url
  auth = grafana_api_key.management.key
}

resource "grafana_folder" "my_folder" {
  provider = grafana.my_stack

  title = "Test Folder"
}

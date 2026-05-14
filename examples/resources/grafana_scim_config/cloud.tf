resource "grafana_cloud_stack" "my_stack" {
  name = "my-stack"
  slug = "my-stack"
}

resource "grafana_cloud_stack_service_account" "cloud_sa" {
  stack_slug = grafana_cloud_stack.my_stack.slug
  name       = "scim-terraform"
  role       = "Admin"
}

resource "grafana_cloud_stack_service_account_token" "cloud_sa" {
  stack_slug         = grafana_cloud_stack.my_stack.slug
  service_account_id = grafana_cloud_stack_service_account.cloud_sa.id
  name               = "scim-terraform"
}

# A stack-level provider must set `stack_id` so SCIM API requests are
# routed to the correct stack namespace. Without it, requests go to the
# default namespace and fail with `403 authn.invalid-namespace`.
provider "grafana" {
  alias    = "stack"
  url      = grafana_cloud_stack.my_stack.url
  auth     = grafana_cloud_stack_service_account_token.cloud_sa.key
  stack_id = grafana_cloud_stack.my_stack.id
}

resource "grafana_scim_config" "default" {
  provider = grafana.stack

  enable_user_sync             = true
  enable_group_sync            = false
  reject_non_provisioned_users = false
}

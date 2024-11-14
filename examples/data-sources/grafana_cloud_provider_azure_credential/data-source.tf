resource "grafana_cloud_provider_azure_credential" "test" {
  stack_id      = "1"
  name          = "test-name"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
  tenant_id     = "my-tenant-id"
}

data "grafana_cloud_provider_azure_credential" "test" {
  stack_id      = grafana_cloud_provider_azure_credential.test.stack_id
  name          = grafana_cloud_provider_azure_credential.test.name
  client_id     = grafana_cloud_provider_azure_credential.test.client_id
  client_secret = grafana_cloud_provider_azure_credential.test.client_secret
  tenant_id     = grafana_cloud_provider_azure_credential.test.tenant_id
  resource_id   = grafana_cloud_provider_azure_credential.test.resource_id
}
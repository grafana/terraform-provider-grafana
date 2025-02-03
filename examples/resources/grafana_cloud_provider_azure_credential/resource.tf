resource "grafana_cloud_provider_azure_credential" "test" {
  stack_id      = "1"
  name          = "test-name"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
  tenant_id     = "my-tenant-id"

  resource_tags_to_add_to_metrics = ["tag1", "tag2"]

  resource_discovery_tag_filter {
    key   = "key-1"
    value = "value-1"
  }

  resource_discovery_tag_filter {
    key   = "key-2"
    value = "value-2"
  }

  auto_discovery_configuration {
    subscription_id = "my-subscription_id"

    resource_type_configurations {
      resource_type_name = "Microsoft.App/containerApps"

      metric_configuration {
        name = "TotalCoresQuotaUsed"
      }
    }

    resource_type_configurations {
      resource_type_name = "Microsoft.Storage/storageAccounts/tableServices"

      metric_configuration {
        name         = "Availability"
        dimensions   = ["GeoType", "ApiName"]
        aggregations = ["Average"]
      }
    }

  }
}

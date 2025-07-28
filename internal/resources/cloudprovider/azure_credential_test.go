package cloudprovider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Tests both managed resource and data source
func TestAcc_AzureCredential(t *testing.T) {
	resourceID := "3"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/stacks/1/azure/credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(fmt.Sprintf(`
{
  "data": {
    "id": "%s",
    "name": "test-name",
    "tenant_id": "my_tenant_id",
    "client_id": "my_client_id",
    "client_secret": "",
    "stack_id": "1",
    "resource_tag_filters": [
      {
        "key": "key-1",
        "value": "value-1"
      },
      {
        "key": "key-2",
        "value": "value-2"
      }
    ],
	"resource_tags_to_add_to_metrics" : ["tag1", "tag2"],
    "auto_discovery_configuration": [
      {
        "subscription_id": "my-subscription_id",
        "resource_type_configurations": [
          {
            "resource_type_name": "Microsoft.App/containerApps",
            "metric_configuration": [
              {
                "name": "TotalCoresQuotaUsed"
              }
            ]
          },
          {
            "resource_type_name": "Microsoft.Storage/storageAccounts/tableServices",
            "metric_configuration": [
              {
                "name": "Availability",
                "dimensions": [
                  "GeoType",
                  "ApiName"
                ],
				"aggregations": [ "Average" ]
              }
            ]
          }
        ]
      }
    ]
  }
}
`, resourceID)))
		}
	})

	mux.HandleFunc("/api/v2/stacks/1/azure/credentials/"+resourceID, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`
{
  "data": {
    "id": "%s",
    "name": "test-name",
    "tenant_id": "my-tenant-id",
    "client_id": "my-client-id",
    "client_secret": "",
    "stack_id":"1",
    "resource_tag_filters": [
      {
        "key": "key-1",
        "value": "value-1"
      },
      {
        "key": "key-2",
        "value": "value-2"
      }
    ],
	"resource_tags_to_add_to_metrics" : ["tag1", "tag2"],
	"auto_discovery_configuration": [
      {
        "subscription_id": "my-subscription_id",
        "resource_type_configurations": [
          {
            "resource_type_name": "Microsoft.App/containerApps",
            "metric_configuration": [
              {
                "name": "TotalCoresQuotaUsed"
              }
            ]
          },
          {
            "resource_type_name": "Microsoft.Storage/storageAccounts/tableServices",
            "metric_configuration": [
              {
                "name": "Availability",
                "dimensions": [
                  "GeoType",
                  "ApiName"
                ],
				"aggregations": [ "Average" ]
              }
            ]
          }
        ]
      }
    ]
  }
}
`, resourceID)))
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`
{
  "data": {
    "id": "%s",
    "name": "test-name",
    "tenant_id": "my-tenant-id",
    "client_id": "my-client-id",
    "client_secret": "",
    "stack_id":"1",
    "resource_tag_filters": [
      {
        "key": "key-1",
        "value": "value-1"
      },
      {
        "key": "key-2",
        "value": "value-2"
      }
    ],
	"resource_tags_to_add_to_metrics" : ["tag1", "tag2"],
	"auto_discovery_configuration": [
      {
        "subscription_id": "my-subscription_id",
        "resource_type_configurations": [
          {
            "resource_type_name": "Microsoft.App/containerApps",
            "metric_configuration": [
              {
                "name": "TotalCoresQuotaUsed"
              }
            ]
          },
          {
            "resource_type_name": "Microsoft.Storage/storageAccounts/tableServices",
            "metric_configuration": [
              {
                "name": "Availability",
                "dimensions": [
                  "GeoType",
                  "ApiName"
                ],
				"aggregations": [ "Average" ]
              }
            ]
          }
        ]
      }
    ]
  }
}
`, resourceID)))

		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	defer os.Setenv("GRAFANA_CLOUD_PROVIDER_URL", os.Getenv("GRAFANA_CLOUD_PROVIDER_URL"))
	defer os.Setenv("GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN", os.Getenv("GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN"))

	// set this to use the mock server
	_ = os.Setenv("GRAFANA_CLOUD_PROVIDER_URL", server.URL)
	_ = os.Setenv("GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN", "some_token")

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Creates a managed resource
				Config: testutils.TestAccExample(t, "resources/grafana_cloud_provider_azure_credential/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "id", "1:3"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "stack_id", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "name", "test-name"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "tenant_id", "my-tenant-id"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "client_id", "my-client-id"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "client_secret", "my-client-secret"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.0.key", "key-1"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.1.key", "key-2"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.0.value", "value-1"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.1.value", "value-2"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.subscription_id", "my-subscription_id"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.0.resource_type_name", "Microsoft.App/containerApps"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.0.metric_configuration.0.name", "TotalCoresQuotaUsed"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.resource_type_name", "Microsoft.Storage/storageAccounts/tableServices"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.name", "Availability"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.dimensions.0", "GeoType"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.dimensions.1", "ApiName"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.aggregations.0", "Average"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "resource_tags_to_add_to_metrics.0", "tag1"),
					resource.TestCheckResourceAttr("grafana_cloud_provider_azure_credential.test", "resource_tags_to_add_to_metrics.1", "tag2"),
				),
			},
			{
				// Tests data source resource
				Config: testutils.TestAccExample(t, "data-sources/grafana_cloud_provider_azure_credential/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "id", "1:3"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "stack_id", "1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "name", "test-name"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "tenant_id", "my-tenant-id"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "client_id", "my-client-id"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "client_secret", ""),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "resource_id", resourceID),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.0.key", "key-1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.1.key", "key-2"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.0.value", "value-1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "resource_discovery_tag_filter.1.value", "value-2"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.subscription_id", "my-subscription_id"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.0.resource_type_name", "Microsoft.App/containerApps"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.0.metric_configuration.0.name", "TotalCoresQuotaUsed"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.resource_type_name", "Microsoft.Storage/storageAccounts/tableServices"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.name", "Availability"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.dimensions.0", "GeoType"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.dimensions.1", "ApiName"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "auto_discovery_configuration.0.resource_type_configurations.1.metric_configuration.0.aggregations.0", "Average"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "resource_tags_to_add_to_metrics.0", "tag1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_provider_azure_credential.test", "resource_tags_to_add_to_metrics.1", "tag2"),
				),
			},
		},
	})
}

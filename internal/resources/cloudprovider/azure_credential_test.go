package cloudprovider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
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
    "stack_id":"1",
    "resource_discovery_tag_filters": [
      {
        "key": "key-1",
        "value": "value-1"
      },
      {
        "key": "key-2",
        "value": "value-2"
      }

    ]
  }
}`, resourceID)))
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
    "resource_discovery_tag_filters": [
      {
        "key": "key-1",
        "value": "value-1"
      },
      {
        "key": "key-2",
        "value": "value-2"
      }

    ]
  }
}`, resourceID)))
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
    "resource_discovery_tag_filters": [
      {
        "key": "key-1",
        "value": "value-1"
      },
      {
        "key": "key-2",
        "value": "value-2"
      }

    ]
  }
}`, resourceID)))

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
				),
			},
		},
	})
}

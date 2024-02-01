package cloud_test

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/api_keys"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGrafanaAuthKeyFromCloud(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gcom.FormattedApiInstance
	prefix := "tfapikeytest"
	slug := GetRandomStackName(prefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyFromCloud(slug, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaAuthCheckKeys(&stack, []string{"management-key"}),
					resource.TestCheckResourceAttrSet("grafana_cloud_stack_api_key.management", "key"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_api_key.management", "name", "management-key"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_api_key.management", "role", "Admin"),
					resource.TestCheckNoResourceAttr("grafana_cloud_stack_api_key.management", "expiration"),
				),
			},
			{
				Config: testAccStackConfigBasic(slug, slug, "description"),
				Check:  testAccGrafanaAuthCheckKeys(&stack, []string{}),
			},
		},
	})
}

func testAccGrafanaAuthKeyFromCloud(name, slug string) string {
	return testAccStackConfigBasic(name, slug, "description") + `
	resource "grafana_cloud_stack_api_key" "management" {
		stack_slug = grafana_cloud_stack.test.slug
		name       = "management-key"
		role       = "Admin"
	}
	`
}

func testAccGrafanaAuthCheckKeys(stack *gcom.FormattedApiInstance, expectedKeys []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cloudClient := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPIOpenAPI
		c, cleanup, err := createTemporaryStackGrafanaClient(context.Background(), cloudClient, stack.Slug, "test-api-key-")
		if err != nil {
			return err
		}
		defer cleanup()

		response, err := c.APIKeys.GetAPIkeys(api_keys.NewGetAPIkeysParams())
		if err != nil {
			return fmt.Errorf("failed to get API keys: %w", err)
		}

		var foundKeys []string
		for _, key := range response.Payload {
			if !strings.HasPrefix(key.Name, "test-api-key-") {
				foundKeys = append(foundKeys, key.Name)
			}
		}

		if len(foundKeys) != len(expectedKeys) {
			return fmt.Errorf("expected %d keys, got %d", len(expectedKeys), len(foundKeys))
		}
		for _, expectedKey := range expectedKeys {
			found := false
			for _, foundKey := range foundKeys {
				if expectedKey == foundKey {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected to find key %s, but it was not found", expectedKey)
			}
		}

		return nil
	}
}

func createTemporaryStackGrafanaClient(ctx context.Context, cloudClient *gcom.APIClient, stackSlug, tempSaPrefix string) (*goapi.GrafanaHTTPAPI, func() error, error) {
	stack, _, err := cloudClient.InstancesAPI.GetInstance(ctx, stackSlug).Execute()
	if err != nil {
		return nil, nil, err
	}

	name := fmt.Sprintf("%s%d", tempSaPrefix, time.Now().UnixNano())

	req := gcom.PostInstanceServiceAccountsRequest{
		Name: name,
		Role: "Admin",
	}

	sa, _, err := cloudClient.InstancesAPI.PostInstanceServiceAccounts(ctx, stackSlug).
		PostInstanceServiceAccountsRequest(req).
		XRequestId(cloud.ClientRequestID()).
		Execute()
	if err != nil {
		return nil, nil, err
	}

	tokenRequest := gcom.PostInstanceServiceAccountTokensRequest{
		Name:          name,
		SecondsToLive: common.Ref(int32(60)),
	}
	token, _, err := cloudClient.InstancesAPI.PostInstanceServiceAccountTokens(ctx, stackSlug, fmt.Sprintf("%d", int(sa.Id))).
		PostInstanceServiceAccountTokensRequest(tokenRequest).
		XRequestId(cloud.ClientRequestID()).
		Execute()
	if err != nil {
		return nil, nil, err
	}

	stackURLParsed, err := url.Parse(stack.Url)
	if err != nil {
		return nil, nil, err
	}

	client := goapi.NewHTTPClientWithConfig(nil, &goapi.TransportConfig{
		Host:         stackURLParsed.Host,
		Schemes:      []string{stackURLParsed.Scheme},
		BasePath:     "api",
		APIKey:       token.Key,
		NumRetries:   5,
		RetryTimeout: 10 * time.Second,
	})

	cleanup := func() error {
		_, err = client.ServiceAccounts.DeleteServiceAccount(int64(sa.Id))
		return err
	}

	return client, cleanup, nil
}

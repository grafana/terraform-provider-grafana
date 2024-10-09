package connections_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/connectionsapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/connections"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

func TestAcc_MetricsEndpointScrapeJob(t *testing.T) {
	// t.Skip("will be enabled after Connections API is available in prod")
	// testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccMetricsEndpointCheckDestroy("1", "scrape-job-name"),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_connections_metrics_endpoint_scrape_job/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "stack_id", "1"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "name", "scrape-job-name"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_method", "basic"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_username", "my-username"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "authentication_basic_password", "my-password"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "url", "https://dev.my-metrics-endpoint-url.com:9000/metrics"),
					resource.TestCheckResourceAttr("grafana_connections_metrics_endpoint_scrape_job.test", "scrape_interval_seconds", "60"),
				),
			},
		},
	})
}

func testAccMetricsEndpointCheckDestroy(stackID string, jobName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).ConnectionsAPIClient
		_, err := client.GetMetricsEndpointScrapeJob(context.Background(), stackID, jobName)
		if err != nil {
			if errors.Is(err, connectionsapi.ErrNotFound) {
				return nil
			}
			return fmt.Errorf("metrics endpoint job should return ErrNotFound but returned error %s", err.Error())
		}

		return fmt.Errorf("metrics endpoint job should return ErrNotFound but returned no error")
	}
}

func Test_httpsURLValidator(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		providedURL   types.String
		expectedDiags diag.Diagnostics
	}{
		"valid url with https": {
			providedURL:   types.StringValue("https://dev.my-metrics-endpoint-url.com:9000/metrics"),
			expectedDiags: nil,
		},
		"invalid empty string": {
			providedURL: types.StringValue(""),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A valid URL is required.\n\nGiven Value: \"\"\n",
			)},
		},
		"invalid null": {
			providedURL: types.StringNull(),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A valid URL is required.\n\nGiven Value: \"\"\n",
			)},
		},
		"invalid unknown": {
			providedURL: types.StringUnknown(),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A valid URL is required.\n\nGiven Value: \"\"\n",
			)},
		},
		"invalid not a url": {
			providedURL: types.StringValue("this is not a url"),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A URL was provided, protocol must be HTTPS.\n\nGiven Value: \"this is not a url\"\n",
			)},
		},
		"invalid leading space url": {
			providedURL: types.StringValue(" https://leading.space"),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A string value was provided that is not a valid URL.\n\nGiven Value:  https://leading.space\nError: parse \" https://leading.space\": first path segment in URL cannot contain colon",
			)},
		},
		"invalid url without https": {
			providedURL: types.StringValue("www.google.com"),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A URL was provided, protocol must be HTTPS.\n\nGiven Value: \"www.google.com\"\n",
			)},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			res := validator.StringResponse{}
			connections.HTTPSURLValidator{}.ValidateString(
				context.Background(),
				validator.StringRequest{
					ConfigValue: tc.providedURL,
					Path:        path.Root("test"),
				},
				&res)

			assert.Equal(t, tc.expectedDiags, res.Diagnostics)
		})
	}
}

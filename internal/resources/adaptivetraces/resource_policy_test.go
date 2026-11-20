package adaptivetraces_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/adaptivetracesapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

const policyResourceName = "grafana_adaptive_traces_policy.test"

var uuidRegexp = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// policyBodies holds a valid body for each policy type: the examples the API publishes at
// GET /adaptive-traces/api/v1/policy_schemas.
var policyBodies = map[string]string{
	"always_sample":     `{}`,
	"and":               `{"and_sub_policy":[{"name":"important-service-policy","type":"string_attribute","string_attribute":{"key":"service.name","values":["service-name"]}},{"name":"twenty-percent","type":"probabilistic","probabilistic":{"sampling_percentage":20}}]}`,
	"boolean_attribute": `{"key":"app.force_sample","value":true}`,
	"bytes_limiting":    `{"bytes_per_second":1048576,"burst_capacity":5242880}`,
	"composite":         `{"max_total_spans_per_second":1000,"policy_order":["composite-policy-1","composite-policy-2","composite-policy-3"],"composite_sub_policy":[{"name":"composite-policy-1","type":"numeric_attribute","numeric_attribute":{"key":"key1","min_value":50,"max_value":100}},{"name":"composite-policy-2","type":"string_attribute","string_attribute":{"key":"key2","values":["value1","value2"]}},{"name":"composite-policy-3","type":"always_sample"}],"rate_allocation":[{"policy":"composite-policy-1","percent":50},{"policy":"composite-policy-2","percent":25}]}`,
	"diversity":         `{}`,
	"drop":              `{"drop_sub_policy":[{"name":"monitoring-endpoints","type":"string_attribute","string_attribute":{"key":"url.path","values":["/health","/metrics"]}}]}`,
	"latency":           `{"threshold_ms":5000,"upper_threshold_ms":10000}`,
	"numeric_attribute": `{"key":"key","min_value":50,"max_value":100}`,
	"ottl_condition":    `{"error_mode":"ignore","span":["attributes[\"test_attr_key_1\"] == \"test_attr_val_1\"","attributes[\"test_attr_key_2\"] != \"test_attr_val_1\""],"spanevent":["name != \"test_span_event_name\"","attributes[\"test_event_attr_key_2\"] != \"test_event_attr_val_1\""]}`,
	"probabilistic":     `{"sampling_percentage":5}`,
	"rate_limiting":     `{"spans_per_second":50}`,
	"span_count":        `{"min_spans":2,"max_spans":20}`,
	"status_code":       `{"status_codes":["ERROR"]}`,
	"string_attribute":  `{"key":"http.url","values":["/health","/metrics"]}`,
	"trace_flags":       `{}`,
	"trace_state":       `{"key":"key","values":["value1","value2"]}`,
	"volumetric":        `{"sampling_percentage":5}`,
}

// The policies share the stack, so these tests must not run in parallel. See testutils.CheckAdaptiveTracesTestsEnabled
// for the stack they need.

// testAccPolicyConfig renders a policy named "test". The body is a JSON document that is passed through
// jsonencode(), as users are expected to write it. A drop policy is only accepted while another policy is
// active, so it gets a policy to depend on.
func testAccPolicyConfig(policyType, name, body string) string {
	body = strings.NewReplacer("${", "$${", "%{", "%%{").Replace(body)

	var base, dependsOn string
	if policyType == "drop" {
		base = `
resource "grafana_adaptive_traces_policy" "base" {
  name = "tf-acc-test-base"
  type = "probabilistic"
  body = jsonencode({ sampling_percentage = 5 })
}
`
		dependsOn = "depends_on = [grafana_adaptive_traces_policy.base]"
	}

	return fmt.Sprintf(`%[1]s
resource "grafana_adaptive_traces_policy" "test" {
  name = %[2]q
  type = %[3]q
  body = jsonencode(jsondecode(<<-EOT
%[4]s
EOT
  ))
  %[5]s
}
`, base, name, policyType, body, dependsOn)
}

func testAccCheckPolicyBody(expected string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(policyResourceName, "body", func(actual string) error {
		var want, got any
		if err := json.Unmarshal([]byte(expected), &want); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(actual), &got); err != nil {
			return err
		}
		if !reflect.DeepEqual(want, got) {
			return fmt.Errorf("expected body %s, got %s", expected, actual)
		}
		return nil
	})
}

func testAccPolicyID(id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[policyResourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", policyResourceName)
		}
		*id = rs.Primary.ID
		return nil
	}
}

func testAccCheckPolicyDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AdaptiveTracesClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_adaptive_traces_policy" {
			continue
		}
		_, err := client.GetPolicy(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("policy %s still exists", rs.Primary.ID)
		}
		if !errors.Is(err, adaptivetracesapi.ErrNotFound) {
			return fmt.Errorf("unexpected error reading policy %s: %w", rs.Primary.ID, err)
		}
	}
	return nil
}

func TestAccPolicy_types(t *testing.T) {
	testutils.CheckAdaptiveTracesTestsEnabled(t)

	for _, policyType := range slices.Sorted(maps.Keys(policyBodies)) {
		t.Run(policyType, func(t *testing.T) {
			example := policyBodies[policyType]
			name := "tf-acc-test-" + policyType
			renamed := name + "-renamed"

			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             testAccCheckPolicyDestroy,
				Steps: []resource.TestStep{
					{
						Config: testAccPolicyConfig(policyType, name, example),
						Check: resource.ComposeTestCheckFunc(
							resource.TestMatchResourceAttr(policyResourceName, "id", uuidRegexp),
							resource.TestCheckResourceAttr(policyResourceName, "name", name),
							resource.TestCheckResourceAttr(policyResourceName, "type", policyType),
							testAccCheckPolicyBody(example),
							resource.TestCheckResourceAttrSet(policyResourceName, "version_created_at"),
							resource.TestCheckResourceAttrSet(policyResourceName, "version_created_by"),
						),
					},
					{
						ResourceName:      policyResourceName,
						ImportState:       true,
						ImportStateVerify: true,
					},
					{
						Config: testAccPolicyConfig(policyType, renamed, example),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(policyResourceName, "name", renamed),
							testAccCheckPolicyBody(example),
						),
					},
					{
						// The refreshed policy must not differ from the configuration.
						Config:   testAccPolicyConfig(policyType, renamed, example),
						PlanOnly: true,
					},
				},
			})
		})
	}
}

func TestAccPolicy_changeType(t *testing.T) {
	testutils.CheckAdaptiveTracesTestsEnabled(t)

	var id string

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyConfig("probabilistic", "tf-acc-test-change-type", `{"sampling_percentage": 5}`),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyID(&id),
					resource.TestCheckResourceAttr(policyResourceName, "type", "probabilistic"),
				),
			},
			{
				Config: testAccPolicyConfig("rate_limiting", "tf-acc-test-change-type", `{"spans_per_second": 50}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(policyResourceName, "type", "rate_limiting"),
					testAccCheckPolicyBody(`{"spans_per_second": 50}`),
					// The policy is updated in place.
					resource.TestCheckResourceAttrWith(policyResourceName, "id", func(value string) error {
						if value != id {
							return fmt.Errorf("policy was replaced: id changed from %s to %s", id, value)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccPolicy_deletedOutsideTerraform(t *testing.T) {
	testutils.CheckAdaptiveTracesTestsEnabled(t)

	var id string
	config := testAccPolicyConfig("probabilistic", "tf-acc-test-deleted", `{"sampling_percentage": 5}`)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccPolicyID(&id),
			},
			{
				PreConfig: func() {
					client := testutils.Provider.Meta().(*common.Client).AdaptiveTracesClient
					if err := client.DeletePolicy(context.Background(), id); err != nil {
						t.Fatalf("failed to delete policy %s: %v", id, err)
					}
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// The policy is created again.
				Config: config,
				Check: resource.TestCheckResourceAttrWith(policyResourceName, "id", func(value string) error {
					if value == id {
						return fmt.Errorf("expected a new policy, got the deleted policy %s again", id)
					}
					return nil
				}),
			},
		},
	})
}

func TestAccPolicy_expirySetOutsideTerraform(t *testing.T) {
	testutils.CheckAdaptiveTracesTestsEnabled(t)

	var id string
	const name = "tf-acc-test-expiry"
	body := `{"sampling_percentage": 5}`

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyConfig("probabilistic", name, body),
				Check:  testAccPolicyID(&id),
			},
			{
				// A policy with an expiry cannot be imported.
				PreConfig: func() {
					client := testutils.Provider.Meta().(*common.Client).AdaptiveTracesClient
					policy, err := client.GetPolicy(context.Background(), id)
					if err != nil {
						t.Fatalf("failed to read policy %s: %v", id, err)
					}
					expiresAt := time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)
					policy.ExpiresAt = &expiresAt
					if _, err := client.UpdatePolicy(context.Background(), id, *policy); err != nil {
						t.Fatalf("failed to set an expiry on policy %s: %v", id, err)
					}
				},
				ResourceName: policyResourceName,
				ImportState:  true,
				ExpectError:  regexp.MustCompile(`Expiring policies cannot be managed`),
			},
			{
				// The next update clears the expiry.
				Config: testAccPolicyConfig("probabilistic", name+"-renamed", body),
				Check: func(*terraform.State) error {
					client := testutils.Provider.Meta().(*common.Client).AdaptiveTracesClient
					policy, err := client.GetPolicy(context.Background(), id)
					if err != nil {
						return fmt.Errorf("failed to read policy %s: %w", id, err)
					}
					if policy.ExpiresAt != nil && *policy.ExpiresAt != "" {
						return fmt.Errorf("expected the update to clear the expiry, got expires_at %s", *policy.ExpiresAt)
					}
					return nil
				},
			},
		},
	})
}

func TestAccPolicy_importMalformedID(t *testing.T) {
	testutils.CheckAdaptiveTracesTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyConfig("probabilistic", "tf-acc-test-import", `{"sampling_percentage": 5}`),
			},
			{
				ResourceName:  policyResourceName,
				ImportState:   true,
				ImportStateId: "tf-acc-test-import",
				ExpectError:   regexp.MustCompile(`Expected the UUID of a policy`),
			},
		},
	})
}

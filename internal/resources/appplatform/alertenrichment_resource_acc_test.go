package appplatform_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const alertEnrichmentResourceName = "grafana_apps_alertenrichment_alertenrichment_v1beta1.test"

// importStateIDFunc returns a function that extracts the UID from metadata for import tests.
// They need an id to fetch the resource, and by default they use ID which is set to UUID in our case,
// but to get the response we need the UID.
func importStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		// Use the UID from metadata for import instead of the UUID ID
		uid := rs.Primary.Attributes["metadata.uid"]
		if uid == "" {
			return "", fmt.Errorf("UID is empty in resource %s", resourceName)
		}
		return uid, nil
	}
}

// alertEnrichmentConfigBuilder helps build HCL for alert enrichments
type alertEnrichmentConfigBuilder struct {
	uid           string
	title         string
	description   string
	alertRuleUIDs []string
	receivers     []string
	labelMatchers []matcherConfig
	annotMatchers []matcherConfig
	assignStep    *assignStepConfig
}

type matcherConfig struct {
	matchType string
	name      string
	value     string
}

type assignStepConfig struct {
	timeout     string
	annotations map[string]string
}

func newAlertEnrichmentConfig(uid, title string) *alertEnrichmentConfigBuilder {
	return &alertEnrichmentConfigBuilder{
		uid:   uid,
		title: title,
	}
}

func (b *alertEnrichmentConfigBuilder) withDescription(desc string) *alertEnrichmentConfigBuilder {
	b.description = desc
	return b
}

func (b *alertEnrichmentConfigBuilder) withAlertRuleUIDs(uids ...string) *alertEnrichmentConfigBuilder {
	b.alertRuleUIDs = uids
	return b
}

func (b *alertEnrichmentConfigBuilder) withReceivers(receivers ...string) *alertEnrichmentConfigBuilder {
	b.receivers = receivers
	return b
}

func (b *alertEnrichmentConfigBuilder) withLabelMatcher(matchType, name, value string) *alertEnrichmentConfigBuilder {
	b.labelMatchers = append(b.labelMatchers, matcherConfig{matchType, name, value})
	return b
}

func (b *alertEnrichmentConfigBuilder) withAnnotationMatcher(matchType, name, value string) *alertEnrichmentConfigBuilder {
	b.annotMatchers = append(b.annotMatchers, matcherConfig{matchType, name, value})
	return b
}

func (b *alertEnrichmentConfigBuilder) withAssignStep(timeout string, annotations map[string]string) *alertEnrichmentConfigBuilder {
	b.assignStep = &assignStepConfig{
		timeout:     timeout,
		annotations: annotations,
	}
	return b
}

func (b *alertEnrichmentConfigBuilder) build() string {
	config := fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
	metadata {
		uid = "%s"
	}

	spec {
		title = "%s"
		description = "%s"`, b.uid, b.title, b.description)

	if len(b.alertRuleUIDs) > 0 {
		config += `
		alert_rule_uids = [`
		for i, uid := range b.alertRuleUIDs {
			if i > 0 {
				config += ", "
			}
			config += fmt.Sprintf(`"%s"`, uid)
		}
		config += "]"
	}

	if len(b.receivers) > 0 {
		config += `
		receivers = [`
		for i, receiver := range b.receivers {
			if i > 0 {
				config += ", "
			}
			config += fmt.Sprintf(`"%s"`, receiver)
		}
		config += "]"
	}

	for _, matcher := range b.labelMatchers {
		config += fmt.Sprintf(`
		label_matchers {
			type  = "%s"
			name  = "%s"
			value = "%s"
		}`, matcher.matchType, matcher.name, matcher.value)
	}

	for _, matcher := range b.annotMatchers {
		config += fmt.Sprintf(`
		annotation_matchers {
			type  = "%s"
			name  = "%s"
			value = "%s"
		}`, matcher.matchType, matcher.name, matcher.value)
	}

	if b.assignStep != nil {
		config += `
		assign_step {`
		timeout := b.assignStep.timeout
		if timeout == "" {
			timeout = "0s"
		}
		config += fmt.Sprintf(`
			timeout = "%s"`, timeout)
		if len(b.assignStep.annotations) > 0 {
			config += `
			annotations = {`
			for name, value := range b.assignStep.annotations {
				config += fmt.Sprintf(`
				%s = "%s"`, name, value)
			}
			config += `
			}`
		}
		config += `
		}`
	}

	config += `
	}

	options {
		overwrite = true
	}
}
`
	return config
}

func TestAccAlertEnrichment(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("full", func(t *testing.T) {
		randSuffix := acctest.RandString(6)

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: newAlertEnrichmentConfig(
						fmt.Sprintf("comprehensive-%s", randSuffix),
						"comprehensive-alert-enrichment",
					).withDescription("description-1").
						withAlertRuleUIDs("critical-api-alerts", "critical-db-alerts").
						withReceivers("pagerduty-critical", "slack-platform", "email-oncall").
						withLabelMatcher("=", "severity", "critical").
						withLabelMatcher("!=", "environment", "test").
						withAssignStep("30s", map[string]string{
							"priority":        "P1",
							"escalation_time": "5m",
							"team_contact":    "platform-{{ $labels.service }}@company.com",
							"runbook":         "https://runbooks.company.com/{{ $labels.alert_name }}",
						}).build(),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "comprehensive-alert-enrichment"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "description-1"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "2"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.#", "3"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", "2"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.annotations.%", "4"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.timeout", "30s"),
						resource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				{
					ResourceName:      alertEnrichmentResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: []string{
						"options.%",
						"options.overwrite",
						"spec.description",
						"spec.assign_step.0.timeout",
					},
					ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
				},
			},
		})
	})

	t.Run("update", func(t *testing.T) {
		randSuffix := acctest.RandString(6)
		uid := fmt.Sprintf("update-test-%s", randSuffix)

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				// Create basic enrichment
				{
					Config: newAlertEnrichmentConfig(uid, "test-alert-enrichment").
						withDescription("description-1").
						withAssignStep("10s", map[string]string{
							"enriched_by": "terraform-test",
						}).
						build(),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "test-alert-enrichment"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.annotations.%", "1"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.timeout", "10s"),
						resource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				// Update with annotations and timeout
				{
					Config: newAlertEnrichmentConfig(uid, "updated-alert-enrichment").
						withDescription("description-2").
						withAssignStep("15s", map[string]string{
							"enriched_by":   "updated-terraform-test",
							"updated_field": "new-annotation",
						}).
						build(),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "updated-alert-enrichment"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "description-2"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.annotations.%", "2"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.timeout", "15s"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.annotations.enriched_by", "updated-terraform-test"),
						resource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				// Add matchers and alert rules
				{
					Config: newAlertEnrichmentConfig(uid, "updated-alert-enrichment").
						withDescription("description-3").
						withAlertRuleUIDs("rule-1", "rule-2").
						withLabelMatcher("=", "severity", "critical").
						withAssignStep("15s", map[string]string{
							"enriched_by":   "updated-terraform-test",
							"updated_field": "new-annotation",
						}).
						build(),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "2"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", "1"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.0.type", "="),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.0.name", "severity"),
						resource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				// Remove some fields
				{
					Config: newAlertEnrichmentConfig(uid, "minimal-alert-enrichment").
						withDescription("").withAssignStep("5s", map[string]string{
						"minimal": "true",
					}).
						build(),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "minimal-alert-enrichment"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "0"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", "0"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.annotation_matchers.#", "0"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.annotations.%", "1"),
						resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.assign_step.0.timeout", "5s"),
						resource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				{
					ResourceName:      alertEnrichmentResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
				},
			},
		})
	})
}

func TestAccAlertEnrichment_matchers(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	testCases := []struct {
		name          string
		labelMatchers []matcherConfig
		annotMatchers []matcherConfig
		expectedTitle string
	}{
		{
			name: "single_equal_matcher",
			labelMatchers: []matcherConfig{
				{"=", "severity", "critical"},
			},
			expectedTitle: "Single Equal Matcher",
		},
		{
			name: "regex_matchers",
			labelMatchers: []matcherConfig{
				{"=~", "service", "api-.*"},
			},
			expectedTitle: "Regex Matchers",
		},
		{
			name: "negative_matchers",
			labelMatchers: []matcherConfig{
				{"!=", "environment", "test"},
				{"!~", "service", "temp-.*"},
			},
			expectedTitle: "Negative Matchers",
		},
		{
			name: "multiple_same_type_matchers",
			labelMatchers: []matcherConfig{
				{"=", "severity", "critical"},
				{"=", "team", "platform"},
			},
			expectedTitle: "Multiple Same Type Matchers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			randSuffix := acctest.RandString(6)

			builder := newAlertEnrichmentConfig(
				fmt.Sprintf("matcher-%s-%s", tc.name, randSuffix),
				tc.expectedTitle,
			).withDescription("").withAssignStep("5s", map[string]string{
				"matched_by": tc.name,
			})

			for _, matcher := range tc.labelMatchers {
				builder = builder.withLabelMatcher(matcher.matchType, matcher.name, matcher.value)
			}

			for _, matcher := range tc.annotMatchers {
				builder = builder.withAnnotationMatcher(matcher.matchType, matcher.name, matcher.value)
			}

			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: builder.build(),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", tc.expectedTitle),
							resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", fmt.Sprintf("%d", len(tc.labelMatchers))),
							resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.annotation_matchers.#", fmt.Sprintf("%d", len(tc.annotMatchers))),
							resource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
						),
					},
					{
						ResourceName:      alertEnrichmentResourceName,
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
					},
				},
			})
		})
	}
}

func TestAccAlertEnrichment_alertRulesAndReceivers(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	randSuffix := acctest.RandString(6)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("rules-receivers-%s", randSuffix),
					"Rules and Receivers Enrichment",
				).withDescription("Enrichment for specific alert rules and receivers").
					withAlertRuleUIDs("rule-uid-1", "rule-uid-2").
					withReceivers("pagerduty", "slack-critical").
					withAssignStep("0s", map[string]string{
						"filtered": "true",
					}).
					build(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Rules and Receivers Enrichment"),
					resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "2"),
					resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.0", "rule-uid-1"),
					resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.1", "rule-uid-2"),
					resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.#", "2"),
					resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.0", "pagerduty"),
					resource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.1", "slack-critical"),
					resource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_invalidMatchers(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	randSuffix := acctest.RandString(6)

	testCases := []struct {
		name          string
		matcherType   string
		matcherName   string
		matcherValue  string
		expectedError string
	}{
		{
			name:          "invalid_matcher_type",
			matcherType:   "invalid",
			matcherName:   "severity",
			matcherValue:  "critical",
			expectedError: "invalid matcher type",
		},
		{
			name:          "empty_matcher_name",
			matcherType:   "=",
			matcherName:   "",
			matcherValue:  "critical",
			expectedError: "matcher 'type' and 'name' must be set",
		},
		{
			name:          "empty_matcher_type",
			matcherType:   "",
			matcherName:   "severity",
			matcherValue:  "critical",
			expectedError: "matcher 'type' and 'name' must be set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: newAlertEnrichmentConfig(
							fmt.Sprintf("invalid-matcher-%s-%s", tc.name, randSuffix),
							"Invalid Matcher Test",
						).withLabelMatcher(tc.matcherType, tc.matcherName, tc.matcherValue).
							withAssignStep("10s", map[string]string{
								"test": "invalid-matcher",
							}).
							build(),
						ExpectError: regexp.MustCompile(tc.expectedError),
					},
				},
			})
		})
	}
}

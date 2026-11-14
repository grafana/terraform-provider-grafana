package incident_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

const customFieldResourceName = "grafana_incident_custom_field.test"

func TestAccIncidentCustomField_basic(t *testing.T) {
	testutils.CheckIncidentTestsEnabled(t)

	// Captured during the run so CheckDestroy can look the field up after the
	// resource is gone from state.
	var fieldID string

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkIncidentCustomFieldDestroyed(&fieldID),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_custom_field/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					captureIncidentRoleID(customFieldResourceName, &fieldID),
					resource.TestCheckResourceAttr(customFieldResourceName, "name", "tf-acc-test-field"),
					resource.TestCheckResourceAttr(customFieldResourceName, "slug", "tf_acc_test_field"),
					resource.TestCheckResourceAttr(customFieldResourceName, "type", "string"),
					resource.TestCheckResourceAttr(customFieldResourceName, "description", "Terraform acceptance test field."),
					resource.TestCheckResourceAttr(customFieldResourceName, "required", "false"),
					resource.TestCheckResourceAttr(customFieldResourceName, "select_option.#", "0"),
					resource.TestCheckResourceAttr(customFieldResourceName, "select_option_ids.%", "0"),
					testutils.CheckLister(customFieldResourceName),
				),
			},
			{
				ResourceName:      customFieldResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// A rename and a slug change update the field in place.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_custom_field/_acc_basic.tf", map[string]string{
					"tf-acc-test-field": "tf-acc-test-field-updated",
					"tf_acc_test_field": "tf_acc_test_field_updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "name", "tf-acc-test-field-updated"),
					resource.TestCheckResourceAttr(customFieldResourceName, "slug", "tf_acc_test_field_updated"),
					checkIncidentCustomFieldKeptID(&fieldID),
				),
			},
			// A type change replaces the field. The test's field is unused, so
			// the API allows the delete half of the replacement. Each step
			// replaces it with the next type, so every non-select type is
			// created once.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_custom_field/_acc_basic.tf", map[string]string{
					`"string"`: `"number"`,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "type", "number"),
					checkIncidentCustomFieldReplaced(&fieldID),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_custom_field/_acc_basic.tf", map[string]string{
					`"string"`: `"bool"`,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "type", "bool"),
					checkIncidentCustomFieldReplaced(&fieldID),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_custom_field/_acc_basic.tf", map[string]string{
					`"string"`: `"date"`,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "type", "date"),
					checkIncidentCustomFieldReplaced(&fieldID),
				),
			},
			{
				ResourceName:      customFieldResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// checkIncidentCustomFieldReplaced checks that the custom field got a new ID,
// so the change replaced it, and captures the new ID.
func checkIncidentCustomFieldReplaced(fieldID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if id := s.RootModule().Resources[customFieldResourceName].Primary.ID; id == *fieldID {
			return fmt.Errorf("the custom field kept its ID %q, so it was not replaced", id)
		}
		return captureIncidentRoleID(customFieldResourceName, fieldID)(s)
	}
}

func TestAccIncidentCustomField_selectOptions(t *testing.T) {
	testutils.CheckIncidentTestsEnabled(t)

	var fieldID, lowOptionID string

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkIncidentCustomFieldDestroyed(&fieldID),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_custom_field/_acc_select.tf"),
				Check: resource.ComposeTestCheckFunc(
					captureIncidentRoleID(customFieldResourceName, &fieldID),
					resource.TestCheckResourceAttr(customFieldResourceName, "type", "single-select"),
					resource.TestCheckResourceAttr(customFieldResourceName, "select_option.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{
						"value": "low",
						"label": "Low",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{
						"value":       "high",
						"label":       "High",
						"description": "Terraform acceptance test option.",
					}),
					resource.TestCheckResourceAttr(customFieldResourceName, "select_option_ids.%", "2"),
					captureAttr(customFieldResourceName, "select_option_ids.low", &lowOptionID),
				),
			},
			{
				ResourceName:      customFieldResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// A label change updates the option in place, so it keeps its UUID.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_custom_field/_acc_select.tf", map[string]string{
					`label = "Low"`: `label = "Low impact"`,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{
						"value": "low",
						"label": "Low impact",
					}),
					resource.TestCheckResourceAttrPtr(customFieldResourceName, "select_option_ids.low", &lowOptionID),
				),
			},
			// Adding an option keeps the others' UUIDs, and removing one drops it.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_custom_field/_acc_select.tf", map[string]string{
					`"high"`: `"medium"`,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "select_option.#", "2"),
					resource.TestCheckResourceAttr(customFieldResourceName, "select_option_ids.%", "2"),
					resource.TestCheckNoResourceAttr(customFieldResourceName, "select_option_ids.high"),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{
						"value":       "medium",
						"description": "Terraform acceptance test option.",
					}),
					resource.TestCheckResourceAttrPtr(customFieldResourceName, "select_option_ids.low", &lowOptionID),
					resource.TestCheckResourceAttrSet(customFieldResourceName, "select_option_ids.medium"),
					checkIncidentCustomFieldKeptID(&fieldID),
				),
			},
		},
	})
}

func TestAccIncidentCustomField_archived(t *testing.T) {
	testutils.CheckIncidentTestsEnabled(t)

	var fieldID string

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkIncidentCustomFieldDestroyed(&fieldID),
		Steps: []resource.TestStep{
			{
				Config: customFieldArchivedConfig("", "", true),
				Check: resource.ComposeTestCheckFunc(
					captureIncidentRoleID(customFieldResourceName, &fieldID),
					resource.TestCheckResourceAttr(customFieldResourceName, "archived", "false"),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{"value": "low", "archived": "false"}),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{"value": "high", "archived": "false"}),
				),
			},
			// Archiving an option updates it in place.
			{
				Config: customFieldArchivedConfig("", "true", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{"value": "low", "archived": "true"}),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{"value": "high", "archived": "false"}),
					checkIncidentCustomFieldKeptID(&fieldID),
				),
			},
			// archived defaults to false, so an option archived outside
			// Terraform shows as drift.
			{
				PreConfig:          func() { archiveIncidentCustomFieldOption(t, fieldID, "high") },
				Config:             customFieldArchivedConfig("", "true", true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Archived options are imported with their flag.
			{
				ResourceName:      customFieldResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// The apply reverts the archive made outside Terraform.
			{
				Config: customFieldArchivedConfig("true", "true", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "archived", "true"),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{"value": "high", "archived": "false"}),
					checkIncidentCustomFieldKeptID(&fieldID),
				),
			},
			// Leaving archived out unarchives the field and the option.
			{
				Config: customFieldArchivedConfig("", "", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "archived", "false"),
					resource.TestCheckTypeSetElemNestedAttrs(customFieldResourceName, "select_option.*", map[string]string{"value": "low", "archived": "false"}),
					checkIncidentCustomFieldKeptID(&fieldID),
				),
			},
			// Removing an archived option deletes it, as UpdateField deletes the
			// options it is not sent, archived or not.
			{
				PreConfig: func() { archiveIncidentCustomFieldOption(t, fieldID, "high") },
				Config:    customFieldArchivedConfig("", "", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(customFieldResourceName, "select_option.#", "1"),
					resource.TestCheckNoResourceAttr(customFieldResourceName, "select_option_ids.high"),
					checkIncidentCustomFieldOptions(&fieldID, "low"),
				),
			},
		},
	})
}

// customFieldArchivedConfig is a select field with a low and, if withHigh, a
// high option. fieldArchived and lowArchived set the archived flags, and are
// left unset when "".
func customFieldArchivedConfig(fieldArchived, lowArchived string, withHigh bool) string {
	archived := func(value string) string {
		if value == "" {
			return ""
		}
		return "archived = " + value
	}
	high := ""
	if withHigh {
		high = `
  select_option {
    value = "high"
    label = "High"
  }`
	}
	return fmt.Sprintf(`
resource "grafana_incident_custom_field" "test" {
  name = "tf-acc-test-archived"
  slug = "tf_acc_test_archived"
  type = "single-select"
  %s

  select_option {
    value = "low"
    label = "Low"
    %s
  }
%s
}
`, archived(fieldArchived), archived(lowArchived), high)
}

// archiveIncidentCustomFieldOption archives an option the way the UI does,
// with an UpdateField that sends the rest of the field unchanged.
func archiveIncidentCustomFieldOption(t *testing.T, fieldID, value string) {
	t.Helper()
	ctx := context.Background()
	fields := incident.NewFieldsService(testutils.Provider.Meta().(*common.Client).IncidentClient)
	got, err := fields.GetField(ctx, incident.GetFieldRequest{FieldUUID: fieldID, IncludeArchived: true})
	if err != nil {
		t.Fatalf("reading incident custom field %s: %s", fieldID, err)
	}
	field := got.Field
	for i := range field.Selectoptions {
		if field.Selectoptions[i].Value == value {
			field.Selectoptions[i].Archived = true
		}
	}
	_, err = fields.UpdateField(ctx, incident.UpdateFieldRequest{
		FieldUUID:     field.UUID,
		Name:          field.Name,
		Slug:          field.Slug,
		Color:         field.Color,
		Icon:          field.Icon,
		Description:   field.Description,
		Required:      field.Required,
		DomainName:    field.DomainName,
		Immutable:     field.Immutable,
		Selectoptions: field.Selectoptions,
		Version:       field.Version,
	})
	if err != nil {
		t.Fatalf("archiving option %q of incident custom field %s: %s", value, fieldID, err)
	}
}

// checkIncidentCustomFieldOptions asserts the field's options in the API,
// archived ones included, have exactly the values.
func checkIncidentCustomFieldOptions(fieldID *string, values ...string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).IncidentClient
		got, err := incident.NewFieldsService(client).GetField(context.Background(), incident.GetFieldRequest{FieldUUID: *fieldID, IncludeArchived: true})
		if err != nil {
			return fmt.Errorf("reading incident custom field %s: %w", *fieldID, err)
		}
		var stored []string
		for _, option := range got.Field.Selectoptions {
			stored = append(stored, option.Value)
		}
		if !slices.Equal(stored, values) {
			return fmt.Errorf("incident custom field %s has options %v, want %v", *fieldID, stored, values)
		}
		return nil
	}
}

// captureAttr records an attribute from Terraform state for a later step.
func captureAttr(resourceName, attribute string, dest *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		value := rs.Primary.Attributes[attribute]
		if value == "" {
			return fmt.Errorf("resource %s has no %s attribute", resourceName, attribute)
		}
		*dest = value
		return nil
	}
}

// checkIncidentCustomFieldKeptID asserts the field was updated in place rather
// than replaced.
func checkIncidentCustomFieldKeptID(fieldID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if id := s.RootModule().Resources[customFieldResourceName].Primary.ID; id != *fieldID {
			return fmt.Errorf("the custom field's ID changed from %q to %q, so it was replaced", *fieldID, id)
		}
		return nil
	}
}

// checkIncidentCustomFieldDestroyed asserts the field is really gone from the
// API.
func checkIncidentCustomFieldDestroyed(fieldID *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if *fieldID == "" {
			return errors.New("no incident custom field ID was captured during the test")
		}
		client := testutils.Provider.Meta().(*common.Client).IncidentClient
		_, err := incident.NewFieldsService(client).GetField(context.Background(), incident.GetFieldRequest{FieldUUID: *fieldID})
		if err == nil {
			return fmt.Errorf("incident custom field %s still exists after destroy", *fieldID)
		}
		if !incident.HasErrorCode(err, incident.ErrFieldNotFound) {
			return fmt.Errorf("reading incident custom field %s: %w", *fieldID, err)
		}
		return nil
	}
}

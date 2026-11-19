package incident_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccIncidentSeverity_basic(t *testing.T) {
	testutils.CheckIncidentTestsEnabled(t)

	// Captured during the run so CheckDestroy can look the severity up after
	// the resource is gone from state.
	var severityID string

	// Not parallel: an organization has only 5 levels to create severities
	// at, so concurrent severity tests would conflict over them.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutils.ProtoV6ProviderFactories,
		CheckDestroy:             checkIncidentSeverityDestroyed(&severityID),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_severity/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					captureIncidentRoleID(severityResourceName, &severityID),
					resource.TestCheckResourceAttr(severityResourceName, "label", "tf-acc-test-severity"),
					resource.TestCheckResourceAttr(severityResourceName, "level", "5"),
					resource.TestCheckResourceAttr(severityResourceName, "description", "Terraform acceptance test severity."),
					resource.TestCheckResourceAttr(severityResourceName, "archived", "false"),
					testutils.CheckLister(severityResourceName),
				),
			},
			{
				ResourceName:      severityResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_severity/_acc_basic.tf", map[string]string{
					"tf-acc-test-severity": "tf-acc-test-severity-updated",
				}),
				Check: resource.TestCheckResourceAttr(severityResourceName, "label", "tf-acc-test-severity-updated"),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_severity/_acc_archived.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "archived", "true"),
					resource.TestCheckResourceAttr(severityResourceName, "label", "tf-acc-test-severity"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_incident_severity/_acc_basic.tf"),
				Check:  resource.TestCheckResourceAttr(severityResourceName, "archived", "false"),
			},
			// A level change replaces the severity. The test's severity is new
			// and unused, so the delete half of the replacement is allowed.
			// Level 4 is the one level a default organization leaves free
			// besides the 5 the test starts at.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_incident_severity/_acc_basic.tf", map[string]string{
					"level       = 5": "level       = 4",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "level", "4"),
					func(s *terraform.State) error {
						if id := s.RootModule().Resources[severityResourceName].Primary.ID; id == severityID {
							return fmt.Errorf("the severity kept its ID %q, so it was not replaced", id)
						}
						return captureIncidentRoleID(severityResourceName, &severityID)(s)
					},
				),
			},
		},
	})
}

// TestAccIncidentSeverity_pending imports the organization's Pending severity,
// edits it, puts it back as it was, and destroys it, which must leave it in
// place.
func TestAccIncidentSeverity_pending(t *testing.T) {
	testutils.CheckIncidentTestsEnabled(t)

	client := incidentClientFromEnv(t)
	var pending *incident.Severity
	for _, severity := range listIncidentSeverities(t, client) {
		if severity.Level == -1 {
			pending = &severity
			break
		}
	}
	if pending == nil {
		t.Skip("the organization has no Pending severity")
	}

	// The test renames the organization's real Pending severity, and only its
	// last step puts it back. If a step in between fails, restore it here so
	// the stack is not left with the test's label, which a later run would
	// then capture as the original.
	t.Cleanup(func() {
		severities := incident.NewSeveritiesService(client)
		got, err := severities.GetSeverityByID(context.Background(), incident.GetSeverityByIDRequest{SeverityID: pending.SeverityID})
		if err != nil {
			t.Errorf("reading the Pending severity to restore it: %s", err)
			return
		}
		if got.Severity.DisplayLabel == pending.DisplayLabel && got.Severity.Description == pending.Description {
			return
		}
		if _, err := severities.UpdateOrgSeverity(context.Background(), incident.UpdateOrgSeverityRequest{Severity: *pending}); err != nil {
			t.Errorf("restoring the Pending severity to label %q: %s", pending.DisplayLabel, err)
		}
	})

	config := func(label, description string) string {
		return fmt.Sprintf(`
resource "grafana_incident_severity" "test" {
  label       = %q
  level       = -1
  description = %q
  icon        = %q
  dark_color  = %q
  light_color = %q
}
`, label, description, pending.IconName, pending.DarkColor, pending.LightColor)
	}
	original := config(pending.DisplayLabel, pending.Description)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutils.ProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			for _, severity := range listIncidentSeverities(t, client) {
				if severity.SeverityID == pending.SeverityID {
					if severity.ArchivedAt != "" {
						return errors.New("the Pending severity was archived")
					}
					return nil
				}
			}
			return errors.New("destroying the Pending severity deleted it")
		},
		Steps: []resource.TestStep{
			{
				Config:             original,
				ResourceName:       severityResourceName,
				ImportState:        true,
				ImportStateId:      pending.SeverityID,
				ImportStatePersist: true,
			},
			{
				Config: config("tf-acc-test-pending", "Terraform acceptance test."),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "id", pending.SeverityID),
					resource.TestCheckResourceAttr(severityResourceName, "label", "tf-acc-test-pending"),
				),
			},
			{
				Config: original,
				Check:  resource.TestCheckResourceAttr(severityResourceName, "label", pending.DisplayLabel),
			},
		},
	})
}

// checkIncidentSeverityDestroyed asserts the severity is really gone from the
// API.
func checkIncidentSeverityDestroyed(severityID *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if *severityID == "" {
			return errors.New("no incident severity ID was captured during the test")
		}
		client := testutils.Provider.Meta().(*common.Client).IncidentClient
		_, err := incident.NewSeveritiesService(client).GetSeverityByID(context.Background(), incident.GetSeverityByIDRequest{SeverityID: *severityID})
		if err == nil {
			return fmt.Errorf("incident severity %s still exists after destroy", *severityID)
		}
		if !incident.HasErrorCode(err, incident.ErrSeverityNotFound) {
			return fmt.Errorf("reading incident severity %s: %w", *severityID, err)
		}
		return nil
	}
}

func listIncidentSeverities(t *testing.T, client *incident.Client) []incident.Severity {
	t.Helper()

	resp, err := incident.NewSeveritiesService(client).GetOrgSeverities(context.Background(), incident.GetOrgSeveritiesRequest{})
	if err != nil {
		t.Fatalf("listing incident severities: %s", err)
	}
	return resp.Severities
}

package incident_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

const severityResourceName = "grafana_incident_severity.test"

// severityConfig renders the severity under test with the given extra lines.
func severityConfig(label string, level int, extra string) string {
	return fmt.Sprintf(`
resource "grafana_incident_severity" "test" {
  label = %q
  level = %d
  %s
}
`, label, level, extra)
}

var (
	activeSeverityConfig   = severityConfig("tf-unit-test-severity", 5, "")
	archivedSeverityConfig = severityConfig("tf-unit-test-severity", 5, "archived = true")
)

// TestUnitIncidentSeverity_UnarchiveRefused pins that a refused unarchive
// leaves the severity archived rather than carrying on to UpdateOrgSeverity.
func TestUnitIncidentSeverity_UnarchiveRefused(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: archivedSeverityConfig,
				Check:  resource.TestCheckResourceAttr(severityResourceName, "archived", "true"),
			},
			{
				PreConfig:   func() { api.setRefuseUnarchive(true) },
				Config:      activeSeverityConfig,
				ExpectError: regexp.MustCompile("unarchive refused by the test"),
			},
			{
				PreConfig: func() { api.setRefuseUnarchive(false) },
				Config:    archivedSeverityConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "archived", "true"),
					func(_ *terraform.State) error {
						severity := api.severity("tf-unit-test-severity")
						if severity == nil {
							return errors.New("the severity under test no longer exists")
						}
						if severity.ArchivedAt == "" {
							return errors.New("the severity was unarchived even though the API refused")
						}
						if api.called("SeveritiesService.UpdateOrgSeverity") {
							return errors.New("UpdateOrgSeverity was called for an unarchive the API had already refused")
						}
						return nil
					},
				),
			},
		},
	})
}

// TestUnitIncidentSeverity_UpdateResponseOmitsArchivedAt pins that archived
// is taken from the plan after an archive, not from the UpdateOrgSeverity
// response, which need not carry archivedAt.
func TestUnitIncidentSeverity_UpdateResponseOmitsArchivedAt(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.setUpdateOmitsArchivedAt(true)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: activeSeverityConfig},
			{
				Config: archivedSeverityConfig,
				Check:  resource.TestCheckResourceAttr(severityResourceName, "archived", "true"),
			},
		},
	})
}

// TestUnitIncidentSeverity_ArchiveUsesDedicatedEndpoints pins that archived
// transitions go through ArchiveOrgSeverity and UnarchiveOrgSeverity, before
// UpdateOrgSeverity writes the rest. It also pins that there is no lower bound:
// the organization holds only Pending and the severity under test, and
// archiving it still succeeds.
func TestUnitIncidentSeverity_ArchiveUsesDedicatedEndpoints(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	renamedArchived := severityConfig("tf-unit-test-severity-renamed", 5, "archived = true")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: activeSeverityConfig},
			{
				Config: renamedArchived,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "archived", "true"),
					resource.TestCheckResourceAttr(severityResourceName, "label", "tf-unit-test-severity-renamed"),
					func(_ *terraform.State) error {
						if got := api.activeCount(); got != 1 {
							return fmt.Errorf("expected only Pending to remain active, got %d active severities", got)
						}
						return nil
					},
				),
			},
			{
				// An update that leaves archived alone must not unarchive.
				Config: severityConfig("tf-unit-test-severity", 5, "archived = true"),
				Check:  resource.TestCheckResourceAttr(severityResourceName, "archived", "true"),
			},
			{
				Config: activeSeverityConfig,
				Check:  resource.TestCheckResourceAttr(severityResourceName, "archived", "false"),
			},
		},
	})

	archive, unarchive, update := api.firstCall("SeveritiesService.ArchiveOrgSeverity"), api.firstCall("SeveritiesService.UnarchiveOrgSeverity"), api.firstCall("SeveritiesService.UpdateOrgSeverity")
	if archive < 0 {
		t.Error("archiving the severity did not call ArchiveOrgSeverity")
	}
	if unarchive < 0 {
		t.Error("unarchiving the severity did not call UnarchiveOrgSeverity")
	}
	if update < 0 {
		t.Error("the update did not call UpdateOrgSeverity")
	}
	if archive >= 0 && update >= 0 && archive > update {
		t.Error("UpdateOrgSeverity ran before ArchiveOrgSeverity")
	}
}

// TestUnitIncidentSeverity_CreateArchived pins that a severity configured as
// archived from the start is created and then archived, and that a refused
// archive leaves it tracked rather than orphaned: the retry must not hit a
// label conflict with a severity Terraform lost track of.
func TestUnitIncidentSeverity_CreateArchived(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.setRefuseArchive(true)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      archivedSeverityConfig,
				ExpectError: regexp.MustCompile("archive refused by the test"),
			},
			{
				PreConfig: func() { api.setRefuseArchive(false) },
				Config:    archivedSeverityConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "archived", "true"),
					func(_ *terraform.State) error {
						count := 0
						for _, severity := range api.snapshot() {
							if severity.DisplayLabel == "tf-unit-test-severity" {
								count++
							}
						}
						if count != 1 {
							return fmt.Errorf("expected one severity under test, got %d", count)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestUnitIncidentSeverity_LevelChangeReplaces pins that a level change
// replaces the severity. The API ignores the level on update, so an in-place
// update would leave a permanent diff.
func TestUnitIncidentSeverity_LevelChangeReplaces(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	var firstID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: activeSeverityConfig,
				Check:  captureIncidentRoleID(severityResourceName, &firstID),
			},
			{
				Config: severityConfig("tf-unit-test-severity", 4, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "level", "4"),
					func(s *terraform.State) error {
						if id := s.RootModule().Resources[severityResourceName].Primary.ID; id == firstID {
							return fmt.Errorf("the severity kept its ID %q, so it was not replaced", id)
						}
						severity := api.severity("tf-unit-test-severity")
						if severity == nil || severity.Level != 4 {
							return fmt.Errorf("expected the severity at level 4, got %+v", severity)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestUnitIncidentSeverity_LevelOutOfRange pins the levels the API accepts:
// 1 to 5, plus -1 for the Pending severity. Level 0 is refused at plan time
// rather than by the API.
func TestUnitIncidentSeverity_LevelOutOfRange(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	for _, level := range []int{0, 6, -2} {
		resource.UnitTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      severityConfig("tf-unit-test-severity", level, ""),
					ExpectError: regexp.MustCompile(`between\s+1\s+and\s+5`),
				},
			},
		})
	}

	if api.called("SeveritiesService.CreateOrgSeverity") {
		t.Error("an out-of-range level reached the API")
	}
}

// TestUnitIncidentSeverity_LevelChangeToTakenLevel pins that a level change to
// a level another severity holds, archived ones included, is refused at plan
// time. The replacement deletes before it creates, so leaving the conflict to
// the API would delete the severity and fail to create its replacement.
func TestUnitIncidentSeverity_LevelChangeToTakenLevel(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.add(incident.Severity{DisplayLabel: "retired", Level: 4, ArchivedAt: "2026-01-01T00:00:00Z"})
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: activeSeverityConfig,
			},
			{
				Config:      severityConfig("tf-unit-test-severity", 4, ""),
				ExpectError: regexp.MustCompile(`Level 4 is held by severity "retired", which is archived`),
			},
			// Checked before the test's destroy, which deletes the severity.
			{
				Config: activeSeverityConfig,
				Check:  checkSeverityNotDeleted(api),
			},
		},
	})
}

// TestUnitIncidentSeverity_LevelChangeToTakenLabel pins the same plan-time
// check for the label a replacement is created with.
func TestUnitIncidentSeverity_LevelChangeToTakenLabel(t *testing.T) {
	api := newFakeSeveritiesAPI(activeSeverities("critical")...)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: activeSeverityConfig,
			},
			{
				Config:      severityConfig("critical", 4, ""),
				ExpectError: regexp.MustCompile(`Label "critical" is held by the severity at level 1`),
			},
			// Checked before the test's destroy, which deletes the severity.
			{
				Config: activeSeverityConfig,
				Check:  checkSeverityNotDeleted(api),
			},
		},
	})
}

// checkSeverityNotDeleted asserts that no severity has been deleted, and that
// the one under test is still in place.
func checkSeverityNotDeleted(api *fakeSeveritiesAPI) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if api.called("SeveritiesService.DeleteOrgSeverity") {
			return errors.New("the refused level change deleted a severity")
		}
		if severity := api.severity("tf-unit-test-severity"); severity == nil || severity.Level != 5 {
			return fmt.Errorf("expected the severity to be left at level 5, got %+v", severity)
		}
		return nil
	}
}

// TestUnitIncidentSeverity_LabelConflictWithArchived pins that an archived
// severity still holds its label, and that the API's conflict reaches the
// practitioner.
func TestUnitIncidentSeverity_LabelConflictWithArchived(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.add(incident.Severity{DisplayLabel: "tf-unit-test-severity", Level: 3, ArchivedAt: "2026-01-01T00:00:00Z"})
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      activeSeverityConfig,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(errSeverityLabelConflict)),
			},
		},
	})
}

// TestUnitIncidentSeverity_DeleteRefusedWhenInUse pins the destroy of a
// severity incidents use: the API's refusal is explained with the incident
// count and what to do instead, and the severity stays in state.
func TestUnitIncidentSeverity_DeleteRefusedWhenInUse(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: activeSeverityConfig,
				Check: func(_ *terraform.State) error {
					api.setInUse("tf-unit-test-severity", 3)
					return nil
				},
			},
			{
				Config:      activeSeverityConfig,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`(?s)used by 3 incidents.*archived = true`),
			},
			{
				Config: activeSeverityConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(severityResourceName, "id"),
					func(_ *terraform.State) error {
						if api.severity("tf-unit-test-severity") == nil {
							return errors.New("the in-use severity was deleted")
						}
						// Free it so the test case's own destroy can remove it.
						api.setInUse("tf-unit-test-severity", 0)
						return nil
					},
				),
			},
		},
	})

	if api.severity("tf-unit-test-severity") != nil {
		t.Error("the severity under test was left behind")
	}
}

// TestUnitIncidentSeverity_DeleteToleratesMissingSeverity pins that a 404 on
// delete is a successful destroy.
func TestUnitIncidentSeverity_DeleteToleratesMissingSeverity(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: activeSeverityConfig,
				Check: func(_ *terraform.State) error {
					api.setDeleteNotFound(true)
					return nil
				},
			},
		},
	})

	if !api.called("SeveritiesService.DeleteOrgSeverity") {
		t.Error("the destroy did not call DeleteOrgSeverity, so the missing-severity path was not exercised")
	}
}

// TestUnitIncidentSeverity_Pending pins how the system-managed Pending severity
// is handled: it can be imported and edited, archiving it, changing its level,
// or creating a second one is refused at plan time, and destroying it leaves it
// in place.
func TestUnitIncidentSeverity_Pending(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)
	pendingID := api.pending().SeverityID

	pendingConfig := func(label, extra string) string {
		return severityConfig(label, pendingLevel, `icon = "question-circle"
  `+extra)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      pendingConfig("Pending", ""),
				ExpectError: regexp.MustCompile("A severity with level -1 cannot be created"),
			},
			{
				Config:             pendingConfig("Pending", ""),
				ResourceName:       severityResourceName,
				ImportState:        true,
				ImportStateId:      pendingID,
				ImportStatePersist: true,
			},
			{
				Config: pendingConfig("Triage", `description = "Being triaged."`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(severityResourceName, "id", pendingID),
					resource.TestCheckResourceAttr(severityResourceName, "label", "Triage"),
					resource.TestCheckResourceAttr(severityResourceName, "description", "Being triaged."),
				),
			},
			{
				Config:      pendingConfig("Triage", "archived = true"),
				ExpectError: regexp.MustCompile("The Pending severity cannot be archived"),
			},
			{
				Config:      severityConfig("Triage", 5, `icon = "question-circle"`),
				ExpectError: regexp.MustCompile("The Pending severity's level cannot change"),
			},
		},
	})

	pending := api.pending()
	if pending == nil {
		t.Fatal("destroying the Pending severity deleted it")
	}
	if pending.ArchivedAt != "" {
		t.Error("the Pending severity was archived")
	}
	if api.called("SeveritiesService.DeleteOrgSeverity") || api.called("SeveritiesService.ArchiveOrgSeverity") {
		t.Error("the provider asked the API to delete or archive the Pending severity")
	}
}

// TestUnitIncidentSeverity_LevelChangeToPending pins that moving an existing
// severity to level -1 gets the same plan-time refusal as creating one there,
// rather than a level conflict that suggests deleting Pending.
func TestUnitIncidentSeverity_LevelChangeToPending(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: activeSeverityConfig},
			{
				Config:      severityConfig("tf-unit-test-severity", pendingLevel, ""),
				ExpectError: regexp.MustCompile("A severity with level -1 cannot be created"),
			},
		},
	})
}

// TestUnitIncidentSeverity_EmptyOptionalStrings pins the "" versus null mapping
// for every optional string. The API stores an unset one as "", so getting it
// wrong fails the apply with an inconsistent result.
func TestUnitIncidentSeverity_EmptyOptionalStrings(t *testing.T) {
	api := newFakeSeveritiesAPI()
	api.start(t)

	attrs := []string{"description", "icon", "dark_color", "light_color"}
	setAll := func(value string) string {
		extra := ""
		for _, attr := range attrs {
			extra += fmt.Sprintf("%s = %q\n  ", attr, value)
		}
		return severityConfig("tf-unit-test-severity", 5, extra)
	}
	checkAll := func(value *string) resource.TestCheckFunc {
		checks := make([]resource.TestCheckFunc, 0, len(attrs))
		for _, attr := range attrs {
			if value == nil {
				checks = append(checks, resource.TestCheckNoResourceAttr(severityResourceName, attr))
			} else {
				checks = append(checks, resource.TestCheckResourceAttr(severityResourceName, attr, *value))
			}
		}
		return resource.ComposeTestCheckFunc(checks...)
	}
	empty, set := "", "rgb(1, 2, 3)"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: setAll(empty), Check: checkAll(&empty)},
			{Config: setAll(set), Check: checkAll(&set)},
			{Config: setAll(empty), Check: checkAll(&empty)},
			{Config: activeSeverityConfig, Check: checkAll(nil)},
			{
				ResourceName:      severityResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

package incident_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// Configs for the role under test. The bounds are about how many active roles
// the organization has, so every test below only varies the archived flag and
// how many roles exist alongside it.
const (
	roleConfig = `
resource "grafana_incident_role" "test" {
  name        = "tf-unit-test-role"
  description = "Terraform unit test role."
}
`

	archivedRoleConfig = `
resource "grafana_incident_role" "test" {
  name        = "tf-unit-test-role"
  description = "Terraform unit test role."
  archived    = true
}
`
)

// TestUnitIncidentRole_CreateRefusedAtActiveCeiling pins the 5-active ceiling on
// create: an organization that is already full cannot take another role, and the
// API's error reaches the practitioner.
func TestUnitIncidentRole_CreateRefusedAtActiveCeiling(t *testing.T) {
	api := newFakeRolesAPI(activeRoles("commander", "investigator", "observer", "scribe", "comms")...)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      roleConfig,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(errActiveCeiling)),
			},
		},
	})

	if got := len(api.snapshot()); got != maxActiveRoles {
		t.Errorf("expected the organization to still hold %d roles, got %d", maxActiveRoles, got)
	}
}

// TestUnitIncidentRole_UnarchiveRefusedAtActiveCeiling pins the same ceiling on
// the other side of the archived flag. Unarchiving is the failure a practitioner
// hits after a plan that looks like a plain attribute change, so the resource
// must leave the role archived rather than write the column with UpdateRole,
// which the API does not bound-check.
func TestUnitIncidentRole_UnarchiveRefusedAtActiveCeiling(t *testing.T) {
	// Four active roles leave exactly one free slot: enough to create the
	// archived role under test, not enough to unarchive it once the slot is
	// taken by someone else.
	api := newFakeRolesAPI(activeRoles("commander", "investigator", "observer", "scribe")...)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: archivedRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "true"),
				),
			},
			{
				// The organization fills up outside Terraform, so the plan to
				// unarchive is only invalid by the time it is applied.
				PreConfig:   func() { api.add(incident.Role{Name: "comms"}) },
				Config:      roleConfig,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(errActiveCeiling)),
			},
			{
				// Asserted in a step rather than after the test case, because the
				// case's own destroy removes the role on the way out.
				Config: archivedRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "true"),
					func(_ *terraform.State) error {
						role := api.role("tf-unit-test-role")
						if role == nil {
							return errors.New("the role under test no longer exists")
						}
						if !role.Archived {
							return errors.New("the role was unarchived even though the API refused: archived must not be written with UpdateRole")
						}
						if api.called("RolesService.UpdateRole") {
							return errors.New("UpdateRole was called for an archived transition the API had already refused")
						}
						if got := api.activeCount(); got != maxActiveRoles {
							return fmt.Errorf("expected %d active roles, got %d", maxActiveRoles, got)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestUnitIncidentRole_ArchiveAndDeleteRefusedAtActiveFloor pins the 2-active
// floor on both operations that hit it. This is the destroy-time failure the
// issue calls out: the plan is clean, and the apply fails because the
// organization is down to its last two roles.
func TestUnitIncidentRole_ArchiveAndDeleteRefusedAtActiveFloor(t *testing.T) {
	// One pre-existing active role plus the role under test puts the
	// organization exactly at the floor.
	api := newFakeRolesAPI(activeRoles("commander")...)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "false"),
					func(_ *terraform.State) error {
						if got := api.activeCount(); got != minActiveRoles {
							return fmt.Errorf("expected the organization to sit at the floor of %d active roles, got %d", minActiveRoles, got)
						}
						return nil
					},
				),
			},
			{
				Config:      archivedRoleConfig,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(errActiveFloor)),
			},
			{
				Config:      roleConfig,
				Destroy:     true,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(errActiveFloor)),
			},
			{
				// Give the organization headroom again so the test case's own
				// destroy can remove the role it created.
				PreConfig: func() { api.add(incident.Role{Name: "investigator"}) },
				Config:    roleConfig,
			},
		},
	})

	if role := api.role("tf-unit-test-role"); role != nil {
		t.Errorf("the role under test was left behind: %+v", *role)
	}
}

// TestUnitIncidentRole_DeleteToleratesMissingRole pins the delete path for a
// role that no longer exists. Terraform's contract is that Delete succeeds when
// the object is already gone, so the 404 the API answers must not surface as a
// failed destroy. Without it, a role removed outside Terraform would leave a
// resource that can never be destroyed.
func TestUnitIncidentRole_DeleteToleratesMissingRole(t *testing.T) {
	// Three pre-existing active roles keep the organization clear of the floor,
	// so a refused delete here can only be the missing role.
	api := newFakeRolesAPI(activeRoles("commander", "investigator", "observer")...)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_incident_role.test", "id"),
					// The role goes missing after the state is written, so the
					// test case's own destroy still calls DeleteRole and the
					// API answers 404.
					func(_ *terraform.State) error {
						api.setDeleteNotFound(true)
						return nil
					},
				),
			},
		},
	})

	if !api.called("RolesService.DeleteRole") {
		t.Error("the destroy did not call DeleteRole, so the missing-role path was not exercised")
	}
}

// TestUnitIncidentRole_ArchiveUsesDedicatedEndpoints pins how an archived
// transition is applied when it is allowed: through ArchiveRole/UnarchiveRole,
// and before UpdateRole writes the rest of the role. Those endpoints are the
// only ones that enforce the bounds, so reversing the order or dropping them
// would let Terraform push an organization past a limit the API means to hold.
func TestUnitIncidentRole_ArchiveUsesDedicatedEndpoints(t *testing.T) {
	// Two pre-existing active roles leave room to archive the role under test
	// and to unarchive it again.
	api := newFakeRolesAPI(activeRoles("commander", "investigator")...)
	api.start(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleConfig,
			},
			{
				Config: archivedRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "true"),
				),
			},
			{
				Config: roleConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_incident_role.test", "archived", "false"),
				),
			},
		},
	})

	archive, unarchive, update := api.firstCall("RolesService.ArchiveRole"), api.firstCall("RolesService.UnarchiveRole"), api.firstCall("RolesService.UpdateRole")
	if archive < 0 {
		t.Error("archiving the role did not call ArchiveRole")
	}
	if unarchive < 0 {
		t.Error("unarchiving the role did not call UnarchiveRole")
	}
	if update < 0 {
		t.Error("the update did not call UpdateRole")
	}
	if archive >= 0 && update >= 0 && archive > update {
		t.Error("UpdateRole ran before ArchiveRole, so a refused archive would already have been written")
	}
}

// TestAccIncidentRole_activeCeiling pins the 5-active ceiling against a real
// stack. It fills the organization's remaining slots with roles it owns and
// asserts the next role is refused, so it works whatever the organization
// starts with. The floor is not exercised here: reaching it means archiving
// roles the test does not own, which is covered by
// TestUnitIncidentRole_ArchiveAndDeleteRefusedAtActiveFloor and, on the API
// side, by grafana/irm's own e2e tests.
func TestAccIncidentRole_activeCeiling(t *testing.T) {
	testutils.CheckIncidentTestsEnabled(t)

	client := incidentClientFromEnv(t)
	active := countActiveIncidentRoles(t, client)
	headroom := maxActiveRoles - active
	if headroom < 1 {
		t.Skipf("the organization already has %d active incident roles, so the ceiling cannot be approached from below", active)
	}

	// Filler roles are created by the test and removed by its destroy. Deleting
	// them never hits the floor: each delete happens while the organization
	// still holds more than the minimum.
	config := func(count int) string {
		return fmt.Sprintf(`
resource "grafana_incident_role" "filler" {
  count = %d

  name        = "tf-acc-test-ceiling-${count.index}"
  description = "Terraform acceptance test role for the active-role ceiling."
}
`, count)
	}

	// Not parallel: the bound is organization-wide, so a concurrent role test
	// would change the count this test depends on.
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(headroom),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(fmt.Sprintf("grafana_incident_role.filler.%d", headroom-1), "id"),
					func(_ *terraform.State) error {
						if got := countActiveIncidentRoles(t, client); got != maxActiveRoles {
							return fmt.Errorf("expected the organization to sit at the ceiling of %d active roles, got %d", maxActiveRoles, got)
						}
						return nil
					},
				),
			},
			{
				Config:      config(headroom + 1),
				ExpectError: errActiveCeilingPattern,
			},
		},
	})
}

// errActiveCeilingPattern matches the active-role ceiling error from either
// generation of the API: the legacy wording, and the structured wording that
// grafana/irm#11569 introduces. The acceptance test runs against whatever a
// real stack has deployed, so it cannot assume one or the other.
var errActiveCeilingPattern = regexp.MustCompile(
	strings.Join([]string{
		regexp.QuoteMeta("createRole: too many active roles"),
		regexp.QuoteMeta(errActiveCeiling),
	}, "|"),
)

// incidentClientFromEnv builds an Incident client from the acceptance test
// environment. The test needs it before the provider is configured, to count
// the roles the organization already has.
func incidentClientFromEnv(t *testing.T) *incident.Client {
	t.Helper()

	auth := os.Getenv("GRAFANA_AUTH")
	client := incident.NewClient(strings.TrimSuffix(os.Getenv("GRAFANA_URL"), "/")+incidentAPIBasePath, auth)
	if user, password, ok := strings.Cut(auth, ":"); ok {
		client.BeforeRequest = func(r *http.Request) error {
			r.SetBasicAuth(user, password)
			return nil
		}
	}
	return client
}

func countActiveIncidentRoles(t *testing.T, client *incident.Client) int {
	t.Helper()

	roles, err := incident.NewRolesService(client).GetRoles(context.Background(), incident.GetRolesRequest{})
	if err != nil {
		t.Fatalf("listing incident roles: %s", err)
	}
	count := 0
	for _, role := range roles.Roles {
		if !role.Archived {
			count++
		}
	}
	return count
}

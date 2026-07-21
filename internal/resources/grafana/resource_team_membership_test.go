package grafana

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUnitTeamMembershipChanges(t *testing.T) {
	state, err := teamMembershipMap(
		[]string{"promoted@example.com"},
		[]string{"demoted@example.com", "removed@example.com"},
	)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := teamMembershipMap(
		[]string{"demoted@example.com", "new-member@example.com"},
		[]string{"promoted@example.com", "new-admin@example.com"},
	)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]int)
	for _, change := range memberChanges(state, plan) {
		key := fmt.Sprintf("%d:%s:%d", change.Type, change.Member.Email, change.Member.Permission)
		got[key]++
	}
	expected := map[string]int{
		fmt.Sprintf("%d:%s:%d", UpdateMemberPermission, "promoted@example.com", teamMemberPermissionAdmin):  1,
		fmt.Sprintf("%d:%s:%d", UpdateMemberPermission, "demoted@example.com", teamMemberPermissionMember):  1,
		fmt.Sprintf("%d:%s:%d", AddMember, "new-member@example.com", teamMemberPermissionMember):            1,
		fmt.Sprintf("%d:%s:%d", AddMember, "new-admin@example.com", teamMemberPermissionAdmin):              1,
		fmt.Sprintf("%d:%s:%d", UpdateMemberPermission, "new-admin@example.com", teamMemberPermissionAdmin): 1,
		fmt.Sprintf("%d:%s:%d", RemoveMember, "removed@example.com", teamMemberPermissionAdmin):             1,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected membership changes:\n got: %#v\nwant: %#v", got, expected)
	}
}

func TestUnitTeamMembershipMapRejectsOverlappingRoles(t *testing.T) {
	_, err := teamMembershipMap(
		[]string{"both@example.com"},
		[]string{"both@example.com"},
	)
	if err == nil {
		t.Fatal("expected an error when a user is both a member and an admin")
	}
}

func TestUnitRemoveAdminsFromMembers(t *testing.T) {
	members, ignoredAdmins := removeAdminsFromMembers(
		[]string{"member@example.com", "ui-admin@example.com"},
		[]string{"ui-admin@example.com"},
	)
	if !reflect.DeepEqual(ignoredAdmins, []string{"ui-admin@example.com"}) {
		t.Fatalf("unexpected ignored administrators: %#v", ignoredAdmins)
	}
	if !reflect.DeepEqual(members, []string{"member@example.com"}) {
		t.Fatalf("unexpected ordinary members: %#v", members)
	}
}

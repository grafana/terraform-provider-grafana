package incident

import (
	"context"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func customFieldOptionValue(value, label string, description types.String) attr.Value {
	return types.ObjectValueMust(customFieldOptionType.AttrTypes, map[string]attr.Value{
		"value":       types.StringValue(value),
		"label":       types.StringValue(label),
		"description": description,
		"color":       types.StringNull(),
		"icon":        types.StringNull(),
		"archived":    types.BoolValue(false),
	})
}

// TestUnitIncidentCustomField_NoOptions pins that a field without options maps
// to an empty select_option set and an empty select_option_ids map, which is
// what Terraform plans for a configuration without select_option blocks.
func TestUnitIncidentCustomField_NoOptions(t *testing.T) {
	field := incident.CustomMetadataField{UUID: "f", Name: "n", Slug: "s", Type: customFieldTypeString}

	got, diags := customFieldToModel(context.Background(), field, customFieldModel{SelectOptions: types.SetNull(customFieldOptionType)})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if got.SelectOptions.IsNull() || len(got.SelectOptions.Elements()) != 0 {
		t.Errorf("select_option = %s, want an empty set", got.SelectOptions)
	}
	if got.SelectOptionIDs.IsNull() || len(got.SelectOptionIDs.Elements()) != 0 {
		t.Errorf("select_option_ids = %s, want an empty map", got.SelectOptionIDs)
	}
}

// TestUnitIncidentCustomField_OptionOptionalStrings pins that an option's empty
// description stays "" when configured as "", and is null when left out or
// when the option is new to Terraform.
func TestUnitIncidentCustomField_OptionOptionalStrings(t *testing.T) {
	field := incident.CustomMetadataField{
		UUID: "f", Name: "n", Slug: "s", Type: customFieldTypeSingleSelect,
		Selectoptions: []incident.CustomMetadataFieldSelectOption{
			{UUID: "u1", Value: "empty", Label: "Empty"},
			{UUID: "u2", Value: "unset", Label: "Unset"},
			{UUID: "u3", Value: "new", Label: "New"},
		},
	}
	configured := customFieldModel{
		SelectOptions: types.SetValueMust(customFieldOptionType, []attr.Value{
			customFieldOptionValue("empty", "Empty", types.StringValue("")),
			customFieldOptionValue("unset", "Unset", types.StringNull()),
		}),
	}

	got, diags := customFieldToModel(context.Background(), field, configured)
	if diags.HasError() {
		t.Fatal(diags)
	}

	var options []customFieldOptionModel
	if diags := got.SelectOptions.ElementsAs(context.Background(), &options, false); diags.HasError() {
		t.Fatal(diags)
	}
	byValue := map[string]customFieldOptionModel{}
	for _, option := range options {
		byValue[option.Value.ValueString()] = option
	}
	if d := byValue["empty"].Description; !d.Equal(types.StringValue("")) {
		t.Errorf("empty description = %s, want \"\"", d)
	}
	for _, value := range []string{"unset", "new"} {
		if d := byValue[value].Description; !d.IsNull() {
			t.Errorf("%s description = %s, want null", value, d)
		}
	}

	wantIDs := map[string]string{"empty": "u1", "unset": "u2", "new": "u3"}
	for value, want := range wantIDs {
		if id := got.SelectOptionIDs.Elements()[value]; !id.Equal(types.StringValue(want)) {
			t.Errorf("select_option_ids[%q] = %s, want %q", value, id, want)
		}
	}
}

// TestUnitIncidentCustomField_SelectOptionsSortedWithoutUUIDs pins that options
// are sent sorted by value and without UUIDs, so UpdateField matches them to
// the stored options by value.
func TestUnitIncidentCustomField_SelectOptionsSortedWithoutUUIDs(t *testing.T) {
	m := customFieldModel{
		SelectOptions: types.SetValueMust(customFieldOptionType, []attr.Value{
			customFieldOptionValue("b", "B", types.StringNull()),
			customFieldOptionValue("a", "A", types.StringValue("first")),
		}),
	}

	options, diags := m.toSelectOptions(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}
	if len(options) != 2 || options[0].Value != "a" || options[1].Value != "b" {
		t.Fatalf("options = %+v, want a then b", options)
	}
	for _, option := range options {
		if option.UUID != "" {
			t.Errorf("option %q was sent with UUID %q", option.Value, option.UUID)
		}
	}
	if options[0].Description != "first" {
		t.Errorf("option a description = %q, want %q", options[0].Description, "first")
	}
}

func TestUnitIncidentCustomField_SlugAndValueRules(t *testing.T) {
	for slug, valid := range map[string]bool{
		"a": true, "team": true, "affected_team": true, "a1b": true,
		"": false, "1a": false, "a1": false, "_a": false, "a_": false, "a-b": false,
	} {
		if got := customFieldSlugRegexp.MatchString(slug); got != valid {
			t.Errorf("slug %q valid = %t, want %t", slug, got, valid)
		}
	}
	for value, valid := range map[string]bool{
		"a": true, "1": true, "p1": true, "us-east.1_b": true,
		"": false, "-a": false, "a.": false, "a b": false,
	} {
		if got := customFieldOptionValueRegexp.MatchString(value); got != valid {
			t.Errorf("option value %q valid = %t, want %t", value, got, valid)
		}
	}
}

func customFieldArchivedOption(value string, archived types.Bool) attr.Value {
	return types.ObjectValueMust(customFieldOptionType.AttrTypes, map[string]attr.Value{
		"value":       types.StringValue(value),
		"label":       types.StringValue(value),
		"description": types.StringNull(),
		"color":       types.StringNull(),
		"icon":        types.StringNull(),
		"archived":    archived,
	})
}

func archivedByValue(t *testing.T, set types.Set) map[string]types.Bool {
	t.Helper()
	var options []customFieldOptionModel
	if diags := set.ElementsAs(context.Background(), &options, false); diags.HasError() {
		t.Fatal(diags)
	}
	got := map[string]types.Bool{}
	for _, option := range options {
		got[option.Value.ValueString()] = option.Archived
	}
	return got
}

// TestUnitIncidentCustomField_ArchivedState pins that archived options are
// mapped into state with their flag, as is the field's.
func TestUnitIncidentCustomField_ArchivedState(t *testing.T) {
	field := incident.CustomMetadataField{
		UUID: "f", Name: "n", Slug: "s", Type: customFieldTypeSingleSelect, Archived: true,
		Selectoptions: []incident.CustomMetadataFieldSelectOption{
			{UUID: "u1", Value: "old", Label: "Old", Archived: true},
			{UUID: "u2", Value: "new", Label: "New"},
		},
	}

	got, diags := customFieldToModel(context.Background(), field, customFieldModel{SelectOptions: types.SetNull(customFieldOptionType)})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got.Archived.Equal(types.BoolValue(true)) {
		t.Errorf("archived = %s, want true", got.Archived)
	}
	archived := archivedByValue(t, got.SelectOptions)
	if !archived["old"].Equal(types.BoolValue(true)) || !archived["new"].Equal(types.BoolValue(false)) {
		t.Errorf("option archived = %v, want old true and new false", archived)
	}
}

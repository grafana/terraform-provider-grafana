package postprocessing

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/require"
)

func TestReplaceReferences_nestedBlocks(t *testing.T) {
	plan := &tfjson.Plan{PlannedValues: &tfjson.StateValues{RootModule: &tfjson.StateModule{Resources: []*tfjson.StateResource{
		{Type: "grafana_contact_point", Name: "cp", AttributeValues: map[string]any{"name": "my-cp"}},
		{Type: "grafana_rule_group", Name: "rg", AttributeValues: map[string]any{
			"name": "my-cp",
			"rule": []any{
				map[string]any{"name": "first", "notification_settings": []any{}},
				map[string]any{"name": "second", "notification_settings": []any{map[string]any{"contact_point": "my-cp"}}},
			},
		}},
		// Plugin Framework single nested blocks are planned as objects, not lists.
		{Type: "grafana_apps_provisioning_connection_v0alpha1", Name: "conn", AttributeValues: map[string]any{
			"metadata": map[string]any{"uid": "my-conn"},
		}},
		{Type: "grafana_apps_provisioning_repository_v0alpha1", Name: "repo", AttributeValues: map[string]any{
			"spec": map[string]any{"connection": map[string]any{"name": "my-conn"}},
		}},
	}}}}

	postprocessingTest(t, "testdata/replace-nested-references.tf", func(fpath string) {
		require.NoError(t, ReplaceReferences(fpath, plan, []string{
			// Top-level "name" must not match: only the nested path is a known reference.
			"grafana_rule_group.rule.notification_settings.contact_point=grafana_contact_point.name",
			"grafana_apps_provisioning_repository_v0alpha1.spec.connection.name=grafana_apps_provisioning_connection_v0alpha1.metadata.uid",
		}))
	})
}

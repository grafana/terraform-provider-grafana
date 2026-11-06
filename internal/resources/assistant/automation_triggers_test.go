package assistant

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

func TestUnitAutomationTriggerNormalization(t *testing.T) {
	for _, tc := range []struct{ kind, input, canonical string }{
		{"slack_message", `{"teamId":" T1 ","channelId":"C1","response":{},"actor":{"requireMappedUser":false}}`, `{"teamId":"T1","channelId":"C1","threadMode":"any","response":{"mode":"thread"},"actor":{}}`},
		{"github_event", `{"repository":" grafana/test ","event":"pull_request","actions":[" MERGED ","merged"],"allowBots":false,"labels":[]}`, `{"repository":"grafana/test","event":"pull_request","actions":["merged"]}`},
		{"github_event", `{"repository":"grafana/test","event":"issue_comment","text":{"contains":" deploy "}}`, `{"repository":"grafana/test","event":"issue_comment","actions":["created"],"commentOn":"any","text":{"contains":"deploy"}}`},
		{"watcher_finding", `{"watcherId":"AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"}`, `{"watcherId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","severity":"critical"}`},
	} {
		t.Run(tc.kind+tc.input, func(t *testing.T) {
			got, err := canonicalAutomationTriggerConfig(tc.kind, tc.input)
			if err != nil {
				t.Fatal(err)
			}
			var want map[string]any
			if err := json.Unmarshal([]byte(tc.canonical), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("canonical config = %#v, want %#v", got, want)
			}
		})
	}
}

func TestUnitAutomationTriggerReadDetectsFilterDrift(t *testing.T) {
	ctx := context.Background()
	const id = "11111111-1111-4111-8111-111111111111"
	before, diags := types.MapValueFrom(ctx, automationTriggerObjectType, map[string]automationTriggerModel{
		id: {Type: types.StringValue("github_event"), Config: types.StringValue(`{"repository":"grafana/test","event":"push"}`)},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	remote := []assistantapi.AutomationTrigger{{ID: id, Type: "github_event", Config: json.RawMessage(`{"repository":"grafana/different","event":"push","actions":["pushed"]}`)}}
	after, diags := automationTriggersToModel(ctx, remote, before)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if after.Equal(before) {
		t.Fatal("a changed repository was hidden as normalization")
	}
	update, diags := automationTriggersFromModel(ctx, after)
	if diags.HasError() || len(update) != 1 || update[0].ID != id {
		t.Fatalf("trigger ID was not retained: %+v, %v", update, diags)
	}
}

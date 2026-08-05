package assistant

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

// The API echoes an action policy on every response, including disabled
// defaults for channels nobody configured. Mirroring that into state would
// leave a config with no actions block permanently inconsistent.
func TestUnitWatcherActionsToModelIgnoresUnconfiguredDefaults(t *testing.T) {
	t.Parallel()

	actions, diags := watcherActionsToModel(context.Background(), &assistantapi.WatcherActions{
		Slack:         &assistantapi.WatcherSlackAction{Enabled: false},
		Investigation: &assistantapi.WatcherInvestigationAction{Enabled: false},
	}, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if actions != nil {
		t.Fatalf("expected nil actions for an unconfigured policy, got %+v", actions)
	}
}

// Import has no prior config, so a genuinely configured policy must still be
// read back into state.
func TestUnitWatcherActionsToModelPopulatesOnImport(t *testing.T) {
	t.Parallel()

	actions, diags := watcherActionsToModel(context.Background(), &assistantapi.WatcherActions{
		Slack: &assistantapi.WatcherSlackAction{
			Enabled: true,
			Target:  &assistantapi.WatcherSlackTarget{Type: "channel", ChannelID: "C0BJTU4CLGK"},
		},
		Investigation: &assistantapi.WatcherInvestigationAction{
			Enabled:   true,
			TeamNames: []string{"Demo Kit Users"},
		},
	}, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if actions == nil || actions.Slack == nil || actions.Slack.Target == nil {
		t.Fatalf("expected a populated slack action, got %+v", actions)
	}
	if got := actions.Slack.Target.ChannelID.ValueString(); got != "C0BJTU4CLGK" {
		t.Fatalf("channel id = %q", got)
	}
	if actions.Investigation == nil || len(actions.Investigation.TeamNames.Elements()) != 1 {
		t.Fatalf("expected one investigation team, got %+v", actions.Investigation)
	}
}

// A thresholds block without a comparator must still reach the API, which
// rejects it with a clear error. Dropping it here would discard configured
// thresholds and leave state without a block the config declares.
func TestUnitWatcherQueriesFromModelKeepsThresholdsWithoutComparator(t *testing.T) {
	t.Parallel()

	queries := watcherQueriesFromModel([]watcherQueryModel{{
		Type:    types.StringValue("promql"),
		Expr:    types.StringValue("up"),
		Enabled: types.BoolValue(true),
		Thresholds: &watcherThresholdsModel{
			Comparator: types.StringNull(),
			Warning:    types.Float64Value(5),
			Critical:   types.Float64Null(),
			Source:     types.StringNull(),
		},
	}})

	if len(queries) != 1 {
		t.Fatalf("want 1 query, got %d", len(queries))
	}
	if queries[0].Thresholds == nil {
		t.Fatal("thresholds were dropped from the request")
	}
	if queries[0].Thresholds.Warning == nil || *queries[0].Thresholds.Warning != 5 {
		t.Fatalf("warning threshold lost: %+v", queries[0].Thresholds)
	}
	if queries[0].Thresholds.Critical != nil {
		t.Fatalf("unset critical should stay nil, got %v", *queries[0].Thresholds.Critical)
	}
}

func TestUnitWatcherQueryRoundTrip(t *testing.T) {
	t.Parallel()

	watcher := assistantapi.Watcher{
		ID:     "w1",
		Name:   "checkout",
		Prompt: "watch checkout",
		Status: assistantapi.WatcherStatusReady,
		Queries: []assistantapi.WatcherQuery{{
			ID:            "server-assigned",
			Type:          "promql",
			Expr:          "up",
			DatasourceUID: "prom",
			Comment:       "availability",
			Enabled:       util.Ptr(false),
			Role:          "fast_incident",
			GoodWhen:      "high",
			Thresholds: &assistantapi.WatcherQueryThresholds{
				Comparator: "lt",
				Warning:    util.Ptr(0.5),
				Critical:   util.Ptr(0.1),
				Source:     "derived",
			},
		}},
	}

	model, diags := watcherToModel(context.Background(), watcher, watcherModel{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(model.Queries) != 1 {
		t.Fatalf("want 1 query, got %d", len(model.Queries))
	}
	q := model.Queries[0]
	if q.Enabled.ValueBool() {
		t.Error("enabled=false did not round-trip")
	}
	if q.Role.ValueString() != "fast_incident" || q.GoodWhen.ValueString() != "high" {
		t.Errorf("role/good_when lost: %q %q", q.Role.ValueString(), q.GoodWhen.ValueString())
	}
	if q.Thresholds == nil || q.Thresholds.Critical.ValueFloat64() != 0.1 {
		t.Errorf("thresholds lost: %+v", q.Thresholds)
	}
}

func TestUnitWatcherStartedDerivedFromStatus(t *testing.T) {
	t.Parallel()

	for status, wantStarted := range map[string]bool{
		assistantapi.WatcherStatusRunning: true,
		assistantapi.WatcherStatusReady:   false,
		assistantapi.WatcherStatusPaused:  false,
		assistantapi.WatcherStatusDraft:   false,
	} {
		model, diags := watcherToModel(context.Background(), assistantapi.Watcher{
			ID: "w", Name: "n", Prompt: "p", Status: status,
		}, watcherModel{})
		if diags.HasError() {
			t.Fatalf("%s: unexpected diagnostics: %v", status, diags)
		}
		if model.Started.ValueBool() != wantStarted {
			t.Errorf("status %q: started = %v, want %v", status, model.Started.ValueBool(), wantStarted)
		}
		if model.Status.ValueString() != status {
			t.Errorf("status %q not reported verbatim", status)
		}
	}
}

// The API returns nothing for an empty collection, but `datasource_uids = []`
// must stay an empty list rather than collapsing to null.
func TestUnitStringsToListValuePreservingKeepsExplicitEmptyList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	empty, diags := types.ListValueFrom(ctx, types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("building empty list: %v", diags)
	}

	got, diags := stringsToListValuePreserving(ctx, nil, empty)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.IsNull() {
		t.Fatal("explicitly empty list collapsed to null")
	}

	got, diags = stringsToListValuePreserving(ctx, nil, types.ListNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !got.IsNull() {
		t.Fatal("absent list should stay null")
	}
}

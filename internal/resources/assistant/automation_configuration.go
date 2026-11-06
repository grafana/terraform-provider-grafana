package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

// Only changed operator fields belong in a patch. In particular, a metadata
// edit must not revalidate unchanged Slack destinations or replace event triggers.
func automationUpdateFromModels(ctx context.Context, plan, state automationModel) (assistantapi.AutomationUpdate, diag.Diagnostics) {
	update := assistantapi.AutomationUpdate{
		Name:             changedAutomationString(plan.Name, state.Name),
		Description:      changedAutomationString(plan.Description, state.Description),
		Prompt:           changedAutomationString(plan.Prompt, state.Prompt),
		Scope:            changedAutomationString(plan.Scope, state.Scope),
		ExecutionMode:    changedAutomationString(plan.ExecutionMode, state.ExecutionMode),
		ScheduleCron:     changedAutomationString(plan.ScheduleCron, state.ScheduleCron),
		ScheduleTimezone: changedAutomationString(plan.ScheduleTimezone, state.ScheduleTimezone),
		ScheduleEndsAt:   changedAutomationString(plan.ScheduleEndsAt, state.ScheduleEndsAt),
	}
	var diags diag.Diagnostics
	if !plan.Enabled.Equal(state.Enabled) {
		update.Enabled = util.Ptr(plan.Enabled.ValueBool())
	}
	if !plan.Labels.Equal(state.Labels) {
		labels := map[string]string{}
		if !plan.Labels.IsNull() {
			diags.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		}
		update.Labels = &labels // {} clears; omitted preserves.
	}
	if !plan.Triggers.Equal(state.Triggers) {
		triggers, triggerDiags := automationTriggersFromModel(ctx, plan.Triggers)
		diags.Append(triggerDiags...)
		update.Triggers = &triggers // [] clears; omitted preserves IDs and subscriptions.
	}

	if !plan.ContextItems.Equal(state.ContextItems) {
		items, itemDiags := rawJSONFromString(ctx, plan.ContextItems)
		diags.Append(itemDiags...)
		if items == nil {
			items = json.RawMessage("[]") // JSON null is decoded as no update.
		}
		update.ContextItems = &items
	}
	if !reflect.DeepEqual(plan.Notifications, state.Notifications) {
		next, nextDiags := automationNotificationsFromModel(ctx, plan.Notifications)
		previous, previousDiags := automationNotificationsFromModel(ctx, state.Notifications)
		diags.Append(nextDiags...)
		diags.Append(previousDiags...)
		if !diags.HasError() {
			update.Notifications = map[string]json.RawMessage{}
			// Only these destinations are owned by the resource. Omitted
			// providers (including write-only webhook settings) stay untouched.
			for key, pair := range map[string][2]any{
				"slack": {next.Slack, previous.Slack},
				"email": {next.Email, previous.Email},
			} {
				if !reflect.DeepEqual(pair[0], pair[1]) {
					raw, err := json.Marshal(pair[0])
					if err != nil {
						diags.AddError("Invalid automation notification", err.Error())
						continue
					}
					update.Notifications[key] = raw // null explicitly removes this destination.
				}
			}
		}
	}
	return update, diags
}

func automationNotificationListToModel(ctx context.Context, values []string, configured types.List, email bool) (types.List, diag.Diagnostics) {
	if !configured.IsNull() && !configured.IsUnknown() {
		var previous []string
		diags := configured.ElementsAs(ctx, &previous, false)
		if diags.HasError() {
			return types.ListNull(types.StringType), diags
		}
		if slices.Equal(canonicalAutomationNotificationList(values, email), canonicalAutomationNotificationList(previous, email)) {
			return configured, nil
		}
	}
	return stringsToListValuePreserving(ctx, values, configured)
}

func canonicalAutomationNotificationList(values []string, email bool) []string {
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if email {
			value = strings.ToLower(value)
		}
		if value != "" && !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	// notifyOn is an unordered selection, emitted in the backend's fixed order.
	if !email {
		slices.Sort(result)
	}
	return result
}

// Unspecified Slack user/workspace IDs may be resolved by Assistant. They are
// not operator-owned inputs. Import has no prior target and reads the saved IDs.
func automationResolvedSlackID(value string, configured *types.String) types.String {
	if configured != nil && configured.IsNull() {
		return types.StringNull()
	}
	if configured != nil && strings.TrimSpace(configured.ValueString()) == value {
		return *configured
	}
	return stringValueOrNull(value)
}

func changedAutomationString(plan, state types.String) *string {
	if plan.Equal(state) {
		return nil
	}
	return util.Ptr(plan.ValueString())
}

func canonicalAutomationCron(value string) string {
	var expressions []string
	for _, expression := range strings.Split(value, ";") {
		if expression = strings.TrimSpace(expression); expression != "" {
			expressions = append(expressions, expression)
		}
	}
	return strings.Join(expressions, "; ")
}

func automationDeadlineToModel(value string, configured types.String) types.String {
	actual, actualErr := time.Parse(time.RFC3339, value)
	wanted, wantedErr := time.Parse(time.RFC3339, configured.ValueString())
	if actualErr == nil && wantedErr == nil && actual.Equal(wanted) {
		// The API responds in UTC. Preserve equivalent configured offsets.
		return configured
	}
	return stringValueOrNull(value)
}

func automationContextToModel(raw json.RawMessage, configured types.String) types.String {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return types.StringNull()
	}
	var actual, wanted any
	if json.Unmarshal(raw, &actual) == nil {
		if configured.IsNull() {
			if items, ok := actual.([]any); ok && len(items) == 0 {
				return types.StringNull()
			}
		} else if json.Unmarshal([]byte(configured.ValueString()), &wanted) == nil && reflect.DeepEqual(actual, wanted) {
			return configured
		}
	}
	return types.StringValue(string(raw))
}

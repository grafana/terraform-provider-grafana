package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

type automationTriggerModel struct {
	Type   types.String `tfsdk:"type"`
	Config types.String `tfsdk:"config"`
}

var automationTriggerObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType, "config": types.StringType,
}}

func validateAutomationTriggers(ctx context.Context, value types.Map, mode types.String) diag.Diagnostics {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return diags
	}
	for key, element := range value.Elements() {
		location := path.Root("triggers").AtMapKey(key)
		id, err := uuid.Parse(key)
		if err != nil || id == uuid.Nil || id.String() != key {
			diags.AddAttributeError(location, "Invalid trigger identity", "Trigger map keys must be nonzero, lowercase UUIDs with hyphens. Keep each UUID stable across edits.")
		}
		// Another resource can supply the entire object at apply time. Its
		// UUID is still validatable now; defer the contents until known.
		object := element.(types.Object)
		if object.IsUnknown() {
			continue
		}
		if object.IsNull() {
			diags.AddAttributeError(location, "Trigger configuration required", "Each trigger must contain a type and config object.")
			continue
		}
		var trigger automationTriggerModel
		diags.Append(object.As(ctx, &trigger, basetypes.ObjectAsOptions{})...)

		if !trigger.Type.IsUnknown() && (trigger.Type.IsNull() || trigger.Type.ValueString() == "") {
			diags.AddAttributeError(location.AtName("type"), "Trigger type required", "Each trigger must declare a nonempty type.")
		}
		if !mode.IsUnknown() && mode.ValueString() != "service_account" &&
			(trigger.Type.ValueString() == "github_event" || trigger.Type.ValueString() == "watcher_finding") {
			diags.AddAttributeError(location.AtName("type"), "Service-account execution required", "GitHub and Watcher triggers require execution_mode = service_account and scope = tenant.")
		}
		if !trigger.Config.IsUnknown() && !trigger.Type.IsUnknown() {
			if _, err := canonicalAutomationTriggerConfig(trigger.Type.ValueString(), trigger.Config.ValueString()); err != nil {
				diags.AddAttributeError(location.AtName("config"), "Invalid trigger configuration", err.Error())
			}
		}
	}
	return diags
}

func automationTriggersFromModel(ctx context.Context, value types.Map) ([]assistantapi.AutomationTrigger, diag.Diagnostics) {
	result := make([]assistantapi.AutomationTrigger, 0)
	if value.IsNull() {
		return result, nil
	}
	var triggers map[string]automationTriggerModel
	diags := value.ElementsAs(ctx, &triggers, false)
	keys := make([]string, 0, len(triggers))
	for key := range triggers {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		trigger := triggers[key]
		result = append(result, assistantapi.AutomationTrigger{
			ID: key, Type: trigger.Type.ValueString(), Config: json.RawMessage(trigger.Config.ValueString()),
		})
	}
	return result, diags
}

func automationTriggersToModel(ctx context.Context, triggers []assistantapi.AutomationTrigger, configured types.Map) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(triggers) == 0 && (configured.IsNull() || configured.IsUnknown()) {
		return types.MapNull(automationTriggerObjectType), diags
	}
	previous := map[string]automationTriggerModel{}
	if !configured.IsNull() && !configured.IsUnknown() {
		diags.Append(configured.ElementsAs(ctx, &previous, false)...)
	}
	values := map[string]automationTriggerModel{}
	for _, trigger := range triggers {
		config := types.StringValue(string(trigger.Config))
		if prior, ok := previous[trigger.ID]; ok && prior.Type.ValueString() == trigger.Type {
			wanted, wantedErr := canonicalAutomationTriggerConfig(trigger.Type, prior.Config.ValueString())
			actual, actualErr := canonicalAutomationTriggerConfig(trigger.Type, string(trigger.Config))
			if wantedErr == nil && actualErr == nil && reflect.DeepEqual(wanted, actual) {
				config = prior.Config
			}
		}
		values[trigger.ID] = automationTriggerModel{Type: types.StringValue(trigger.Type), Config: config}
	}
	result, valueDiags := types.MapValueFrom(ctx, automationTriggerObjectType, values)
	diags.Append(valueDiags...)
	return result, diags
}

// Compare the backend's documented normalization, not merely JSON whitespace.
// Only equivalent configurations retain the input spelling: changed filters
// remain visible as drift. Unknown trigger types use exact JSON equivalence.
func canonicalAutomationTriggerConfig(kind, raw string) (map[string]any, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(raw), &config); err != nil || config == nil {
		return nil, fmt.Errorf("config must be a JSON object")
	}
	trim := func(key string) {
		if value, ok := config[key].(string); ok {
			config[key] = strings.TrimSpace(value)
		}
	}
	defaultString := func(key, fallback string) {
		if value, exists := config[key]; !exists || value == nil || value == "" {
			config[key] = fallback
		}
	}
	switch kind {
	case "slack_message":
		trim("teamId")
		trim("channelId")
		defaultString("threadMode", "any")
		if response, ok := config["response"].(map[string]any); ok {
			if mode, exists := response["mode"]; !exists || mode == "" {
				response["mode"] = "thread"
			}
			omitEmptyTriggerFields(response, "channelId")
		}
		if text, ok := config["text"].(map[string]any); ok {
			omitEmptyTriggerFields(text, "contains", "exact", "regex")
		}
		if actor, ok := config["actor"].(map[string]any); ok {
			omitEmptyTriggerFields(actor, "requireMappedUser")
		}
		omitEmptyTriggerFields(config, "text", "response", "actor")
	case "github_event":
		trim("repository")
		for _, key := range []string{"actions", "baseBranches", "labels", "branches"} {
			if values, ok := config[key].([]any); ok {
				config[key] = normalizeAutomationTriggerList(values, key == "actions")
			}
		}
		omitEmptyTriggerFields(config, "actions", "baseBranches", "labels", "branches", "includeBody", "allowBots", "commentOn", "text")
		if _, exists := config["actions"]; !exists {
			for event, action := range map[string]string{"issue_comment": "created", "push": "pushed", "release": "published"} {
				if config["event"] == event {
					config["actions"] = []any{action}
				}
			}
		}
		if config["event"] == "issue_comment" {
			defaultString("commentOn", "any")
		}
		if text, ok := config["text"].(map[string]any); ok {
			for _, key := range []string{"contains", "exact"} {
				if value, ok := text[key].(string); ok {
					text[key] = strings.TrimSpace(value)
				}
			}
			omitEmptyTriggerFields(text, "contains", "exact")
		}
	case "watcher_finding":
		trim("watcherId")
		if value, ok := config["watcherId"].(string); ok {
			if id, err := uuid.Parse(value); err == nil {
				config["watcherId"] = id.String()
			}
		}
		trim("severity")
		if severity, ok := config["severity"].(string); ok {
			config["severity"] = strings.ToLower(severity)
		}
		defaultString("severity", "critical")
	}
	return config, nil
}

func normalizeAutomationTriggerList(values []any, lowercase bool) []any {
	var normalized []any
	for _, value := range values {
		if text, ok := value.(string); ok {
			text = strings.TrimSpace(text)
			if lowercase {
				text = strings.ToLower(text)
			}
			value = text
		}
		if !slices.ContainsFunc(normalized, func(existing any) bool { return reflect.DeepEqual(existing, value) }) {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func omitEmptyTriggerFields(config map[string]any, keys ...string) {
	for _, key := range keys {
		value := config[key]
		empty := value == nil
		switch value := value.(type) {
		case string:
			empty = value == ""
		case bool:
			empty = !value
		case []any:
			empty = len(value) == 0
		}
		if empty {
			delete(config, key)
		}
	}
}

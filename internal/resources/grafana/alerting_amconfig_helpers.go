package grafana

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/iancoleman/strcase"
)

const (
	// grafanaAlertmanagerUID is the UID of the built-in Grafana Alertmanager.
	grafanaAlertmanagerUID = "grafana"

	// Native Alertmanager supported contact point types.
	notifierTypeEmail     = "email"
	notifierTypePagerduty = "pagerduty"
	notifierTypePushover  = "pushover"
	notifierTypeSlack     = "slack"
	notifierTypeOpsgenie  = "opsgenie"
	notifierTypeVictorops = "victorops"
	notifierTypeWebhook   = "webhook"
	notifierTypeWecom     = "wecom"
	notifierTypeTelegram  = "telegram"
	notifierTypeSns       = "sns"
	notifierTypeTeams     = "teams"
	notifierTypeWebex     = "webex"
	notifierTypeDiscord   = "discord"
)

// parseNotificationPolicyAMConfigID parses a resource ID of format {orgID}:{amUID}/policy.
func parseNotificationPolicyAMConfigID(id string) (int64, string) {
	orgID, rest := SplitOrgResourceID(id)
	amUID, _, _ := strings.Cut(rest, "/")
	return orgID, amUID
}

// parseContactPointAMConfigID parses a resource ID of format {orgID}:{amUID}/{name}.
func parseContactPointAMConfigID(id string) (int64, string, string) {
	orgID, rest := SplitOrgResourceID(id)
	amUID, name, _ := strings.Cut(rest, "/")
	return orgID, amUID, name
}

// routeModelToAMConfig converts a models.Route to the AM Config API format (map[string]any).
// Since models.Route JSON tags already match the AM Config field names, we use JSON marshaling
// and only need to handle object_matchers → matchers conversion (structured → string format).
func routeModelToAMConfig(r *models.Route) (map[string]any, error) {
	// Convert ObjectMatchers to string-format matchers for the AM Config API
	var matchers []string
	for _, m := range r.ObjectMatchers {
		matchers = append(matchers, fmt.Sprintf("%s%s%s", m[0], m[1], m[2]))
	}

	// Marshal the route to JSON, then unmarshal to map to get all fields
	data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal route: %w", err)
	}
	var route map[string]any
	if err := json.Unmarshal(data, &route); err != nil {
		return nil, fmt.Errorf("failed to unmarshal route: %w", err)
	}

	// Remove fields not used in AM Config API
	delete(route, "object_matchers")
	delete(route, "provenance")

	// Add string-format matchers if present
	if len(matchers) > 0 {
		route["matchers"] = matchers
	}

	// Recursively convert child routes
	if r.Routes != nil {
		routes := make([]any, 0, len(r.Routes))
		for _, child := range r.Routes {
			childMap, err := routeModelToAMConfig(child)
			if err != nil {
				return nil, err
			}
			routes = append(routes, childMap)
		}
		route["routes"] = routes
	}

	return route, nil
}

// amConfigToRouteModel converts an AM Config API route (map[string]any) to a models.Route.
// Since models.Route JSON tags match the AM Config field names, we use JSON marshaling
// and only need to handle matchers → object_matchers conversion (string → structured format).
func amConfigToRouteModel(m map[string]any) (*models.Route, error) {
	// Extract and convert string-format matchers before JSON unmarshaling.
	// Handle both []any (from JSON decoding) and []string (from direct Go code).
	var objectMatchers models.ObjectMatchers
	if matchersRaw, ok := m["matchers"]; ok {
		switch v := matchersRaw.(type) {
		case []any:
			for _, raw := range v {
				if s, ok := raw.(string); ok {
					objectMatchers = append(objectMatchers, parseMatcherString(s))
				}
			}
		case []string:
			for _, s := range v {
				objectMatchers = append(objectMatchers, parseMatcherString(s))
			}
		}
	}

	// Remove fields that don't map cleanly to models.Route before marshaling.
	// "matchers" uses a different format (string vs structured) and "routes"
	// are handled recursively below to process matchers in child routes.
	cleaned := make(map[string]any, len(m))
	for k, v := range m {
		cleaned[k] = v
	}
	delete(cleaned, "matchers")
	delete(cleaned, "routes")

	data, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal route map: %w", err)
	}
	var route models.Route
	if err := json.Unmarshal(data, &route); err != nil {
		return nil, fmt.Errorf("failed to unmarshal route: %w", err)
	}

	route.ObjectMatchers = objectMatchers

	// Recursively convert child routes (JSON unmarshal already handles this,
	// but we need to process matchers in children too)
	if routesRaw, ok := m["routes"]; ok {
		if routesList, ok := routesRaw.([]any); ok {
			routes := make([]*models.Route, 0, len(routesList))
			for _, child := range routesList {
				if childMap, ok := child.(map[string]any); ok {
					childRoute, err := amConfigToRouteModel(childMap)
					if err != nil {
						return nil, err
					}
					routes = append(routes, childRoute)
				}
			}
			route.Routes = routes
		}
	}

	return &route, nil
}

// parseMatcherString parses a matcher string like "label=value" or "label=~value" into an ObjectMatcher.
func parseMatcherString(s string) models.ObjectMatcher {
	for _, op := range []string{"=~", "!~", "!=", "="} {
		idx := strings.Index(s, op)
		if idx >= 0 {
			return models.ObjectMatcher{
				s[:idx],
				op,
				s[idx+len(op):],
			}
		}
	}
	return models.ObjectMatcher{s, "=", ""}
}

// isGrafanaManagedAM detects whether the alertmanager is Grafana-managed (uses grafana_managed_receiver_configs)
// or native (uses standard Alertmanager receiver configs like opsgenie_configs, webhook_configs, etc.).
func isGrafanaManagedAM(amUID string, amConfig map[string]any) bool {
	// The built-in "grafana" alertmanager is always Grafana-managed.
	// Note: grafanacloud-ngalertmanager is a native Alertmanager, not Grafana-managed.
	if amUID == grafanaAlertmanagerUID {
		return true
	}

	// Check existing receivers for grafana_managed_receiver_configs
	receivers, _ := amConfig["receivers"].([]any)
	for _, r := range receivers {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if _, ok := rm["grafana_managed_receiver_configs"]; ok {
			return true
		}
	}

	return false
}

// isNativeAMSupportedType returns true if the Grafana contact point type has a native Alertmanager equivalent.
// Types like oncall, googlechat, kafka, line, sensu, threema are Grafana-only.
func isNativeAMSupportedType(grafanaType string) bool {
	switch grafanaType {
	case notifierTypeEmail, notifierTypePagerduty, notifierTypePushover, notifierTypeSlack,
		notifierTypeOpsgenie, notifierTypeVictorops, notifierTypeWebhook, notifierTypeWecom,
		notifierTypeTelegram, notifierTypeSns, notifierTypeTeams, notifierTypeWebex, notifierTypeDiscord:
		return true
	default:
		return false
	}
}

// grafanaTypeToNativeConfigKey converts a Grafana notifier type to the native Alertmanager config key.
// e.g. "opsgenie" → "opsgenie_configs", "teams" → "msteams_configs"
func grafanaTypeToNativeConfigKey(grafanaType string) string {
	switch grafanaType {
	case notifierTypeTeams:
		return "msteams_configs"
	case notifierTypeWecom:
		return "wechat_configs"
	default:
		return grafanaType + "_configs"
	}
}

// nativeConfigKeyToGrafanaType converts a native Alertmanager config key to the Grafana notifier type.
// e.g. "opsgenie_configs" → "opsgenie", "msteams_configs" → "teams"
func nativeConfigKeyToGrafanaType(configKey string) string {
	switch configKey {
	case "msteams_configs":
		return notifierTypeTeams
	case "wechat_configs":
		return notifierTypeWecom
	default:
		return strings.TrimSuffix(configKey, "_configs")
	}
}

// isGrafanaOnlySetting returns true if the setting is Grafana-specific and doesn't exist in native Alertmanager.
func isGrafanaOnlySetting(key string) bool {
	switch key {
	case "overridePriority", "override_priority", "sendTagsAs", "send_tags_as":
		return true
	default:
		return false
	}
}

// grafanaToNativeFieldName maps a Grafana-specific field name to its native Alertmanager equivalent.
// Returns the original key if no mapping exists.
func grafanaToNativeFieldName(key string) string {
	if key == "og_priority" {
		return "priority"
	}
	return key
}

// nativeToGrafanaFieldName maps a native Alertmanager field name to its Grafana equivalent.
// Returns the original key if no mapping exists.
func nativeToGrafanaFieldName(key string) string {
	if key == "priority" {
		return "og_priority"
	}
	return key
}

// embeddedContactPointToNativeConfig converts a Grafana EmbeddedContactPoint to a native Alertmanager
// receiver config entry. It returns the config key (e.g. "opsgenie_configs") and the config map.
// Returns an error for Grafana-only types that don't exist in native Alertmanager.
func embeddedContactPointToNativeConfig(p *models.EmbeddedContactPoint) (string, map[string]any, error) {
	if !isNativeAMSupportedType(*p.Type) {
		return "", nil, fmt.Errorf("contact point type %q is not supported by native Alertmanager", *p.Type)
	}

	configKey := grafanaTypeToNativeConfigKey(*p.Type)

	settings, ok := p.Settings.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("unexpected settings type %T", p.Settings)
	}

	native := make(map[string]any, len(settings)+1)
	for k, v := range settings {
		// Skip Grafana-only settings that don't exist in native Alertmanager
		if isGrafanaOnlySetting(k) {
			continue
		}
		snakeKey := strcase.ToSnake(k)
		// Also check the snake_case version
		if isGrafanaOnlySetting(snakeKey) {
			continue
		}
		// Skip false booleans — these are likely unset Grafana-specific defaults
		// that would be rejected by native alertmanagers as unknown fields.
		if b, ok := v.(bool); ok && !b {
			continue
		}
		// Apply field mappings for Grafana fields that have native equivalents
		snakeKey = grafanaToNativeFieldName(snakeKey)
		native[snakeKey] = v
	}

	native["send_resolved"] = !p.DisableResolveMessage

	return configKey, native, nil
}

// nativeConfigToEmbeddedContactPoint converts a native Alertmanager config entry to a Grafana EmbeddedContactPoint.
func nativeConfigToEmbeddedContactPoint(grafanaType string, name string, config map[string]any) *models.EmbeddedContactPoint {
	// Extract send_resolved before converting keys
	sendResolved := true
	if sr, ok := config["send_resolved"].(bool); ok {
		sendResolved = sr
	}

	settings := make(map[string]any, len(config))
	for k, v := range config {
		if k == "send_resolved" {
			continue
		}
		// Apply reverse field mappings for native fields that have Grafana equivalents
		key := nativeToGrafanaFieldName(k)
		if key == k {
			// No mapping found, convert to camelCase
			key = strcase.ToLowerCamel(k)
		}
		settings[key] = v
	}

	return &models.EmbeddedContactPoint{
		Name:                  name,
		Type:                  &grafanaType,
		DisableResolveMessage: !sendResolved,
		Settings:              settings,
	}
}

package models

import "time"

// RolloutLevel represents the rollout level for Grafana-managed alerts migration.
type RolloutLevel int

// Integration represents an integration from the API
type Integration struct {
	Name              string        `json:"name"`
	Slug              string        `json:"slug"`
	Version           string        `json:"version"`
	Overview          string        `json:"overview"`
	Logo              Logo          `json:"logo"`
	Type              string        `json:"type"`
	Installation      *Installation `json:"installation,omitempty"`
	SearchKeywords    []string      `json:"search_keywords"`
	Categories        []string      `json:"categories"`
	DashboardFolder   string        `json:"dashboard_folder"`
	HasUpdate         bool          `json:"has_update"`
	MetricsCheckQuery string        `json:"metrics_check_query"`
	LogsCheckQuery    string        `json:"logs_check_query"`
	RuleNamespace     string        `json:"rule_namespace"`

	GrafanaManagedAlertsRolloutLevel *RolloutLevel `json:"grafana_managed_alerts_rollout_level,omitempty"`
}

// Logo represents the logo URLs for an integration
type Logo struct {
	DarkThemeURL  string `json:"dark_theme_url"`
	LightThemeURL string `json:"light_theme_url"`
}

// Installation represents the installation details of an integration
type Installation struct {
	Version       string              `json:"version"`
	InstalledOn   time.Time           `json:"installed_on"`
	Configuration *InstallationConfig `json:"configuration,omitempty"`
}

// InstallationConfig represents the configuration for installing an integration
type InstallationConfig struct {
	ConfigurableLogs   *ConfigurableLogs   `json:"configurable_logs,omitempty"`
	ConfigurableAlerts *ConfigurableAlerts `json:"configurable_alerts,omitempty"`
}

// ConfigurableLogs represents the logs configuration
type ConfigurableLogs struct {
	LogsDisabled bool `json:"logs_disabled"`
}

// ConfigurableAlerts represents the alerts configuration
type ConfigurableAlerts struct {
	AlertsDisabled bool `json:"alerts_disabled"`
}

// GetIntegrationResponse represents the response from the get integration API
type GetIntegrationResponse struct {
	Data Integration `json:"data"`
}

// InstallIntegrationRequest represents the request body for installing an integration
type InstallIntegrationRequest struct {
	Configuration *InstallationConfig `json:"configuration,omitempty"`
}

// Dashboard represents a dashboard from the get dashboards API
type Dashboard struct {
	Dashboard  map[string]interface{} `json:"dashboard"`
	FolderName string                 `json:"folder_name"`
	Overwrite  bool                   `json:"overwrite"`
}

// GetDashboardsResponse represents the response from the get dashboards API
type GetDashboardsResponse struct {
	Data []Dashboard `json:"data"`
}

// RuleGroup represents a Prometheus rule group (recording or alerting)
type RuleGroup struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

// Rule represents a single Prometheus rule (recording or alerting)
type Rule struct {
	Record        string            `json:"record,omitempty"`
	Alert         string            `json:"alert,omitempty"`
	Expr          string            `json:"expr"`
	For           string            `json:"for,omitempty"`
	KeepFiringFor string            `json:"keep_firing_for,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// IntegrationRulesResponse is the response from GET /integrations/{id}/rules
type IntegrationRulesResponse struct {
	Data IntegrationRulesData `json:"data"`
}

// IntegrationRulesData contains the rule groups for an integration
type IntegrationRulesData struct {
	Namespace      string      `json:"namespace"`
	RecordingRules []RuleGroup `json:"recording_rules,omitempty"`
	AlertingRules  []RuleGroup `json:"alerting_rules,omitempty"`
}

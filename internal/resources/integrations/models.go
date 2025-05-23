package integrations

import "time"

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

// ListIntegrationsResponse represents the response from the list integrations API
type ListIntegrationsResponse struct {
	Data map[string]Integration `json:"data"`
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

// CreateFolderRequest represents the request body for creating a folder
type CreateFolderRequest struct {
	Title string `json:"title"`
	UID   string `json:"uid"`
}

// CreateFolderResponse represents the response from creating a folder
type CreateFolderResponse struct {
	ID        int       `json:"id"`
	UID       string    `json:"uid"`
	OrgID     int       `json:"orgId"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	HasACL    bool      `json:"hasAcl"`
	CanSave   bool      `json:"canSave"`
	CanEdit   bool      `json:"canEdit"`
	CanAdmin  bool      `json:"canAdmin"`
	CanDelete bool      `json:"canDelete"`
	CreatedBy string    `json:"createdBy"`
	Created   time.Time `json:"created"`
	UpdatedBy string    `json:"updatedBy"`
	Updated   time.Time `json:"updated"`
	Version   int       `json:"version"`
}

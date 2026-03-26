package cloudintegrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana-openapi-client-go/client/dashboards"
	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	editorBasePath = "/api/plugin-proxy/grafana-easystart-app/integrations-api-editor"
	adminBasePath  = "/api/plugin-proxy/grafana-easystart-app/integrations-api-admin"

	grafanaCloudPromUID = "grafanacloud-prom"
	rulesConvertAPIPath = "/api/convert/prometheus/config/v1/rules"

	defaultRetries = 3
	defaultTimeout = 90 * time.Second

	RolloutLevelMimir       = 0
	RolloutLevelInstallOnly = 1
	RolloutLevelGrafana     = 2
)

// Client wraps the HTTP client for integrations API calls
type Client struct {
	authToken        string
	client           *http.Client
	grafanaAPIHost   string
	userAgent        string
	defaultHeaders   map[string]string
	foldersClient    folders.ClientService    // Grafana OpenAPI client for folder operations
	dashboardsClient dashboards.ClientService // Grafana OpenAPI client for dashboard operations
}

// NewClient creates a new integrations client
func NewClient(grafanaAPIHost string, authToken string, client *http.Client, userAgent string, defaultHeaders map[string]string) (*Client, error) {
	if client == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = defaultRetries
		client = retryClient.StandardClient()
		client.Timeout = defaultTimeout
	}

	return &Client{
		authToken:        authToken,
		client:           client,
		grafanaAPIHost:   grafanaAPIHost,
		userAgent:        userAgent,
		defaultHeaders:   defaultHeaders,
		foldersClient:    nil, // Will be set by the resource when available
		dashboardsClient: nil, // Will be set by the resource when available
	}, nil
}

// SetFoldersClient sets the Grafana OpenAPI folders client
func (c *Client) SetFoldersClient(foldersClient folders.ClientService) {
	c.foldersClient = foldersClient
}

// SetDashboardsClient sets the Grafana OpenAPI dashboards client
func (c *Client) SetDashboardsClient(dashboardsClient dashboards.ClientService) {
	c.dashboardsClient = dashboardsClient
}

// ListIntegrations retrieves all integrations, optionally filtering by installed status
func (c *Client) ListIntegrations(ctx context.Context, installed bool) (*ListIntegrationsResponse, error) {
	path := fmt.Sprintf("%s/integrations", editorBasePath)

	// Add query parameter if filtering by installed
	if installed {
		path += "?installed=true"
	}

	var response ListIntegrationsResponse
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	return &response, nil
}

// GetIntegration retrieves a specific integration by slug
func (c *Client) GetIntegration(ctx context.Context, slug string) (*GetIntegrationResponse, error) {
	path := fmt.Sprintf("%s/integrations/%s", editorBasePath, url.PathEscape(slug))

	var response GetIntegrationResponse
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration %s: %w", slug, err)
	}

	return &response, nil
}

// PostDashboards posts dashboards for an integration with the given configuration
func (c *Client) PostDashboards(ctx context.Context, slug string, config *InstallationConfig) (*GetDashboardsResponse, error) {
	path := fmt.Sprintf("%s/integrations/%s/dashboards", adminBasePath, url.PathEscape(slug))

	requestBody := InstallIntegrationRequest{
		Configuration: config,
	}

	var response GetDashboardsResponse
	err := c.doAPIRequest(ctx, http.MethodPost, path, &requestBody, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to post dashboards for integration %s: %w", slug, err)
	}

	return &response, nil
}

// generateFolderUID generates a folder UID from an integration slug
func (c *Client) generateFolderUID(folderName string) string {
	// Take in Dashboard Folder and sanitise the whitespace with dashes
	return strings.ReplaceAll(strings.ToLower(folderName), " ", "-")
}

// CreateFolder creates a folder
func (c *Client) CreateFolder(ctx context.Context, title, uid string) error {
	if c.foldersClient == nil {
		return fmt.Errorf("folders client not available")
	}

	body := models.CreateFolderCommand{
		Title: title,
		UID:   uid,
	}

	_, err := c.foldersClient.CreateFolder(&body)
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %w", title, err)
	}

	return nil
}

// DeleteFolder deletes a folder
func (c *Client) DeleteFolder(ctx context.Context, uid string) error {
	if c.foldersClient == nil {
		return fmt.Errorf("folders client not available")
	}

	force := true
	params := folders.NewDeleteFolderParams().WithFolderUID(uid)
	params.WithForceDeleteRules(&force)
	_, err := c.foldersClient.DeleteFolder(params)
	if err != nil {
		return fmt.Errorf("failed to delete folder %s: %w", uid, err)
	}

	return nil
}

// CreateDashboard creates a dashboard in the specified folder
func (c *Client) CreateDashboard(ctx context.Context, dashboard Dashboard, folderUID string) error {

	// Make a copy of the dashboard data to avoid modifying the original
	dashboardData := make(map[string]interface{})
	for k, v := range dashboard.Dashboard {
		dashboardData[k] = v
	}

	// Remove id from dashboard if present (similar to resource_dashboard.go)
	delete(dashboardData, "id")

	// Convert the dashboard data to the proper format
	dashboardCommand := models.SaveDashboardCommand{
		Dashboard: dashboardData,
		FolderUID: folderUID,
		Overwrite: dashboard.Overwrite,
		Message:   "creating dashboard from the Cloud Connections plugin",
	}

	// Use the OpenAPI client
	_, err := c.dashboardsClient.PostDashboard(&dashboardCommand)
	if err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	return nil
}

// InstallDashboards creates the folder and dashboards for an integration.
// Used for both install and upgrade
func (c *Client) InstallDashboards(ctx context.Context, slug string, config *InstallationConfig) error {
	integration, err := c.GetIntegration(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get integration details: %w", err)
	}

	dashboardsResponse, err := c.PostDashboards(ctx, slug, config)
	if err != nil {
		return fmt.Errorf("failed to post dashboards: %w", err)
	}

	dashboardFolder := integration.Data.DashboardFolder
	folderUID := c.generateFolderUID(dashboardFolder)
	err = c.CreateFolder(ctx, dashboardFolder, folderUID)
	if err != nil {
		if !strings.Contains(err.Error(), "412") && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create folder: %w", err)
		}
	}

	for i, dashboard := range dashboardsResponse.Data {
		err = c.CreateDashboard(ctx, dashboard, folderUID)
		if err != nil {
			_ = c.DeleteFolder(ctx, folderUID)
			return fmt.Errorf("failed to create dashboard %d: %w", i+1, err)
		}
	}

	return nil
}

// InstallIntegration installs an integration with the given configuration
func (c *Client) InstallIntegration(ctx context.Context, slug string, config *InstallationConfig) error {
	integration, err := c.GetIntegration(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get integration details: %w", err)
	}

	// Step 1: Create folder and dashboards
	err = c.InstallDashboards(ctx, slug, config)
	if err != nil {
		return err
	}

	folderUID := c.generateFolderUID(integration.Data.DashboardFolder)

	// Step 2: Install rules to Grafana Alerting if applicable
	if shouldInstallRulesOnInstall(integration.Data.GrafanaManagedAlertsRolloutLevel) {
		err = c.InstallIntegrationRules(ctx, slug, config)
		if err != nil {
			_ = c.DeleteFolder(ctx, folderUID)
			return fmt.Errorf("failed to install integration rules: %w", err)
		}
	}

	// Step 3: Install the integration
	path := fmt.Sprintf("%s/integrations/%s/install", adminBasePath, url.PathEscape(slug))

	requestBody := InstallIntegrationRequest{
		Configuration: config,
	}

	err = c.doAPIRequest(ctx, http.MethodPost, path, &requestBody, nil)
	if err != nil {
		_ = c.DeleteFolder(ctx, folderUID)
		return fmt.Errorf("failed to install integration %s: %w", slug, err)
	}

	return nil
}

// UninstallIntegration uninstalls an integration and deletes its folder and rules.
// Resources are cleaned up before calling the uninstall API, matching the plugin's
// order of operations.
func (c *Client) UninstallIntegration(ctx context.Context, slug string) error {
	integration, err := c.GetIntegration(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get integration details: %w", err)
	}

	// Clean up dashboards and alerts
	folderUID := c.generateFolderUID(integration.Data.DashboardFolder)
	_ = c.DeleteFolder(ctx, folderUID)
	_ = c.UninstallIntegrationRules(ctx, slug)

	// Remove install status in API (legacy behaviour)
	path := fmt.Sprintf("%s/integrations/%s/uninstall", adminBasePath, url.PathEscape(slug))
	err = c.doAPIRequest(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to uninstall integration %s: %w", slug, err)
	}

	return nil
}

// IsIntegrationInstalled checks if an integration is currently installed
func (c *Client) IsIntegrationInstalled(ctx context.Context, slug string) (bool, error) {
	integration, err := c.GetIntegration(ctx, slug)
	if err != nil {
		return false, err
	}

	return integration.Data.Installation != nil, nil
}

// GetIntegrationRules fetches recording and alerting rule groups for an integration
func (c *Client) GetIntegrationRules(ctx context.Context, slug string) (*IntegrationRulesData, error) {
	path := fmt.Sprintf("%s/integrations/%s/rules", adminBasePath, url.PathEscape(slug))

	var response IntegrationRulesResponse
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules for integration %s: %w", slug, err)
	}

	return &response.Data, nil
}

// resolveGrafanaRulesNamespace determines the Grafana Alerting namespace.
// Priority: dashboard_folder > rule_namespace > "Integration - {name}"
func resolveGrafanaRulesNamespace(dashboardFolder, ruleNamespace, integrationName string) string {
	if dashboardFolder != "" {
		return dashboardFolder
	}
	if ruleNamespace != "" {
		return ruleNamespace
	}
	if integrationName != "" {
		return fmt.Sprintf("Integration - %s", integrationName)
	}
	return ""
}

// shouldInstallRulesOnInstall returns true if rules should be installed to
// Grafana Alerting for a new installation (rollout level >= 1).
func shouldInstallRulesOnInstall(rolloutLevel *int) bool {
	return rolloutLevel != nil && *rolloutLevel >= RolloutLevelInstallOnly
}

// shouldInstallRulesOnUpgrade returns true if rules are managed by Grafana Alerting
func shouldInstallRulesOnUpgrade(rulesExistInGrafana bool, rolloutLevel *int) bool {
	if rolloutLevel == nil {
		return false
	}
	level := *rolloutLevel

	if rulesExistInGrafana && level != RolloutLevelMimir {
		return true
	}
	if !rulesExistInGrafana && level == RolloutLevelGrafana {
		return true
	}
	return false
}

// InstallIntegrationRules fetches rules from the integrations API and imports
// them into Grafana's native alerting system via the conversion-prometheus API.
// Source: https://grafana.com/docs/grafana/latest/alerting/alerting-rules/alerting-migration/#compatible-endpoints
func (c *Client) InstallIntegrationRules(ctx context.Context, slug string, config *InstallationConfig) error {
	if config != nil &&
		config.ConfigurableAlerts != nil &&
		config.ConfigurableAlerts.AlertsDisabled {
		return nil
	}

	rulesData, err := c.GetIntegrationRules(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get integration rules: %w", err)
	}

	integration, err := c.GetIntegration(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get integration details for rules namespace: %w", err)
	}

	namespace := resolveGrafanaRulesNamespace(
		integration.Data.DashboardFolder,
		integration.Data.RuleNamespace,
		integration.Data.Name,
	)
	if namespace == "" {
		return nil
	}

	var allGroups []RuleGroup
	allGroups = append(allGroups, rulesData.RecordingRules...)
	allGroups = append(allGroups, rulesData.AlertingRules...)

	if len(allGroups) == 0 {
		return nil
	}

	payload := map[string][]RuleGroup{
		namespace: allGroups,
	}

	return c.doAPIRequestWithHeaders(ctx, http.MethodPost, rulesConvertAPIPath, payload, nil, map[string]string{
		"X-Grafana-Alerting-Datasource-UID": grafanaCloudPromUID,
	})
}

// UninstallIntegrationRules deletes the rule namespace from Grafana
func (c *Client) UninstallIntegrationRules(ctx context.Context, slug string) error {
	integration, err := c.GetIntegration(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get integration details for rules namespace: %w", err)
	}

	namespace := resolveGrafanaRulesNamespace(
		integration.Data.DashboardFolder,
		integration.Data.RuleNamespace,
		integration.Data.Name,
	)
	if namespace == "" {
		return nil
	}

	path := fmt.Sprintf("%s/%s", rulesConvertAPIPath, url.PathEscape(namespace))
	err = c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		if err == ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete rule namespace %s: %w", namespace, err)
	}
	return nil
}

// CheckRulesExist checks whether rules exist in Grafana for a given namespace
func (c *Client) CheckRulesExist(ctx context.Context, namespace string) (bool, error) {
	path := fmt.Sprintf("%s/%s", rulesConvertAPIPath, url.PathEscape(namespace))
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		if err == ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UpgradeIntegration upgrades an installed integration to its latest version.
func (c *Client) UpgradeIntegration(ctx context.Context, slug string) error {
	path := fmt.Sprintf("%s/integrations/%s/upgrade", adminBasePath, url.PathEscape(slug))
	err := c.doAPIRequest(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade integration %s: %w", slug, err)
	}
	return nil
}

var (
	ErrNotFound     = fmt.Errorf("not found")
	ErrUnauthorized = fmt.Errorf("request not authorized")
)

func (c *Client) doAPIRequest(ctx context.Context, method string, path string, body any, responseData any) error {
	return c.doAPIRequestWithHeaders(ctx, method, path, body, responseData, nil)
}

func (c *Client) doAPIRequestWithHeaders(
	ctx context.Context, method, path string, body, responseData any,
	extraHeaders map[string]string,
) error {
	parsedURL, err := url.Parse(c.grafanaAPIHost)
	if err != nil {
		return fmt.Errorf("failed to parse grafana API url: %w", err)
	}

	var reqBodyBytes io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBodyBytes = bytes.NewReader(bs)
	}

	baseURL := strings.TrimSuffix(parsedURL.String(), "/")
	fullURL := baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBodyBytes)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request to %s: %w", fullURL, err)
	}

	bodyContents, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode >= 400:
		return fmt.Errorf("status: %d for URL: %s, body: %s", resp.StatusCode, fullURL, string(bodyContents))
	case responseData == nil || resp.StatusCode == http.StatusNoContent:
		return nil
	}

	err = json.Unmarshal(bodyContents, &responseData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return nil
}

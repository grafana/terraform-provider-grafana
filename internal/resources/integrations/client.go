package integrations

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

	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	// Base paths for different API operations
	editorBasePath = "/api/plugin-proxy/grafana-easystart-app/integrations-api-editor"
	adminBasePath  = "/api/plugin-proxy/grafana-easystart-app/integrations-api-admin"

	defaultRetries = 3
	defaultTimeout = 90 * time.Second
)

// Client wraps the HTTP client for integrations API calls
type Client struct {
	authToken      string
	client         *http.Client
	grafanaAPIHost string
	userAgent      string
	defaultHeaders map[string]string
	foldersClient  folders.ClientService // Grafana OpenAPI client for folder operations
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
		authToken:      authToken,
		client:         client,
		grafanaAPIHost: grafanaAPIHost,
		userAgent:      userAgent,
		defaultHeaders: defaultHeaders,
		foldersClient:  nil, // Will be set by the resource when available
	}, nil
}

// SetFoldersClient sets the Grafana OpenAPI folders client
func (c *Client) SetFoldersClient(foldersClient folders.ClientService) {
	c.foldersClient = foldersClient
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
func (c *Client) generateFolderUID(slug string) string {
	// Replace any special characters with dashes and add the integration prefix
	uid := strings.ReplaceAll(slug, "_", "-")
	uid = strings.ReplaceAll(uid, " ", "-")
	return fmt.Sprintf("integration---%s", uid)
}

// CreateFolder creates a folder using the Grafana OpenAPI client
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

// DeleteFolder deletes a folder using the Grafana OpenAPI client
func (c *Client) DeleteFolder(ctx context.Context, uid string) error {
	if c.foldersClient == nil {
		return fmt.Errorf("folders client not available")
	}

	_, err := c.foldersClient.DeleteFolder(folders.NewDeleteFolderParams().WithFolderUID(uid))
	if err != nil {
		return fmt.Errorf("failed to delete folder %s: %w", uid, err)
	}

	return nil
}

// CreateDashboardRequest represents the request body for creating a dashboard
type CreateDashboardRequest struct {
	Dashboard map[string]interface{} `json:"dashboard"`
	FolderUID string                 `json:"folderUid"`
	Overwrite bool                   `json:"overwrite"`
	Message   string                 `json:"message"`
}

// CreateDashboard creates a dashboard in the specified folder
func (c *Client) CreateDashboard(ctx context.Context, dashboard Dashboard, folderUID string) error {
	path := "/api/dashboards/db"

	requestBody := CreateDashboardRequest{
		Dashboard: dashboard.Dashboard,
		FolderUID: folderUID,
		Overwrite: dashboard.Overwrite,
		Message:   "creating dashboard from the Cloud Connections plugin",
	}

	err := c.doAPIRequest(ctx, http.MethodPost, path, &requestBody, nil)
	if err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	return nil
}

// InstallIntegration installs an integration with the given configuration using the new multi-step workflow
func (c *Client) InstallIntegration(ctx context.Context, slug string, config *InstallationConfig) error {
	// Step 1: Get the integration details to get the folder name
	integration, err := c.GetIntegration(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to get integration details: %w", err)
	}

	// Step 2: Post dashboards (this prepares the dashboards)
	dashboardsResponse, err := c.PostDashboards(ctx, slug, config)
	if err != nil {
		return fmt.Errorf("failed to post dashboards: %w", err)
	}

	// Step 3: Create the folder
	folderUID := c.generateFolderUID(slug)
	folderTitle := integration.Data.DashboardFolder
	err = c.CreateFolder(ctx, folderTitle, folderUID)
	if err != nil {
		// Check if it's a 412 error (folder already exists)
		if strings.Contains(err.Error(), "412") {
			// Folder already exists, continue with dashboard creation
		} else {
			return fmt.Errorf("failed to create folder: %w", err)
		}
	}

	// Step 4: Add each dashboard to the folder
	for _, dashboard := range dashboardsResponse.Data {
		err = c.CreateDashboard(ctx, dashboard, folderUID)
		if err != nil {
			// If dashboard creation fails, try to clean up the folder
			_ = c.DeleteFolder(ctx, folderUID)
			return fmt.Errorf("failed to create dashboard: %w", err)
		}
	}

	// Step 5: Install the integration
	path := fmt.Sprintf("%s/integrations/%s/install", adminBasePath, url.PathEscape(slug))

	requestBody := InstallIntegrationRequest{
		Configuration: config,
	}

	err = c.doAPIRequest(ctx, http.MethodPost, path, &requestBody, nil)
	if err != nil {
		// If installation fails, try to clean up the folder
		_ = c.DeleteFolder(ctx, folderUID)
		return fmt.Errorf("failed to install integration %s: %w", slug, err)
	}

	return nil
}

// UninstallIntegration uninstalls an integration and deletes its folder
func (c *Client) UninstallIntegration(ctx context.Context, slug string) error {
	// Step 1: Uninstall the integration
	path := fmt.Sprintf("%s/integrations/%s/uninstall", adminBasePath, url.PathEscape(slug))

	err := c.doAPIRequest(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to uninstall integration %s: %w", slug, err)
	}

	// Step 2: Delete the folder
	folderUID := c.generateFolderUID(slug)
	err = c.DeleteFolder(ctx, folderUID)
	if err != nil {
		// Log the error but don't fail the uninstall if folder deletion fails
		return fmt.Errorf("integration uninstalled but failed to delete folder %s: %w", folderUID, err)
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

var (
	ErrNotFound     = fmt.Errorf("not found")
	ErrUnauthorized = fmt.Errorf("request not authorized")
)

func (c *Client) doAPIRequest(ctx context.Context, method string, path string, body any, responseData any) error {
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

	// Ensure no double slashes in URL construction
	baseURL := strings.TrimSuffix(parsedURL.String(), "/")
	fullURL := baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBodyBytes)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add default headers
	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}

	// Add authentication
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", c.userAgent)

	// Debug logging - add the full URL to error messages
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
		return fmt.Errorf("not found (404) for URL: %s, response: %s", fullURL, string(bodyContents))
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("unauthorized (401) for URL: %s, response: %s", fullURL, string(bodyContents))
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

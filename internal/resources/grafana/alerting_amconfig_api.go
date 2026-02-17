package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const orgIDHeader = "X-Grafana-Org-Id"

// AMConfigClient is an HTTP client for the Alertmanager Config API.
type AMConfigClient struct {
	client   *http.Client
	baseURL  url.URL
	apiKey   string
	username string
	password string
}

// NewAMConfigClient creates a new AMConfigClient from the provider meta.
func NewAMConfigClient(meta any, httpClient *http.Client) (*AMConfigClient, error) {
	metaClient := meta.(*common.Client)
	transportConfig := metaClient.GrafanaAPIConfig
	if transportConfig == nil {
		return nil, fmt.Errorf("transport configuration not available")
	}

	if httpClient == nil {
		httpClient = &http.Client{}
	}

	baseURL := url.URL{
		Scheme: transportConfig.Schemes[0],
		Host:   transportConfig.Host,
	}

	c := &AMConfigClient{
		client:  httpClient,
		baseURL: baseURL,
		apiKey:  transportConfig.APIKey,
	}

	if transportConfig.BasicAuth != nil {
		c.username = transportConfig.BasicAuth.Username()
		c.password, _ = transportConfig.BasicAuth.Password()
	}

	return c, nil
}

// Get fetches the Alertmanager configuration for the given alertmanager UID.
// It returns the full config as a generic map to preserve all unknown fields through round-trip.
func (c *AMConfigClient) Get(ctx context.Context, orgID int64, amUID string) (map[string]any, error) {
	req, err := c.newRequest(ctx, orgID, amUID, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get alertmanager config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("alertmanager %q not found", amUID)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get alertmanager config, status %d: %s", resp.StatusCode, string(body))
	}

	var config map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode alertmanager config: %w", err)
	}
	return config, nil
}

// Post writes the Alertmanager configuration for the given alertmanager UID.
func (c *AMConfigClient) Post(ctx context.Context, orgID int64, amUID string, config map[string]any) error {
	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal alertmanager config: %w", err)
	}

	req, err := c.newRequest(ctx, orgID, amUID, http.MethodPost, jsonData)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post alertmanager config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to post alertmanager config, status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// newRequest creates an HTTP request for the AM Config API endpoint.
func (c *AMConfigClient) newRequest(ctx context.Context, orgID int64, amUID string, method string, body []byte) (*http.Request, error) {
	reqURL := c.baseURL
	reqURL.Path = fmt.Sprintf("/api/alertmanager/%s/config/api/v1/alerts", amUID)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if orgID > 0 {
		req.Header.Set(orgIDHeader, fmt.Sprintf("%d", orgID))
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	return req, nil
}

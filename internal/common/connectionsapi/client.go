package connectionsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type Client struct {
	authToken string
	apiURL    url.URL
	client    *http.Client
}

const (
	defaultRetries = 3
	defaultTimeout = 90 * time.Second
	pathPrefix     = "/api/v1/metrics-endpoint/stacks"
)

func NewClient(authToken string, rawURL string, client *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(rawURL)
	if parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("https URL scheme expected, provided %q", parsedURL.Scheme)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse connections API url: %w", err)
	}

	if client == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = defaultRetries
		client = retryClient.StandardClient()
		client.Timeout = defaultTimeout
	}

	return &Client{
		authToken: authToken,
		apiURL:    *parsedURL,
		client:    client,
	}, nil
}

type apiResponseWrapper[T any] struct {
	Data T `json:"data"`
}

type MetricsEndpointScrapeJob struct {
	Name                        string `json:"name"`
	Enabled                     bool   `json:"enabled"`
	AuthenticationMethod        string `json:"authentication_method"`
	AuthenticationBearerToken   string `json:"bearer_token,omitempty"`
	AuthenticationBasicUsername string `json:"basic_username,omitempty"`
	AuthenticationBasicPassword string `json:"basic_password,omitempty"`
	URL                         string `json:"url"`
	ScrapeIntervalSeconds       int64  `json:"scrape_interval_seconds"`
}

func (c *Client) CreateMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobData MetricsEndpointScrapeJob) (MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("%s/%s/jobs/%s", pathPrefix, stackID, jobData.Name)
	respData := apiResponseWrapper[map[string]MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodPost, path, &jobData, &respData)
	if err != nil {
		return MetricsEndpointScrapeJob{}, fmt.Errorf("failed to create metrics endpoint scrape job: %w", err)
	}
	return respData.Data[jobData.Name], nil
}

func (c *Client) GetMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobName string) (MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("%s/%s/jobs/%s", pathPrefix, stackID, jobName)
	respData := apiResponseWrapper[map[string]MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &respData)
	if err != nil {
		return MetricsEndpointScrapeJob{}, fmt.Errorf("failed to get metrics endpoint scrape job: %w", err)
	}
	return respData.Data[jobName], nil
}

func (c *Client) UpdateMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobName string, jobData MetricsEndpointScrapeJob) (MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("%s/%s/jobs/%s", pathPrefix, stackID, jobName)
	respData := apiResponseWrapper[map[string]MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodPut, path, &jobData, &respData)
	if err != nil {
		return MetricsEndpointScrapeJob{}, fmt.Errorf("failed to update metrics endpoint scrape job: %w", err)
	}
	return respData.Data[jobName], nil
}

func (c *Client) DeleteMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobName string) error {
	path := fmt.Sprintf("%s/%s/jobs/%s", pathPrefix, stackID, jobName)
	err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete metrics endpoint scrape job: %w", err)
	}
	return nil
}

var ErrNotFound = fmt.Errorf("job not found")

func (c *Client) doAPIRequest(ctx context.Context, method string, path string, body any, responseData any) error {
	var reqBodyBytes io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBodyBytes = bytes.NewReader(bs)
	}
	var resp *http.Response

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL.String()+path, reqBodyBytes)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err = c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	bodyContents, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		if resp.StatusCode == 404 {
			return ErrNotFound
		}
		return fmt.Errorf("status: %d, body: %v", resp.StatusCode, string(bodyContents))
	}
	if responseData != nil && resp.StatusCode != http.StatusNoContent {
		err = json.Unmarshal(bodyContents, &responseData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}
	return nil
}

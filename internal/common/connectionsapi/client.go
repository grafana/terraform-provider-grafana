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
)

func NewClient(authToken string, rawAPIURL string, client *http.Client) (*Client, error) {
	parsedAPIURL, err := url.Parse(rawAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Metrics Endpoint API url: %w", err)
	}

	if client == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = defaultRetries
		client = retryClient.StandardClient()
		client.Timeout = defaultTimeout
	}

	return &Client{
		authToken: authToken,
		apiURL:    *parsedAPIURL,
		client:    client,
	}, nil
}

type apiResponseWrapper[T any] struct {
	Data T `json:"data"`
}

type MetricsEndpointScrapeJob struct {
	Name                        string `json:"name"`
	Enabled                     bool   `json:"enabled"`
	AuthenticationMethod        string `json:"authenticationMethod"`
	AuthenticationBearerToken   string `json:"authenticationBearerToken"`
	AuthenticationBasicUsername string `json:"authenticationBasicUsername"`
	AuthenticationBasicPassword string `json:"authenticationBasicPassword"`
	URL                         string `json:"url"`
	ScrapeIntervalSeconds       int64  `json:"scrapeIntervalSeconds"`
}

func (c *Client) CreateMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobData MetricsEndpointScrapeJob) (*MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("TODO %s", stackID)
	respData := apiResponseWrapper[MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodPost, path, &jobData, &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to create Metrics Endpoint scrape job: %w", err)
	}
	return &respData.Data, nil
}

func (c *Client) GetMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobName string) (*MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("TODO %s %s", stackID, jobName)
	respData := apiResponseWrapper[MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to get Metrics Endpoint scrape job: %w", err)
	}
	return &respData.Data, nil
}

func (c *Client) UpdateMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobName string, jobData MetricsEndpointScrapeJob) (*MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("TODO %s %s", stackID, jobName)
	respData := apiResponseWrapper[MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodPut, path, &jobData, &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to update Metrics Endpoint scrape job: %w", err)
	}
	return &respData.Data, nil
}

func (c *Client) DeleteMetricsEndpointScrapeJob(ctx context.Context, stackID string, jobName string) error {
	path := fmt.Sprintf("TODO %s %s", stackID, jobName)
	err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Metrics Endpoint scrape job: %w", err)
	}
	return nil
}

var ErrNotFound = fmt.Errorf("metrics endpoint scrape job not found")

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

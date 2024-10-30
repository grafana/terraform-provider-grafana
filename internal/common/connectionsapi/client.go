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
	authToken      string
	apiURL         url.URL
	client         *http.Client
	userAgent      string
	defaultHeaders map[string]string
}

const (
	defaultRetries = 3
	defaultTimeout = 90 * time.Second
	pathPrefix     = "/api/v1/stacks"
)

func NewClient(authToken string, rawURL string, client *http.Client, userAgent string, defaultHeaders map[string]string) (*Client, error) {
	parsedURL, err := url.Parse(rawURL)

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
		authToken:      authToken,
		apiURL:         *parsedURL,
		client:         client,
		userAgent:      userAgent,
		defaultHeaders: defaultHeaders,
	}, nil
}

type apiResponseWrapper[T any] struct {
	Data T `json:"data"`
}

type MetricsEndpointScrapeJob struct {
	Enabled                     bool   `json:"enabled"`
	AuthenticationMethod        string `json:"authentication_method"`
	AuthenticationBearerToken   string `json:"bearer_token,omitempty"`
	AuthenticationBasicUsername string `json:"basic_username,omitempty"`
	AuthenticationBasicPassword string `json:"basic_password,omitempty"`
	URL                         string `json:"url"`
	ScrapeIntervalSeconds       int64  `json:"scrape_interval_seconds"`
}

func (c *Client) CreateMetricsEndpointScrapeJob(ctx context.Context, stackID, jobName string, jobData MetricsEndpointScrapeJob) (MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("%s/%s/metrics-endpoint/jobs/%s", pathPrefix, stackID, jobName)
	respData := apiResponseWrapper[MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodPost, path, &jobData, &respData)
	if err != nil {
		return MetricsEndpointScrapeJob{}, fmt.Errorf("failed to create metrics endpoint scrape job %q: %w", jobName, err)
	}
	return respData.Data, nil
}

func (c *Client) GetMetricsEndpointScrapeJob(ctx context.Context, stackID, jobName string) (MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("%s/%s/metrics-endpoint/jobs/%s", pathPrefix, stackID, jobName)
	respData := apiResponseWrapper[MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &respData)
	if err != nil {
		return MetricsEndpointScrapeJob{}, fmt.Errorf("failed to get metrics endpoint scrape job %q: %w", jobName, err)
	}
	return respData.Data, nil
}

func (c *Client) UpdateMetricsEndpointScrapeJob(ctx context.Context, stackID, jobName string, jobData MetricsEndpointScrapeJob) (MetricsEndpointScrapeJob, error) {
	path := fmt.Sprintf("%s/%s/metrics-endpoint/jobs/%s", pathPrefix, stackID, jobName)
	respData := apiResponseWrapper[MetricsEndpointScrapeJob]{}
	err := c.doAPIRequest(ctx, http.MethodPut, path, &jobData, &respData)
	if err != nil {
		return MetricsEndpointScrapeJob{}, fmt.Errorf("failed to update metrics endpoint scrape job %q: %w", jobName, err)
	}
	return respData.Data, nil
}

func (c *Client) DeleteMetricsEndpointScrapeJob(ctx context.Context, stackID, jobName string) error {
	path := fmt.Sprintf("%s/%s/metrics-endpoint/jobs/%s", pathPrefix, stackID, jobName)
	err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete metrics endpoint scrape job %q: %w", jobName, err)
	}
	return nil
}

var (
	ErrNotFound     = fmt.Errorf("not found")
	ErrUnauthorized = fmt.Errorf("request not authorized for stack")
)

func (c *Client) doAPIRequest(ctx context.Context, method string, path string, body any, responseData any) error {
	var reqBodyBytes io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBodyBytes = bytes.NewReader(bs)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL.String()+path, reqBodyBytes)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
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
		if resp.StatusCode == 401 {
			return ErrUnauthorized
		}
		return fmt.Errorf("status: %d", resp.StatusCode)
	}
	if responseData != nil && resp.StatusCode != http.StatusNoContent {
		err = json.Unmarshal(bodyContents, &responseData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}
	return nil
}

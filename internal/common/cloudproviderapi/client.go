package cloudproviderapi

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
	defaultHeaders map[string]string
}

const (
	defaultRetries = 3
	defaultTimeout = 90 * time.Second
)

func NewClient(authToken string, rawAPIURL string, client *http.Client, defaultHeaders map[string]string) (*Client, error) {
	parsedAPIURL, err := url.Parse(rawAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Cloud Provider API url: %w", err)
	}

	if client == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = defaultRetries
		client = retryClient.StandardClient()
		client.Timeout = defaultTimeout
	}

	return &Client{
		authToken:      authToken,
		apiURL:         *parsedAPIURL,
		client:         client,
		defaultHeaders: defaultHeaders,
	}, nil
}

type apiResponseWrapper[T any] struct {
	Data T `json:"data"`
}

type AWSAccount struct {
	// ID is the unique identifier for the AWS account in our systems.
	ID string `json:"id"`

	// RoleARN is the AWS ARN of the associated IAM role granting Grafana access to the AWS Account.
	RoleARN string `json:"role_arn"`

	// Regions is the list of AWS regions in use for the AWS Account.
	Regions []string `json:"regions"`
}

func (c *Client) CreateAWSAccount(ctx context.Context, stackID string, accountData AWSAccount) (AWSAccount, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts", stackID)
	respData := apiResponseWrapper[AWSAccount]{}
	err := c.doAPIRequest(ctx, http.MethodPost, path, &accountData, &respData)
	if err != nil {
		return AWSAccount{}, fmt.Errorf("failed to create AWS account: %w", err)
	}
	return respData.Data, nil
}

func (c *Client) GetAWSAccount(ctx context.Context, stackID string, accountID string) (AWSAccount, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts/%s", stackID, accountID)
	respData := apiResponseWrapper[AWSAccount]{}
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &respData)
	if err != nil {
		return AWSAccount{}, fmt.Errorf("failed to get AWS account: %w", err)
	}
	return respData.Data, nil
}

func (c *Client) UpdateAWSAccount(ctx context.Context, stackID string, accountID string, accountData AWSAccount) (AWSAccount, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts/%s", stackID, accountID)
	respData := apiResponseWrapper[AWSAccount]{}
	err := c.doAPIRequest(ctx, http.MethodPut, path, &accountData, &respData)
	if err != nil {
		return AWSAccount{}, fmt.Errorf("failed to update AWS account: %w", err)
	}
	return respData.Data, nil
}

func (c *Client) DeleteAWSAccount(ctx context.Context, stackID string, accountID string) error {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts/%s", stackID, accountID)
	err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete AWS account: %w", err)
	}
	return nil
}

type AWSCloudWatchScrapeJobRequest struct {
	Name                  string                         `json:"name"`
	Enabled               bool                           `json:"enabled"`
	AWSAccountResourceID  string                         `json:"awsAccountResourceID"`
	RegionsSubsetOverride []string                       `json:"regionsSubsetOverride"`
	ExportTags            bool                           `json:"exportTags"`
	Services              []AWSCloudWatchService         `json:"services"`
	CustomNamespaces      []AWSCloudWatchCustomNamespace `json:"customNamespaces"`
	StaticLabels          map[string]string              `json:"staticLabels"`
}
type AWSCloudWatchScrapeJobResponse struct {
	Name                 string                         `json:"name"`
	Enabled              bool                           `json:"enabled"`
	AWSAccountResourceID string                         `json:"awsAccountResourceID"`
	ExportTags           bool                           `json:"exportTags"`
	Services             []AWSCloudWatchService         `json:"services"`
	CustomNamespaces     []AWSCloudWatchCustomNamespace `json:"customNamespaces"`
	// computed fields beyond the original request
	RoleARN                   string            `json:"roleARN"`
	Regions                   []string          `json:"regions"`
	RegionsSubsetOverrideUsed bool              `json:"regionsSubsetOverrideUsed"`
	DisabledReason            string            `json:"disabledReason"`
	Provenance                string            `json:"provenance"`
	StaticLabels              map[string]string `json:"staticLabels"`
}
type AWSCloudWatchService struct {
	Name                        string                   `json:"name"`
	Metrics                     []AWSCloudWatchMetric    `json:"metrics"`
	ScrapeIntervalSeconds       int64                    `json:"scrapeIntervalSeconds"`
	ResourceDiscoveryTagFilters []AWSCloudWatchTagFilter `json:"resourceDiscoveryTagFilters"`
	TagsToAddToMetrics          []string                 `json:"tagsToAddToMetrics"`
}
type AWSCloudWatchCustomNamespace struct {
	Name                  string                `json:"name"`
	Metrics               []AWSCloudWatchMetric `json:"metrics"`
	ScrapeIntervalSeconds int64                 `json:"scrapeIntervalSeconds"`
}
type AWSCloudWatchMetric struct {
	Name       string   `json:"name"`
	Statistics []string `json:"statistics"`
}
type AWSCloudWatchTagFilter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c *Client) CreateAWSCloudWatchScrapeJob(ctx context.Context, stackID string, jobData AWSCloudWatchScrapeJobRequest) (AWSCloudWatchScrapeJobResponse, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/jobs/cloudwatch", stackID)
	respData := apiResponseWrapper[AWSCloudWatchScrapeJobResponse]{}
	err := c.doAPIRequest(ctx, http.MethodPost, path, &jobData, &respData)
	if err != nil {
		return AWSCloudWatchScrapeJobResponse{}, fmt.Errorf("failed to create AWS CloudWatch scrape job: %w", err)
	}
	return respData.Data, nil
}

func (c *Client) GetAWSCloudWatchScrapeJob(ctx context.Context, stackID string, jobName string) (AWSCloudWatchScrapeJobResponse, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/jobs/cloudwatch/%s", stackID, jobName)
	respData := apiResponseWrapper[AWSCloudWatchScrapeJobResponse]{}
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &respData)
	if err != nil {
		return AWSCloudWatchScrapeJobResponse{}, fmt.Errorf("failed to get AWS CloudWatch scrape job: %w", err)
	}
	return respData.Data, nil
}

func (c *Client) ListAWSCloudWatchScrapeJobs(ctx context.Context, stackID string) ([]AWSCloudWatchScrapeJobResponse, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/jobs/cloudwatch", stackID)
	respData := apiResponseWrapper[[]AWSCloudWatchScrapeJobResponse]{}
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS CloudWatch scrape job: %w", err)
	}
	return respData.Data, nil
}

func (c *Client) UpdateAWSCloudWatchScrapeJob(ctx context.Context, stackID string, jobName string, jobData AWSCloudWatchScrapeJobRequest) (AWSCloudWatchScrapeJobResponse, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/jobs/cloudwatch/%s", stackID, jobName)
	respData := apiResponseWrapper[AWSCloudWatchScrapeJobResponse]{}
	err := c.doAPIRequest(ctx, http.MethodPut, path, &jobData, &respData)
	if err != nil {
		return AWSCloudWatchScrapeJobResponse{}, fmt.Errorf("failed to update AWS CloudWatch scrape job: %w", err)
	}
	return respData.Data, nil
}

func (c *Client) DeleteAWSCloudWatchScrapeJob(ctx context.Context, stackID string, jobName string) error {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/jobs/cloudwatch/%s", stackID, jobName)
	err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete AWS CloudWatch scrape job: %w", err)
	}
	return nil
}

type AzureCredential struct {
	// ID is the unique identifier for the Azure credential in our systems.
	ID string `json:"id"`

	// Name is the user-defined name for the Azure credential.
	Name string `json:"name"`

	// TenantID is the Azure tenant ID.
	TenantID string `json:"tenant_id"`

	// ClientID is the Azure client ID.
	ClientID string `json:"client_id"`

	// ClientSecret is the Azure client secret.
	ClientSecret string `json:"client_secret"`

	// StackID is the unique identifier for the stack in our systems.
	StackID string `json:"stack_id"`

	// ResourceTagFilters is the list of Azure resource tag filters.
	ResourceTagFilters []TagFilter `json:"resource_tag_filters"`
}

type TagFilter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c *Client) CreateAzureCredential(ctx context.Context, stackID string, credentialData AzureCredential) (AzureCredential, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/azure/credentials", stackID)
	respData := apiResponseWrapper[AzureCredential]{}
	err := c.doAPIRequest(ctx, http.MethodPost, path, &credentialData, &respData)
	if err != nil {
		return AzureCredential{}, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	return respData.Data, nil
}

func (c *Client) GetAzureCredential(ctx context.Context, stackID string, credentialID string) (AzureCredential, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/azure/credentials/%s", stackID, credentialID)
	respData := apiResponseWrapper[AzureCredential]{}
	err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &respData)
	if err != nil {
		return AzureCredential{}, fmt.Errorf("failed to get Azure credential: %w", err)
	}

	return respData.Data, nil
}

func (c *Client) UpdateAzureCredential(ctx context.Context, stackID string, accountID string, credentialData AzureCredential) (AzureCredential, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/azure/credentials/%s", stackID, accountID)
	respData := apiResponseWrapper[AzureCredential]{}
	err := c.doAPIRequest(ctx, http.MethodPut, path, &credentialData, &respData)
	if err != nil {
		return AzureCredential{}, fmt.Errorf("failed to update Azure credential: %w", err)
	}

	return respData.Data, nil
}

func (c *Client) DeleteAzureCredential(ctx context.Context, stackID string, credentialID string) error {
	path := fmt.Sprintf("/api/v2/stacks/%s/azure/credentials/%s", stackID, credentialID)
	err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Azure credential: %w", err)
	}
	return nil
}

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
	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}

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

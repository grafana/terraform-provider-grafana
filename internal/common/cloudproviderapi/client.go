package cloudproviderapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	authToken string
	apiURL    url.URL
	client    *http.Client
}

func NewClient(authToken string, rawAPIURL string, client *http.Client) (*Client, error) {
	parsedAPIURL, err := url.Parse(rawAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Cloud Provider API url: %w", err)
	}

	return &Client{
		authToken: authToken,
		apiURL:    *parsedAPIURL,
		client:    client,
	}, nil
}

type awsAccountsAPIResponseWrapper struct {
	Data AWSAccount `json:"data"`
}

type AWSAccount struct {
	// ID is the unique identifier for the AWS account in our systems.
	ID string `json:"id"`

	// RoleARN is the AWS ARN of the associated IAM role granting Grafana access to the AWS Account.
	RoleARN string `json:"role_arn"`

	// Regions is the list of AWS regions in use for the AWS Account.
	Regions []string `json:"regions"`
}

func (c *Client) CreateAWSAccount(ctx context.Context, stackID string, accountData AWSAccount) (*AWSAccount, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts", stackID)
	account, err := c.doAWSAccountsAPIRequest(ctx, http.MethodPost, path, &accountData)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS account: %w", err)
	}
	return account, nil
}

func (c *Client) GetAWSAccount(ctx context.Context, stackID string, accountID string) (*AWSAccount, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts/%s", stackID, accountID)
	account, err := c.doAWSAccountsAPIRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS account: %w", err)
	}
	return account, nil
}

func (c *Client) UpdateAWSAccount(ctx context.Context, stackID string, accountID string, accountData AWSAccount) (*AWSAccount, error) {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts/%s", stackID, accountID)
	account, err := c.doAWSAccountsAPIRequest(ctx, http.MethodPut, path, &accountData)
	if err != nil {
		return nil, fmt.Errorf("failed to update AWS account: %w", err)
	}
	return account, nil
}

func (c *Client) DeleteAWSAccount(ctx context.Context, stackID string, accountID string) error {
	path := fmt.Sprintf("/api/v2/stacks/%s/aws/accounts/%s", stackID, accountID)
	_, err := c.doAWSAccountsAPIRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete AWS account: %w", err)
	}
	return nil
}

type AWSCloudWatchScrapeJob struct {
	StackID               string
	Name                  string
	Enabled               bool
	AWSAccountResourceID  string
	Regions               []string
	ServiceConfigurations []AWSCloudWatchServiceConfiguration
}
type AWSCloudWatchServiceConfiguration struct {
	Name                        string
	Metrics                     []AWSCloudWatchMetric
	ScrapeIntervalSeconds       int64
	ResourceDiscoveryTagFilters []AWSCloudWatchTagFilter
	TagsToAddToMetrics          []string
	IsCustomNamespace           bool
}
type AWSCloudWatchMetric struct {
	Name       string
	Statistics []string
}
type AWSCloudWatchTagFilter struct {
	Key   string
	Value string
}

func (c *Client) doAWSAccountsAPIRequest(ctx context.Context, method string, path string, body any) (*AWSAccount, error) {
	var reqBodyBytes io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBodyBytes = bytes.NewReader(bs)
	}
	var resp *http.Response

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL.String()+path, reqBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err = c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	bodyContents, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status: %d, body: %v", resp.StatusCode, string(bodyContents))
	}
	if resp.StatusCode != http.StatusNoContent {
		responseData := awsAccountsAPIResponseWrapper{}
		err = json.Unmarshal(bodyContents, &responseData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
		}
		return &responseData.Data, nil
	}
	return nil, nil
}

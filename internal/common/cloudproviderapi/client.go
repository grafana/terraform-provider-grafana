package cloudproviderapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	apiURL url.URL
	client *http.Client
}

func NewClient(rawAPIURL string) (*Client, error) {
	parsedAPIURL, err := url.Parse(rawAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Cloud Provider API url: %w", err)
	}

	return &Client{
		apiURL: *parsedAPIURL,
		client: http.DefaultClient,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL.String()+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.client.Do(req)
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
	responseData := awsAccountsAPIResponseWrapper{}
	err = json.Unmarshal(bodyContents, &responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return &responseData.Data, nil
}

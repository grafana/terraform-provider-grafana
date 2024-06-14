package cloudproviderapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	authToken   string
	apiURL      url.URL
	client      *http.Client
	retryConfig retryConfig
}

type retryConfig struct {
	maxAttempts int
	baseTimeout time.Duration
	maxTimeout  time.Duration
}

func NewClient(authToken string, rawAPIURL string) (*Client, error) {
	parsedAPIURL, err := url.Parse(rawAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Cloud Provider API url: %w", err)
	}

	return &Client{
		authToken: authToken,
		apiURL:    *parsedAPIURL,
		client:    http.DefaultClient,
		retryConfig: retryConfig{
			maxAttempts: 5,
			baseTimeout: 5 * time.Second,
			maxTimeout:  125 * time.Second,
		},
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

func (c *Client) doAWSAccountsAPIRequest(ctx context.Context, method string, path string, accountData *AWSAccount) (*AWSAccount, error) {
	var reqBodyBytes io.Reader
	if accountData != nil {
		bs, err := json.Marshal(accountData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBodyBytes = bytes.NewReader(bs)
	}
	var resp *http.Response
	timeoutDuration := c.retryConfig.baseTimeout
	for i := 1; i <= c.retryConfig.maxAttempts; i++ {
		logStrPrefix := fmt.Sprintf("attempt %d/%d: ", i, c.retryConfig.maxAttempts)

		req, err := http.NewRequestWithContext(ctx, method, c.apiURL.String()+path, reqBodyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
		req.Header.Add("Content-Type", "application/json")

		resp, err = c.client.Do(req)
		if err != nil {
			log.Printf("%s - failed to do request: %s", logStrPrefix, err.Error())
			timeoutDuration = c.doRetryTimeout(timeoutDuration)
			continue
		}
		if resp.StatusCode < http.StatusInternalServerError &&
			resp.StatusCode != http.StatusTooManyRequests {
			break
		} else {
			log.Printf("%s - received status code %d, retrying...", logStrPrefix, resp.StatusCode)
			timeoutDuration = c.doRetryTimeout(timeoutDuration)
			continue
		}
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

func (c *Client) doRetryTimeout(timeoutDuration time.Duration) time.Duration {
	time.Sleep(timeoutDuration)
	nextTimeoutBase := min(timeoutDuration*2, c.retryConfig.maxTimeout)
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(nextTimeoutBase)))
	if err != nil {
		panic(err)
	}
	return nextTimeoutBase/2 + time.Duration(nBig.Int64())
}

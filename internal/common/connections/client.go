package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Client struct {
	token              string
	userAgent          string
	integrationsURL    url.URL
	hostedExportersURL url.URL
	NumRetries         int
	client             *http.Client
}

func New(token string, userAgent string, integrationsURL string, hostedExportersURL string) (*Client, error) {
	iu, err := url.Parse(integrationsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse integrationsURL: %w", err)
	}

	hu, err := url.Parse(hostedExportersURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hostedExportersURL: %w", err)
	}

	return &Client{
		token:              token,
		userAgent:          userAgent,
		integrationsURL:    *iu,
		hostedExportersURL: *hu,
		NumRetries:         3,
		client:             http.DefaultClient,
	}, nil
}

type awsConnectionResponseWrapper struct {
	Data AWSConnection `json:"data"`
}

// AWSConnection represents a connection to a user's AWS account.
type AWSConnection struct {
	// Name is a human-friendly identifier of the connection.
	Name string `json:"name"`

	// RoleARN is the ARN of the associated IAM role.
	RoleARN string `json:"role_arn"`

	// Regions is the list of AWS regions that the connections can be used with
	Regions []string `json:"regions"`
}

func (c *Client) CreateAWSConnection(ctx context.Context, stackID string, connection AWSConnection) error {
	path := fmt.Sprintf("v2/aws/stack/%s/connections", stackID)
	err := c.request(ctx, http.MethodPost, path, nil, connection, nil)
	if err != nil {
		return fmt.Errorf("failed to create aws connection at %s: %w", path, err)
	}
	return nil
}

func (c *Client) GetAWSConnection(ctx context.Context, stackID string, name string) (*AWSConnection, error) {
	path := fmt.Sprintf("v2/aws/stack/%s/connections/%s", stackID, name)
	connection := &awsConnectionResponseWrapper{}
	err := c.request(ctx, http.MethodGet, path, nil, nil, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws connection at %s: %w", path, err)
	}
	return &connection.Data, nil
}

func (c *Client) UpdateAWSConnection(ctx context.Context, stackID string, connection AWSConnection) error {
	path := fmt.Sprintf("v2/aws/stack/%s/connections/%s", stackID, connection.Name)
	err := c.request(ctx, http.MethodPut, path, nil, connection, nil)
	if err != nil {
		return fmt.Errorf("failed to update aws connection at %s: %w", path, err)
	}
	return nil
}

func (c *Client) DeleteAWSConnection(ctx context.Context, stackID string, name string) error {
	path := fmt.Sprintf("v2/aws/stack/%s/connections/%s", stackID, name)
	err := c.request(ctx, http.MethodDelete, path, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete aws connection at %s: %w", path, err)
	}
	return nil
}

type awsServiceResponseWrapper struct {
	Data AWSService `json:"data"`
}

type AWSService struct {
	Name                  string   `json:"name"`
	Metrics               []Metric `json:"metrics"`
	ScrapeIntervalSeconds *int64   `json:"scrape_interval_seconds"`
}

type Metric struct {
	Name       string   `json:"name"`
	Statistics []string `json:"statistics,omitempty"`
}

func (c *Client) CreateAWSService(ctx context.Context, stackID string, connectionName string, service AWSService) error {
	return c.UpdateAWSService(ctx, stackID, connectionName, service)
}

func (c *Client) GetAWSService(ctx context.Context, stackID string, connectionName string, serviceName string) (*AWSService, error) {
	path := fmt.Sprintf("v2/aws/stack/%s/connections/%s/services/%s", stackID, connectionName, serviceName)
	service := &awsServiceResponseWrapper{}
	err := c.request(ctx, http.MethodGet, path, nil, nil, service)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws connection at %s: %w", path, err)
	}
	return &service.Data, nil
}

func (c *Client) UpdateAWSService(ctx context.Context, stackID string, connectionName string, service AWSService) error {
	path := fmt.Sprintf("v2/aws/stack/%s/connections/%s/services/%s", stackID, connectionName, service.Name)
	err := c.request(ctx, http.MethodPut, path, nil, service, nil)
	if err != nil {
		return fmt.Errorf("failed to update aws connection at %s: %w", path, err)
	}
	return nil
}

func (c *Client) DeleteAWSService(ctx context.Context, stackID string, connectionName string, serviceName string) error {
	path := fmt.Sprintf("v2/aws/stack/%s/connections/%s/services/%s", stackID, connectionName, serviceName)
	err := c.request(ctx, http.MethodDelete, path, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete aws connection at %s: %w", path, err)
	}
	return nil
}

func (c *Client) request(ctx context.Context, method, requestPath string, query url.Values, body any, responseStruct interface{}) error {
	var (
		req          *http.Request
		resp         *http.Response
		err          error
		bodyContents []byte
	)

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	// retry logic
	for n := 0; n <= c.NumRetries; n++ {
		req, err = c.newRequest(ctx, method, requestPath, query, bytes.NewBuffer(data))
		if err != nil {
			return err
		}
		// Wait a bit if that's not the first request
		if n != 0 {
			time.Sleep(time.Second * 5)
		}

		resp, err = c.client.Do(req)

		// If err is not nil, retry again
		// That's either caused by client policy, or failure to speak HTTP (such as network connectivity problem). A
		// non-2xx status code doesn't cause an error.
		if err != nil {
			continue
		}

		// read the body (even on non-successful HTTP status codes), as that's what the unit tests expect
		bodyContents, err = io.ReadAll(resp.Body)

		resp.Body.Close()

		// if there was an error reading the body, try again
		if err != nil {
			continue
		}

		// Exit the loop if we have something final to return. This is anything < 500, if it's not a 429.
		if resp.StatusCode < http.StatusInternalServerError && resp.StatusCode != http.StatusTooManyRequests {
			break
		}
	}
	if err != nil {
		return err
	}

	// check status code.
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status: %d, body: %v", resp.StatusCode, string(bodyContents))
	}

	if responseStruct == nil {
		return nil
	}

	err = json.Unmarshal(bodyContents, responseStruct)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context, method, requestPath string, query url.Values, body io.Reader) (*http.Request, error) {
	endpoint := c.hostedExportersURL
	endpoint.Path = path.Join(endpoint.Path, requestPath)
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return req, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Add("Content-Type", "application/json")
	return req, err
}

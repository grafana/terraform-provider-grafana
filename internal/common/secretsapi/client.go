package secretsapi

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
	basicAuth      *url.Userinfo
	apiURL         url.URL
	client         *http.Client
	userAgent      string
	defaultHeaders map[string]string
	namespace      string
}

const (
	defaultRetries = 3
	defaultTimeout = 90 * time.Second
	pathPrefix     = "apis/secret.grafana.app/v1beta1"
)

func NewClient(authToken string, rawURL string, basicAuth *url.Userinfo, client *http.Client, userAgent string, defaultHeaders map[string]string, namespace string) (*Client, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secrets API url: %w", err)
	}

	if client == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = defaultRetries
		client = retryClient.StandardClient()
		client.Timeout = defaultTimeout
	}

	return &Client{
		authToken:      authToken,
		basicAuth:      basicAuth,
		apiURL:         *parsedURL,
		client:         client,
		userAgent:      userAgent,
		defaultHeaders: defaultHeaders,
		namespace:      namespace,
	}, nil
}

func (c *Client) Namespace() string {
	return c.namespace
}

type ObjectMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Keeper struct {
	APIVersion string         `json:"apiVersion,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	Metadata   ObjectMetadata `json:"metadata"`
	Spec       KeeperSpec     `json:"spec"`
}

type KeeperSpec struct {
	Description string     `json:"description,omitempty"`
	Type        string     `json:"type,omitempty"`
	AWS         *KeeperAWS `json:"aws,omitempty"`
}

type KeeperAWS struct {
	Region     string               `json:"region"`
	AssumeRole *KeeperAWSAssumeRole `json:"assumeRole,omitempty"`
}

type KeeperAWSAssumeRole struct {
	AssumeRoleARN string `json:"assumeRoleArn"`
	ExternalID    string `json:"externalID"`
}

type SecureValue struct {
	APIVersion string            `json:"apiVersion,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Metadata   ObjectMetadata    `json:"metadata"`
	Spec       SecureValueSpec   `json:"spec"`
	Status     SecureValueStatus `json:"status"`
}

type SecureValueSpec struct {
	Description string   `json:"description,omitempty"`
	Value       string   `json:"value,omitempty"`
	Ref         string   `json:"ref,omitempty"`
	Decrypters  []string `json:"decrypters,omitempty"`
}

type SecureValueStatus struct {
	Keeper string `json:"keeper"`
}

func (c *Client) CreateKeeper(ctx context.Context, namespace string, keeper Keeper) (Keeper, error) {
	path := fmt.Sprintf("%s/namespaces/%s/keepers", pathPrefix, namespace)
	var resp Keeper
	if err := c.doAPIRequest(ctx, http.MethodPost, path, keeper, &resp); err != nil {
		return Keeper{}, fmt.Errorf("failed to create keeper %q: %w", keeper.Metadata.Name, err)
	}
	return resp, nil
}

func (c *Client) GetKeeper(ctx context.Context, namespace, name string) (Keeper, error) {
	path := fmt.Sprintf("%s/namespaces/%s/keepers/%s", pathPrefix, namespace, name)
	var resp Keeper
	if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return Keeper{}, fmt.Errorf("failed to get keeper %q: %w", name, err)
	}
	return resp, nil
}

func (c *Client) UpdateKeeper(ctx context.Context, namespace, name string, keeper Keeper) (Keeper, error) {
	path := fmt.Sprintf("%s/namespaces/%s/keepers/%s", pathPrefix, namespace, name)
	var resp Keeper
	if err := c.doAPIRequest(ctx, http.MethodPut, path, keeper, &resp); err != nil {
		return Keeper{}, fmt.Errorf("failed to update keeper %q: %w", name, err)
	}
	return resp, nil
}

func (c *Client) DeleteKeeper(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("%s/namespaces/%s/keepers/%s", pathPrefix, namespace, name)
	if err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete keeper %q: %w", name, err)
	}
	return nil
}

func (c *Client) ActivateKeeper(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("%s/namespaces/%s/keepers/%s/activate", pathPrefix, namespace, name)
	if err := c.doAPIRequest(ctx, http.MethodPost, path, map[string]any{}, nil); err != nil {
		return fmt.Errorf("failed to activate keeper %q: %w", name, err)
	}
	return nil
}

func (c *Client) CreateSecureValue(ctx context.Context, namespace string, secureValue SecureValue) (SecureValue, error) {
	path := fmt.Sprintf("%s/namespaces/%s/securevalues", pathPrefix, namespace)
	var resp SecureValue
	if err := c.doAPIRequest(ctx, http.MethodPost, path, secureValue, &resp); err != nil {
		return SecureValue{}, fmt.Errorf("failed to create secure value %q: %w", secureValue.Metadata.Name, err)
	}
	return resp, nil
}

func (c *Client) GetSecureValue(ctx context.Context, namespace, name string) (SecureValue, error) {
	path := fmt.Sprintf("%s/namespaces/%s/securevalues/%s", pathPrefix, namespace, name)
	var resp SecureValue
	if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return SecureValue{}, fmt.Errorf("failed to get secure value %q: %w", name, err)
	}
	return resp, nil
}

func (c *Client) UpdateSecureValue(ctx context.Context, namespace, name string, secureValue SecureValue) (SecureValue, error) {
	path := fmt.Sprintf("%s/namespaces/%s/securevalues/%s", pathPrefix, namespace, name)
	var resp SecureValue
	if err := c.doAPIRequest(ctx, http.MethodPut, path, secureValue, &resp); err != nil {
		return SecureValue{}, fmt.Errorf("failed to update secure value %q: %w", name, err)
	}
	return resp, nil
}

func (c *Client) DeleteSecureValue(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("%s/namespaces/%s/securevalues/%s", pathPrefix, namespace, name)
	if err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete secure value %q: %w", name, err)
	}
	return nil
}

func (c *Client) doAPIRequest(ctx context.Context, method string, path string, body any, responseData any) error {
	var reqBody io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(bs)
	}

	requestURL, err := url.JoinPath(c.apiURL.String(), path)
	if err != nil {
		return fmt.Errorf("failed to build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	if c.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	} else if c.basicAuth != nil {
		username := c.basicAuth.Username()
		password, _ := c.basicAuth.Password()
		req.SetBasicAuth(username, password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	bodyContents, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("status: %d body: %s", resp.StatusCode, string(bodyContents))
	}

	if responseData != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(bodyContents, responseData); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	return nil
}

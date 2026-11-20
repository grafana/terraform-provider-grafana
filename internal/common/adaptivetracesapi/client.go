package adaptivetracesapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	pathPrefix           = "/adaptive-traces/api/v1"
	defaultTimeout       = 90 * time.Second
	maxResponseBytes     = 1 << 20
	maxErrorMessageBytes = 1024
)

// Client is a client for the Adaptive Traces API.
type Client struct {
	auth           string
	apiURL         string
	client         *http.Client
	userAgent      string
	defaultHeaders map[string]string
}

// NewClient creates a client for the Adaptive Traces API at the given base URL (the stack's traces_url).
// auth is "<traces_user_id>:<token>", sent as HTTP Basic auth.
func NewClient(auth string, baseURL string, client *http.Client, userAgent string, defaultHeaders map[string]string) *Client {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		auth:           auth,
		apiURL:         strings.TrimRight(baseURL, "/") + pathPrefix,
		client:         client,
		userAgent:      userAgent,
		defaultHeaders: defaultHeaders,
	}
}

// GetPolicy returns ErrNotFound if the policy does not exist or has expired.
func (c *Client) GetPolicy(ctx context.Context, id string) (*Policy, error) {
	return c.doPolicyRequest(ctx, http.MethodGet, policyPath(id), nil)
}

// CreatePolicy creates a policy and returns it with the server-assigned ID.
func (c *Client) CreatePolicy(ctx context.Context, policy Policy) (*Policy, error) {
	return c.doPolicyRequest(ctx, http.MethodPost, "/policies", newPolicyRequest(policy))
}

// UpdatePolicy replaces a policy with a full-body PUT, so a nil ExpiresAt clears the expiry.
func (c *Client) UpdatePolicy(ctx context.Context, id string, policy Policy) (*Policy, error) {
	return c.doPolicyRequest(ctx, http.MethodPut, policyPath(id), newPolicyRequest(policy))
}

// DeletePolicy returns ErrNotFound if the policy does not exist.
func (c *Client) DeletePolicy(ctx context.Context, id string) error {
	return c.doAPIRequest(ctx, http.MethodDelete, policyPath(id), nil, nil)
}

func policyPath(id string) string {
	return "/policies/" + url.PathEscape(id)
}

// doPolicyRequest sends a request whose response is a policy, and fails if the response has no policy ID.
func (c *Client) doPolicyRequest(ctx context.Context, method string, path string, body any) (*Policy, error) {
	var policy Policy
	if err := c.doAPIRequest(ctx, method, path, body, &policy); err != nil {
		return nil, err
	}
	if policy.ID == "" {
		return nil, errors.New("adaptive traces API returned a response without a policy ID")
	}
	return &policy, nil
}

// ErrNotFound is returned when the API responds with 404. On create and update, that means the base URL does not
// serve the API.
var ErrNotFound = errors.New("adaptive traces: not found")

func (c *Client) doAPIRequest(ctx context.Context, method string, path string, body any, responseData any) error {
	var reqBody io.Reader
	if body != nil {
		// Don't HTML-escape, so the policy body reaches the server unchanged.
		buf := &bytes.Buffer{}
		encoder := json.NewEncoder(buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(body); err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}

	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.auth)))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s %s", ErrNotFound, method, req.URL.Redacted())
	}

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// A read error is ignored here: the status is the error, and the message is best effort.
		return fmt.Errorf("adaptive traces API returned status %d: %s", resp.StatusCode, errorMessage(resp.StatusCode, respBody))
	}
	if readErr != nil {
		return fmt.Errorf("failed to read response body: %w", readErr)
	}
	if len(respBody) > maxResponseBytes {
		return fmt.Errorf("adaptive traces API response body exceeds %d bytes", maxResponseBytes)
	}
	if responseData == nil {
		return nil
	}

	if err := json.Unmarshal(respBody, responseData); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return nil
}

// errorMessage extracts the API's error message, falling back to the raw body and then the status text.
func errorMessage(statusCode int, body []byte) string {
	message := strings.TrimSpace(string(body))

	var apiErr struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
		message = apiErr.Error
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	if len(message) > maxErrorMessageBytes {
		message = strings.ToValidUTF8(message[:maxErrorMessageBytes], "") + " (truncated)"
	}

	return message
}

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
	pathPrefix = "/adaptive-traces/api/v1"

	// defaultTimeout bounds a request when NewClient gets no HTTP client, as in the other API clients.
	defaultTimeout = 90 * time.Second

	// maxResponseBytes caps how much of a response body is read. A policy is a few KiB.
	maxResponseBytes = 1 << 20

	// maxErrorMessageBytes caps how much of an error response is copied into the returned error.
	maxErrorMessageBytes = 1024
)

// Client talks to the Adaptive Traces API at <base URL>/adaptive-traces/api/v1.
type Client struct {
	auth           string
	apiURL         string
	client         *http.Client
	userAgent      string
	defaultHeaders map[string]string
}

// NewClient returns a client for the Tempo base URL (the stack's traces_url). auth is "<traces_user_id>:<token>"
// and is sent as HTTP Basic auth. A nil client falls back to an HTTP client with a 90 second timeout and no retries.
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

// CreatePolicy creates a policy and returns it with the ID and version fields the server assigned. The ID and
// read-only fields of policy are ignored. A 404 returns ErrNotFound, which here means the base URL does not serve
// the API.
func (c *Client) CreatePolicy(ctx context.Context, policy Policy) (*Policy, error) {
	return c.doPolicyRequest(ctx, http.MethodPost, "/policies", newPolicyRequest(policy))
}

// UpdatePolicy replaces the policy with a full-body PUT, so a nil ExpiresAt clears the expiry. The ID and read-only
// fields of policy are ignored. It returns ErrNotFound if the policy does not exist or the base URL does not serve
// the API.
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

// doPolicyRequest sends a request whose response is a policy. A success response without a policy ID, such as a
// 204 or a JSON null, is an error, so that callers never get an empty policy.
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

// ErrNotFound is returned when the API responds with 404. The error wraps it with the request method and URL.
var ErrNotFound = errors.New("adaptive traces: not found")

func (c *Client) doAPIRequest(ctx context.Context, method string, path string, body any, responseData any) error {
	var reqBody io.Reader
	if body != nil {
		// HTML escaping is off so that the policy body reaches the server byte for byte.
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

	// The status decides the outcome before the body is read, so that a failed read cannot hide a 404.
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s %s", ErrNotFound, method, req.URL.Redacted())
	}

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Whatever was read is enough for the message: the status matters more than a read error.
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

// errorMessage returns the error field of a JSON error body, falling back to the trimmed raw body, then the HTTP
// status text. The message is cut to maxErrorMessageBytes.
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
		// Drop a multi-byte character the cut split in two.
		message = strings.ToValidUTF8(message[:maxErrorMessageBytes], "") + " (truncated)"
	}

	return message
}

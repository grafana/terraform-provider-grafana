package assistantapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const pathPrefix = "/api/plugins/grafana-assistant-app/resources/api/v1"

// Client talks to the Grafana Assistant app plugin REST API.
type Client struct {
	baseURL        url.URL
	basicAuth      *url.Userinfo
	apiKey         string
	client         *http.Client
	userAgent      string
	defaultHeaders map[string]string
}

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
)

// NewClient creates a client for the assistant plugin API at the given Grafana URL.
func NewClient(grafanaURL string, basicAuth *url.Userinfo, apiKey string, httpClient *http.Client, userAgent string, defaultHeaders map[string]string) (*Client, error) {
	parsedURL, err := url.Parse(grafanaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grafana url: %w", err)
	}

	if httpClient == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 3
		httpClient = retryClient.StandardClient()
		httpClient.Timeout = 90 * time.Second
	}

	return &Client{
		baseURL:        *parsedURL,
		basicAuth:      basicAuth,
		apiKey:         apiKey,
		client:         httpClient,
		userAgent:      userAgent,
		defaultHeaders: defaultHeaders,
	}, nil
}

func (c *Client) doAPIRequest(ctx context.Context, method, path string, body any, responseData any, extraHeaders map[string]string) error {
	var reqBodyBytes io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBodyBytes = bytes.NewReader(bs)
	}

	fullURL := c.baseURL.JoinPath(pathPrefix + path).String()
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBodyBytes)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	if c.basicAuth != nil {
		password, _ := c.basicAuth.Password()
		req.SetBasicAuth(c.basicAuth.Username(), password)
	} else if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
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
		switch resp.StatusCode {
		case http.StatusNotFound:
			return ErrNotFound
		case http.StatusUnauthorized:
			return ErrUnauthorized
		default:
			msg := strings.TrimSpace(string(bodyContents))
			if msg == "" {
				msg = resp.Status
			}
			return fmt.Errorf("status %d: %s", resp.StatusCode, msg)
		}
	}

	if responseData != nil && resp.StatusCode != http.StatusNoContent && len(bodyContents) > 0 {
		if err := json.Unmarshal(bodyContents, responseData); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}
	return nil
}

func scopeHeader(scope string) map[string]string {
	return map[string]string{"X-Resource-Scope": scope}
}

// CreateRule creates a new assistant rule.
func (c *Client) CreateRule(ctx context.Context, body RuleCreate) (Rule, error) {
	var resp apiResponseWrapper[Rule]
	if err := c.doAPIRequest(ctx, http.MethodPost, "/rules", body, &resp, nil); err != nil {
		return Rule{}, fmt.Errorf("failed to create rule: %w", err)
	}
	return resp.Data, nil
}

// GetRule retrieves a rule by ID.
func (c *Client) GetRule(ctx context.Context, id string) (Rule, error) {
	var resp apiResponseWrapper[Rule]
	if err := c.doAPIRequest(ctx, http.MethodGet, "/rules/"+id, nil, &resp, nil); err != nil {
		return Rule{}, fmt.Errorf("failed to get rule %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateRule updates an existing rule.
func (c *Client) UpdateRule(ctx context.Context, id, resourceScope string, body RuleUpdate) (Rule, error) {
	var resp apiResponseWrapper[Rule]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/rules/"+id, body, &resp, scopeHeader(resourceScope)); err != nil {
		return Rule{}, fmt.Errorf("failed to update rule %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteRule deletes a rule by ID.
func (c *Client) DeleteRule(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/rules/"+id, nil, nil, scopeHeader(resourceScope)); err != nil {
		return fmt.Errorf("failed to delete rule %q: %w", id, err)
	}
	return nil
}

// CreateSkill creates a new assistant skill.
func (c *Client) CreateSkill(ctx context.Context, body SkillCreate) (Skill, error) {
	var resp apiResponseWrapper[Skill]
	if err := c.doAPIRequest(ctx, http.MethodPost, "/skills", body, &resp, nil); err != nil {
		return Skill{}, fmt.Errorf("failed to create skill: %w", err)
	}
	return resp.Data, nil
}

// GetSkill retrieves a skill by ID.
func (c *Client) GetSkill(ctx context.Context, id string) (Skill, error) {
	var resp apiResponseWrapper[Skill]
	if err := c.doAPIRequest(ctx, http.MethodGet, "/skills/"+id, nil, &resp, nil); err != nil {
		return Skill{}, fmt.Errorf("failed to get skill %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateSkill updates an existing skill.
func (c *Client) UpdateSkill(ctx context.Context, id, resourceScope string, body SkillUpdate) (Skill, error) {
	var resp apiResponseWrapper[Skill]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/skills/"+id, body, &resp, scopeHeader(resourceScope)); err != nil {
		return Skill{}, fmt.Errorf("failed to update skill %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteSkill deletes a skill by ID.
func (c *Client) DeleteSkill(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/skills/"+id, nil, nil, scopeHeader(resourceScope)); err != nil {
		return fmt.Errorf("failed to delete skill %q: %w", id, err)
	}
	return nil
}

// CreateQuickstart creates a new quickstart prompt.
func (c *Client) CreateQuickstart(ctx context.Context, body QuickstartCreate) (Quickstart, error) {
	var resp apiResponseWrapper[Quickstart]
	if err := c.doAPIRequest(ctx, http.MethodPost, "/quickstarts", body, &resp, nil); err != nil {
		return Quickstart{}, fmt.Errorf("failed to create quickstart: %w", err)
	}
	return resp.Data, nil
}

// GetQuickstart retrieves a quickstart by ID.
func (c *Client) GetQuickstart(ctx context.Context, id string) (Quickstart, error) {
	var resp apiResponseWrapper[Quickstart]
	if err := c.doAPIRequest(ctx, http.MethodGet, "/quickstarts/"+id, nil, &resp, nil); err != nil {
		return Quickstart{}, fmt.Errorf("failed to get quickstart %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateQuickstart updates an existing quickstart.
func (c *Client) UpdateQuickstart(ctx context.Context, id, resourceScope string, body QuickstartUpdate) (Quickstart, error) {
	var resp apiResponseWrapper[Quickstart]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/quickstarts/"+id, body, &resp, scopeHeader(resourceScope)); err != nil {
		return Quickstart{}, fmt.Errorf("failed to update quickstart %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteQuickstart deletes a quickstart by ID.
func (c *Client) DeleteQuickstart(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/quickstarts/"+id, nil, nil, scopeHeader(resourceScope)); err != nil {
		return fmt.Errorf("failed to delete quickstart %q: %w", id, err)
	}
	return nil
}

// CreateIntegration creates a new MCP server integration.
func (c *Client) CreateIntegration(ctx context.Context, body IntegrationCreate) (Integration, error) {
	var resp apiResponseWrapper[Integration]
	if err := c.doAPIRequest(ctx, http.MethodPost, "/integrations", body, &resp, nil); err != nil {
		return Integration{}, fmt.Errorf("failed to create integration: %w", err)
	}
	return resp.Data, nil
}

// GetIntegration retrieves an integration by ID.
func (c *Client) GetIntegration(ctx context.Context, id string) (Integration, error) {
	var resp apiResponseWrapper[Integration]
	if err := c.doAPIRequest(ctx, http.MethodGet, "/integrations/"+id, nil, &resp, nil); err != nil {
		return Integration{}, fmt.Errorf("failed to get integration %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateIntegration updates an existing integration.
func (c *Client) UpdateIntegration(ctx context.Context, id, resourceScope string, body IntegrationUpdate) (Integration, error) {
	var resp apiResponseWrapper[Integration]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/integrations/"+id, body, &resp, scopeHeader(resourceScope)); err != nil {
		return Integration{}, fmt.Errorf("failed to update integration %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteIntegration deletes an integration by ID.
func (c *Client) DeleteIntegration(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/integrations/"+id, nil, nil, scopeHeader(resourceScope)); err != nil {
		return fmt.Errorf("failed to delete integration %q: %w", id, err)
	}
	return nil
}

// MarshalMCPConfig serializes MCP configuration for the integration API.
func MarshalMCPConfig(cfg MCPConfig) (json.RawMessage, error) {
	if cfg.URL == "" && cfg.BuiltinID == "" && len(cfg.ToolPreferences) == 0 && len(cfg.ToolApprovalPolicies) == 0 {
		return nil, nil
	}
	return json.Marshal(cfg)
}

// ParseMCPConfig deserializes MCP configuration from the integration API.
func ParseMCPConfig(raw json.RawMessage) (MCPConfig, error) {
	if len(raw) == 0 {
		return MCPConfig{}, nil
	}
	var cfg MCPConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return MCPConfig{}, err
	}
	return cfg, nil
}

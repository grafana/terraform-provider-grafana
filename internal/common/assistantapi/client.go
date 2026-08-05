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

	// path may contain a pre-escaped id segment and/or a raw query string, so we
	// build the URL by concatenation rather than url.JoinPath (which would
	// re-escape an already-escaped id and encode the query separator).
	fullURL := strings.TrimRight(c.baseURL.String(), "/") + pathPrefix + path
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
	if err := c.doAPIRequest(ctx, http.MethodGet, "/rules/"+url.PathEscape(id), nil, &resp, nil); err != nil {
		return Rule{}, fmt.Errorf("failed to get rule %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateRule updates an existing rule.
func (c *Client) UpdateRule(ctx context.Context, id, resourceScope string, body RuleUpdate) (Rule, error) {
	var resp apiResponseWrapper[Rule]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/rules/"+url.PathEscape(id), body, &resp, scopeHeader(resourceScope)); err != nil {
		return Rule{}, fmt.Errorf("failed to update rule %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteRule deletes a rule by ID.
func (c *Client) DeleteRule(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/rules/"+url.PathEscape(id), nil, nil, scopeHeader(resourceScope)); err != nil {
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
	if err := c.doAPIRequest(ctx, http.MethodGet, "/skills/"+url.PathEscape(id), nil, &resp, nil); err != nil {
		return Skill{}, fmt.Errorf("failed to get skill %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateSkill updates an existing skill.
func (c *Client) UpdateSkill(ctx context.Context, id, resourceScope string, body SkillUpdate) (Skill, error) {
	var resp apiResponseWrapper[Skill]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/skills/"+url.PathEscape(id), body, &resp, scopeHeader(resourceScope)); err != nil {
		return Skill{}, fmt.Errorf("failed to update skill %q: %w", id, err)
	}
	return resp.Data, nil
}

// SetSkillCommand sets, updates, or disables the slash command for a skill.
func (c *Client) SetSkillCommand(ctx context.Context, id, resourceScope string, commandName *string) (Skill, error) {
	var resp apiResponseWrapper[Skill]
	path := "/skills/" + url.PathEscape(id) + "/command"
	if err := c.doAPIRequest(ctx, http.MethodPut, path, SkillCommandUpdate{CommandName: commandName}, &resp, scopeHeader(resourceScope)); err != nil {
		return Skill{}, fmt.Errorf("failed to set command for skill %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteSkill deletes a skill by ID.
func (c *Client) DeleteSkill(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/skills/"+url.PathEscape(id), nil, nil, scopeHeader(resourceScope)); err != nil {
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
	if err := c.doAPIRequest(ctx, http.MethodGet, "/quickstarts/"+url.PathEscape(id), nil, &resp, nil); err != nil {
		return Quickstart{}, fmt.Errorf("failed to get quickstart %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateQuickstart updates an existing quickstart.
func (c *Client) UpdateQuickstart(ctx context.Context, id, resourceScope string, body QuickstartUpdate) (Quickstart, error) {
	var resp apiResponseWrapper[Quickstart]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/quickstarts/"+url.PathEscape(id), body, &resp, scopeHeader(resourceScope)); err != nil {
		return Quickstart{}, fmt.Errorf("failed to update quickstart %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteQuickstart deletes a quickstart by ID.
func (c *Client) DeleteQuickstart(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/quickstarts/"+url.PathEscape(id), nil, nil, scopeHeader(resourceScope)); err != nil {
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
	if err := c.doAPIRequest(ctx, http.MethodGet, "/integrations/"+url.PathEscape(id), nil, &resp, nil); err != nil {
		return Integration{}, fmt.Errorf("failed to get integration %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateIntegration updates an existing integration.
func (c *Client) UpdateIntegration(ctx context.Context, id, resourceScope string, body IntegrationUpdate) (Integration, error) {
	var resp apiResponseWrapper[Integration]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/integrations/"+url.PathEscape(id), body, &resp, scopeHeader(resourceScope)); err != nil {
		return Integration{}, fmt.Errorf("failed to update integration %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteIntegration deletes an integration by ID.
func (c *Client) DeleteIntegration(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/integrations/"+url.PathEscape(id), nil, nil, scopeHeader(resourceScope)); err != nil {
		return fmt.Errorf("failed to delete integration %q: %w", id, err)
	}
	return nil
}

// listPageSize is the maximum page size accepted by the list endpoints.
const listPageSize = 100

// ListRules returns all rules visible to the caller (tenant-scoped rules plus
// the caller's own user-scoped rules), paginating through every page.
func (c *Client) ListRules(ctx context.Context) ([]Rule, error) {
	var all []Rule
	for offset := 0; ; offset += listPageSize {
		var resp apiResponseWrapper[ruleListData]
		path := fmt.Sprintf("/rules?limit=%d&offset=%d", listPageSize, offset)
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp, nil); err != nil {
			return nil, fmt.Errorf("failed to list rules: %w", err)
		}
		all = append(all, resp.Data.Rules...)
		if len(resp.Data.Rules) < listPageSize {
			break
		}
	}
	return all, nil
}

// ListSkills returns all skills visible to the caller, paginating through every page.
func (c *Client) ListSkills(ctx context.Context) ([]Skill, error) {
	var all []Skill
	for offset := 0; ; offset += listPageSize {
		var resp apiResponseWrapper[skillListData]
		path := fmt.Sprintf("/skills?limit=%d&offset=%d", listPageSize, offset)
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp, nil); err != nil {
			return nil, fmt.Errorf("failed to list skills: %w", err)
		}
		all = append(all, resp.Data.Skills...)
		if len(resp.Data.Skills) < listPageSize {
			break
		}
	}
	return all, nil
}

// ListQuickstarts returns all quickstarts visible to the caller, paginating through every page.
func (c *Client) ListQuickstarts(ctx context.Context) ([]Quickstart, error) {
	var all []Quickstart
	for offset := 0; ; offset += listPageSize {
		var resp apiResponseWrapper[quickstartListData]
		path := fmt.Sprintf("/quickstarts?limit=%d&offset=%d", listPageSize, offset)
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp, nil); err != nil {
			return nil, fmt.Errorf("failed to list quickstarts: %w", err)
		}
		all = append(all, resp.Data.Quickstarts...)
		if len(resp.Data.Quickstarts) < listPageSize {
			break
		}
	}
	return all, nil
}

// ListIntegrations returns all MCP server integrations visible to the caller,
// paginating through every page.
func (c *Client) ListIntegrations(ctx context.Context) ([]Integration, error) {
	var all []Integration
	for offset := 0; ; offset += listPageSize {
		var resp apiResponseWrapper[integrationListData]
		path := fmt.Sprintf("/integrations?limit=%d&offset=%d", listPageSize, offset)
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp, nil); err != nil {
			return nil, fmt.Errorf("failed to list integrations: %w", err)
		}
		all = append(all, resp.Data.Integrations...)
		if len(resp.Data.Integrations) < listPageSize {
			break
		}
	}
	return all, nil
}

// cursorPageSize is the maximum page size accepted by the cursor-paginated
// watcher and automation list endpoints.
const cursorPageSize = 100

// CreateWatcher creates a new watcher agent. The watcher starts in the draft
// state; call AddWatcherQueries with FinalizeCalibration set before starting it.
func (c *Client) CreateWatcher(ctx context.Context, body WatcherCreate) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	if err := c.doAPIRequest(ctx, http.MethodPost, "/watcher-agents", body, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to create watcher: %w", err)
	}
	return resp.Data, nil
}

// GetWatcher retrieves a watcher by ID.
func (c *Client) GetWatcher(ctx context.Context, id string) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	if err := c.doAPIRequest(ctx, http.MethodGet, "/watcher-agents/"+url.PathEscape(id), nil, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to get watcher %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateWatcher updates an existing watcher. A non-nil Queries replaces the
// whole editable query list.
func (c *Client) UpdateWatcher(ctx context.Context, id string, body WatcherUpdate) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/watcher-agents/"+url.PathEscape(id), body, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to update watcher %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteWatcher deletes a watcher by ID.
func (c *Client) DeleteWatcher(ctx context.Context, id string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/watcher-agents/"+url.PathEscape(id), nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete watcher %q: %w", id, err)
	}
	return nil
}

// AddWatcherQueries appends queries to a watcher's check set. With
// FinalizeCalibration set, this marks the watcher calibrated and moves a draft
// watcher to ready without an interactive calibration session.
func (c *Client) AddWatcherQueries(ctx context.Context, id string, body WatcherAddQueries) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	path := "/watcher-agents/" + url.PathEscape(id) + "/queries"
	if err := c.doAPIRequest(ctx, http.MethodPost, path, body, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to add queries to watcher %q: %w", id, err)
	}
	return resp.Data, nil
}

// StartWatcher turns on scheduled runs for a calibrated watcher.
func (c *Client) StartWatcher(ctx context.Context, id string) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	path := "/watcher-agents/" + url.PathEscape(id) + "/start"
	if err := c.doAPIRequest(ctx, http.MethodPost, path, nil, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to start watcher %q: %w", id, err)
	}
	return resp.Data, nil
}

// PauseWatcher stops future scheduled runs without deleting the watcher.
func (c *Client) PauseWatcher(ctx context.Context, id string) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	path := "/watcher-agents/" + url.PathEscape(id) + "/pause"
	if err := c.doAPIRequest(ctx, http.MethodPost, path, nil, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to pause watcher %q: %w", id, err)
	}
	return resp.Data, nil
}

// ListWatchers returns all watchers visible to the caller, following the
// cursor until the API stops returning one.
func (c *Client) ListWatchers(ctx context.Context) ([]Watcher, error) {
	var all []Watcher
	cursor := ""
	for {
		var resp apiResponseWrapper[watcherListData]
		path := fmt.Sprintf("/watcher-agents?page_size=%d", cursorPageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp, nil); err != nil {
			return nil, fmt.Errorf("failed to list watchers: %w", err)
		}
		all = append(all, resp.Data.Agents...)
		if resp.Data.NextCursor == "" || len(resp.Data.Agents) == 0 {
			break
		}
		cursor = resp.Data.NextCursor
	}
	return all, nil
}

// CreateAutomation creates a new automation.
func (c *Client) CreateAutomation(ctx context.Context, body AutomationCreate) (Automation, error) {
	var resp apiResponseWrapper[Automation]
	if err := c.doAPIRequest(ctx, http.MethodPost, "/automations", body, &resp, nil); err != nil {
		return Automation{}, fmt.Errorf("failed to create automation: %w", err)
	}
	return resp.Data, nil
}

// GetAutomation retrieves an automation by ID.
func (c *Client) GetAutomation(ctx context.Context, id string) (Automation, error) {
	var resp apiResponseWrapper[Automation]
	if err := c.doAPIRequest(ctx, http.MethodGet, "/automations/"+url.PathEscape(id), nil, &resp, nil); err != nil {
		return Automation{}, fmt.Errorf("failed to get automation %q: %w", id, err)
	}
	return resp.Data, nil
}

// UpdateAutomation updates an existing automation. resourceScope must be the
// automation's current scope: the plugin permission layer reads it from the
// X-Resource-Scope header to authorize the write, and rejects a value that does
// not match what is stored.
func (c *Client) UpdateAutomation(ctx context.Context, id, resourceScope string, body AutomationUpdate) (Automation, error) {
	var resp apiResponseWrapper[Automation]
	if err := c.doAPIRequest(ctx, http.MethodPut, "/automations/"+url.PathEscape(id), body, &resp, scopeHeader(resourceScope)); err != nil {
		return Automation{}, fmt.Errorf("failed to update automation %q: %w", id, err)
	}
	return resp.Data, nil
}

// DeleteAutomation deletes an automation by ID. resourceScope must be the
// automation's current scope; the delete route is scope-aware and the plugin
// permission layer requires the X-Resource-Scope header.
func (c *Client) DeleteAutomation(ctx context.Context, id, resourceScope string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/automations/"+url.PathEscape(id), nil, nil, scopeHeader(resourceScope)); err != nil {
		return fmt.Errorf("failed to delete automation %q: %w", id, err)
	}
	return nil
}

// ListAutomations returns all automations visible to the caller, following the
// cursor until the API stops returning one.
func (c *Client) ListAutomations(ctx context.Context) ([]Automation, error) {
	var all []Automation
	cursor := ""
	for {
		var resp apiResponseWrapper[automationListData]
		path := fmt.Sprintf("/automations?page_size=%d", cursorPageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp, nil); err != nil {
			return nil, fmt.Errorf("failed to list automations: %w", err)
		}
		all = append(all, resp.Data.Automations...)
		if resp.Data.NextCursor == "" || len(resp.Data.Automations) == 0 {
			break
		}
		cursor = resp.Data.NextCursor
	}
	return all, nil
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

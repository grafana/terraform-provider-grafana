// Package agento11yapi provides a thin HTTP client for the Grafana Agent
// Observability (Sigil) evaluation control plane, reached through the
// grafana-agento11y-app plugin backend proxy. The plugin injects tenant and
// trusted-actor identity headers on write requests, so this client only needs
// to authenticate as a Grafana user (basic auth or bearer token), exactly like
// the assistant plugin client.
package agento11yapi

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

// pathPrefix is the plugin resource route that fronts the Sigil
// `/api/v1/eval/*` control plane. The plugin backend strips the
// `/api/plugins/grafana-agento11y-app/resources` prefix before routing, and
// maps `/eval/...` to the upstream `/api/v1/eval/...` paths.
const pathPrefix = "/api/plugins/grafana-agento11y-app/resources/eval"

// listPageSize is the page size requested when paginating list endpoints.
const listPageSize = 100

// Client talks to the agent observability evaluation control plane via the
// grafana-agento11y-app plugin API.
type Client struct {
	baseURL        url.URL
	basicAuth      *url.Userinfo
	apiKey         string
	client         *http.Client
	userAgent      string
	defaultHeaders map[string]string
}

var (
	// ErrNotFound is returned when the API responds with 404.
	ErrNotFound = errors.New("not found")
	// ErrUnauthorized is returned when the API responds with 401.
	ErrUnauthorized = errors.New("unauthorized")
)

// NewClient creates a client for the agent observability plugin API at the given Grafana URL.
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

func (c *Client) doAPIRequest(ctx context.Context, method, path string, body any, responseData any) error {
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

// Evaluators.

// CreateEvaluator creates or updates an evaluator definition.
func (c *Client) CreateEvaluator(ctx context.Context, body EvaluatorWrite) (Evaluator, error) {
	var resp Evaluator
	if err := c.doAPIRequest(ctx, http.MethodPost, "/evaluators", body, &resp); err != nil {
		return Evaluator{}, fmt.Errorf("failed to create evaluator: %w", err)
	}
	return resp, nil
}

// GetEvaluator retrieves the latest active version of an evaluator by ID.
func (c *Client) GetEvaluator(ctx context.Context, id string) (Evaluator, error) {
	var resp Evaluator
	if err := c.doAPIRequest(ctx, http.MethodGet, "/evaluators/"+url.PathEscape(id), nil, &resp); err != nil {
		return Evaluator{}, fmt.Errorf("failed to get evaluator %q: %w", id, err)
	}
	return resp, nil
}

// DeleteEvaluator soft-deletes an evaluator by ID.
func (c *Client) DeleteEvaluator(ctx context.Context, id string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/evaluators/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("failed to delete evaluator %q: %w", id, err)
	}
	return nil
}

// ListEvaluators returns all tenant evaluators, paginating through every page.
func (c *Client) ListEvaluators(ctx context.Context) ([]Evaluator, error) {
	var all []Evaluator
	cursor := ""
	for {
		var resp listResponse[Evaluator]
		path := fmt.Sprintf("/evaluators?limit=%d", listPageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, fmt.Errorf("failed to list evaluators: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}
	return all, nil
}

// Rules.

// CreateRule creates or updates an evaluation rule.
func (c *Client) CreateRule(ctx context.Context, body RuleWrite) (Rule, error) {
	var resp Rule
	if err := c.doAPIRequest(ctx, http.MethodPost, "/rules", body, &resp); err != nil {
		return Rule{}, fmt.Errorf("failed to create rule: %w", err)
	}
	return resp, nil
}

// GetRule retrieves an evaluation rule by ID.
func (c *Client) GetRule(ctx context.Context, id string) (Rule, error) {
	var resp Rule
	if err := c.doAPIRequest(ctx, http.MethodGet, "/rules/"+url.PathEscape(id), nil, &resp); err != nil {
		return Rule{}, fmt.Errorf("failed to get rule %q: %w", id, err)
	}
	return resp, nil
}

// UpdateRule partially updates an evaluation rule.
func (c *Client) UpdateRule(ctx context.Context, id string, body RulePatch) (Rule, error) {
	var resp Rule
	if err := c.doAPIRequest(ctx, http.MethodPatch, "/rules/"+url.PathEscape(id), body, &resp); err != nil {
		return Rule{}, fmt.Errorf("failed to update rule %q: %w", id, err)
	}
	return resp, nil
}

// DeleteRule soft-deletes an evaluation rule by ID.
func (c *Client) DeleteRule(ctx context.Context, id string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/rules/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("failed to delete rule %q: %w", id, err)
	}
	return nil
}

// ListRules returns all evaluation rules, paginating through every page.
func (c *Client) ListRules(ctx context.Context) ([]Rule, error) {
	var all []Rule
	cursor := ""
	for {
		var resp listResponse[Rule]
		path := fmt.Sprintf("/rules?limit=%d", listPageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, fmt.Errorf("failed to list rules: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}
	return all, nil
}

// Rule actions.

// CreateRuleAction creates a rule action under the given rule.
func (c *Client) CreateRuleAction(ctx context.Context, ruleID string, body RuleActionCreate) (RuleAction, error) {
	var resp RuleAction
	path := "/rules/" + url.PathEscape(ruleID) + "/actions"
	if err := c.doAPIRequest(ctx, http.MethodPost, path, body, &resp); err != nil {
		return RuleAction{}, fmt.Errorf("failed to create rule action: %w", err)
	}
	return resp, nil
}

// GetRuleAction retrieves a rule action by rule ID and action ID.
func (c *Client) GetRuleAction(ctx context.Context, ruleID, actionID string) (RuleAction, error) {
	var resp RuleAction
	path := "/rules/" + url.PathEscape(ruleID) + "/actions/" + url.PathEscape(actionID)
	if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return RuleAction{}, fmt.Errorf("failed to get rule action %q: %w", actionID, err)
	}
	return resp, nil
}

// UpdateRuleAction patches a rule action.
func (c *Client) UpdateRuleAction(ctx context.Context, ruleID, actionID string, body RuleActionUpdate) (RuleAction, error) {
	var resp RuleAction
	path := "/rules/" + url.PathEscape(ruleID) + "/actions/" + url.PathEscape(actionID)
	if err := c.doAPIRequest(ctx, http.MethodPatch, path, body, &resp); err != nil {
		return RuleAction{}, fmt.Errorf("failed to update rule action %q: %w", actionID, err)
	}
	return resp, nil
}

// DeleteRuleAction deletes a rule action.
func (c *Client) DeleteRuleAction(ctx context.Context, ruleID, actionID string) error {
	path := "/rules/" + url.PathEscape(ruleID) + "/actions/" + url.PathEscape(actionID)
	if err := c.doAPIRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete rule action %q: %w", actionID, err)
	}
	return nil
}

// ListRuleActions returns all actions attached to a rule.
func (c *Client) ListRuleActions(ctx context.Context, ruleID string) ([]RuleAction, error) {
	var resp itemsResponse[RuleAction]
	path := "/rules/" + url.PathEscape(ruleID) + "/actions"
	if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to list rule actions for rule %q: %w", ruleID, err)
	}
	return resp.Items, nil
}

// Collections.

// CreateCollection creates a collection.
func (c *Client) CreateCollection(ctx context.Context, body CollectionCreate) (Collection, error) {
	var resp Collection
	if err := c.doAPIRequest(ctx, http.MethodPost, "/collections", body, &resp); err != nil {
		return Collection{}, fmt.Errorf("failed to create collection: %w", err)
	}
	return resp, nil
}

// GetCollection retrieves a collection by ID.
func (c *Client) GetCollection(ctx context.Context, id string) (Collection, error) {
	var resp Collection
	if err := c.doAPIRequest(ctx, http.MethodGet, "/collections/"+url.PathEscape(id), nil, &resp); err != nil {
		return Collection{}, fmt.Errorf("failed to get collection %q: %w", id, err)
	}
	return resp, nil
}

// UpdateCollection updates the name and description of a collection.
func (c *Client) UpdateCollection(ctx context.Context, id string, body CollectionPatch) (Collection, error) {
	var resp Collection
	if err := c.doAPIRequest(ctx, http.MethodPatch, "/collections/"+url.PathEscape(id), body, &resp); err != nil {
		return Collection{}, fmt.Errorf("failed to update collection %q: %w", id, err)
	}
	return resp, nil
}

// DeleteCollection deletes a collection by ID.
func (c *Client) DeleteCollection(ctx context.Context, id string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/collections/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("failed to delete collection %q: %w", id, err)
	}
	return nil
}

// ListCollections returns all tenant collections, paginating through every page.
func (c *Client) ListCollections(ctx context.Context) ([]Collection, error) {
	var all []Collection
	cursor := ""
	for {
		var resp listResponse[Collection]
		path := fmt.Sprintf("/collections?limit=%d", listPageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, fmt.Errorf("failed to list collections: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}
	return all, nil
}

// Hook rules.

// CreateHookRule creates a synchronous hook (guard) rule.
func (c *Client) CreateHookRule(ctx context.Context, body HookRuleWrite) (HookRule, error) {
	var resp HookRule
	if err := c.doAPIRequest(ctx, http.MethodPost, "/hook-rules", body, &resp); err != nil {
		return HookRule{}, fmt.Errorf("failed to create hook rule: %w", err)
	}
	return resp, nil
}

// GetHookRule retrieves a hook rule by ID.
func (c *Client) GetHookRule(ctx context.Context, id string) (HookRule, error) {
	var resp HookRule
	if err := c.doAPIRequest(ctx, http.MethodGet, "/hook-rules/"+url.PathEscape(id), nil, &resp); err != nil {
		return HookRule{}, fmt.Errorf("failed to get hook rule %q: %w", id, err)
	}
	return resp, nil
}

// UpsertHookRule creates or updates a hook rule by ID (PUT semantics).
func (c *Client) UpsertHookRule(ctx context.Context, id string, body HookRuleWrite) (HookRule, error) {
	var resp HookRule
	if err := c.doAPIRequest(ctx, http.MethodPut, "/hook-rules/"+url.PathEscape(id), body, &resp); err != nil {
		return HookRule{}, fmt.Errorf("failed to update hook rule %q: %w", id, err)
	}
	return resp, nil
}

// DeleteHookRule deletes a hook rule by ID.
func (c *Client) DeleteHookRule(ctx context.Context, id string) error {
	if err := c.doAPIRequest(ctx, http.MethodDelete, "/hook-rules/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("failed to delete hook rule %q: %w", id, err)
	}
	return nil
}

// ListHookRules returns all hook rules, paginating through every page.
func (c *Client) ListHookRules(ctx context.Context) ([]HookRule, error) {
	var all []HookRule
	cursor := ""
	for {
		var resp listResponse[HookRule]
		path := fmt.Sprintf("/hook-rules?limit=%d", listPageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		if err := c.doAPIRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, fmt.Errorf("failed to list hook rules: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}
	return all, nil
}

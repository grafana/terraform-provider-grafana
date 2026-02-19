package frontendo11yapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

var _ json.Unmarshaler = &App{}

type Client struct {
	apiURL         string
	authToken      string
	client         *http.Client
	cloudAPIHost   string
	userAgent      string
	defaultHeaders map[string]string
}

const (
	defaultRetries = 3
	defaultTimeout = 90 * time.Second
	pathPrefix     = "/api/v1"
)

func NewClient(feo11yAPIURL, cloudAPIHost, authToken string, client *http.Client, userAgent string, defaultHeaders map[string]string) (*Client, error) {
	if client == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = defaultRetries
		client = retryClient.StandardClient()
		client.Timeout = defaultTimeout
	}

	return &Client{
		apiURL:         feo11yAPIURL,
		authToken:      authToken,
		client:         client,
		cloudAPIHost:   cloudAPIHost,
		userAgent:      userAgent,
		defaultHeaders: defaultHeaders,
	}, nil
}

// LogLabels ...
type LogLabel struct {
	ID    int64  `json:"id,omitempty"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// AllowedOrigin ...
type AllowedOrigin struct {
	ID  int64  `json:"id,omitempty"`
	URL string `json:"url"`
}

type AppSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// App model
type App struct {
	ID                 int64             `json:"id,omitempty"`
	Name               string            `json:"name,omitempty"`
	Key                string            `json:"appKey,omitempty"`
	ExtraLogLabels     []LogLabel        `json:"extraLogLabels,omitempty"`
	CORSAllowedOrigins []AllowedOrigin   `json:"corsOrigins,omitempty"`
	AllowedRate        uint64            `json:"allowedRate,omitempty"`
	Settings           map[string]string `json:"settings,omitempty"`
	CollectEndpointURL string            `json:"collectEndpointURL,omitempty"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

func (c *Client) Host() string {
	return c.cloudAPIHost
}

// faroEndpointUrlsRegionExceptions contains hardcoded URLs for specific regions
// TODO: make faroEndpointUrl visible in gcom response
var faroEndpointUrlsRegionExceptions = map[string]string{
	"au":       "https://faro-api-prod-au-southeast-0.grafana.net/faro",
	"eu":       "https://faro-api-prod-eu-west-0.grafana.net/faro",
	"us-azure": "https://faro-api-prod-us-central-7.grafana.net/faro",
	"us":       "https://faro-api-prod-us-central-0.grafana.net/faro",
}

type faroEndpointURLRegionCutoff struct {
	cutoffDate      time.Time
	faroEndpointURL string // URL to use after the cutoff date
}

var faroEndpointURLsAfterCutoff = map[string]faroEndpointURLRegionCutoff{
	"prod-us-east-0": {
		cutoffDate:      time.Date(2024, 12, 18, 0, 0, 0, 0, time.UTC),
		faroEndpointURL: "https://faro-api-prod-us-east-2.grafana.net/faro",
	},
}

// FaroEndpointURL returns the Faro API endpoint URL for a given region and stack creation date.
// Some regions have hardcoded exception URLs, and some regions have different URLs based on
// when the stack was created.
func (c *Client) FaroEndpointURL(regionSlug string, createdAt time.Time) string {
	// The URL is manually supplied.
	if c.apiURL != "" {
		return c.apiURL
	}

	if cutoffInfo, ok := faroEndpointURLsAfterCutoff[regionSlug]; ok {
		if createdAt.After(cutoffInfo.cutoffDate) {
			return cutoffInfo.faroEndpointURL
		}
	}

	if url, ok := faroEndpointUrlsRegionExceptions[regionSlug]; ok {
		return url
	}

	return fmt.Sprintf("https://faro-api-%s.%s/faro", regionSlug, c.cloudAPIHost)
}

func (c *Client) CreateApp(ctx context.Context, baseURL string, stackID int64, appData App) (App, error) {
	path := fmt.Sprintf("%s/app", pathPrefix)
	var app App
	err := c.doAPIRequest(ctx, baseURL, stackID, http.MethodPost, path, &appData, &app)
	if err != nil {
		return App{}, fmt.Errorf("failed to create faro app %q: %w", appData.Name, err)
	}
	return app, nil
}

func (c *Client) GetApps(ctx context.Context, baseURL string, stackID int64) ([]App, error) {
	path := fmt.Sprintf("%s/app", pathPrefix)
	var apps []App
	err := c.doAPIRequest(ctx, baseURL, stackID, http.MethodGet, path, nil, &apps)
	if err != nil {
		return []App{}, fmt.Errorf("failed to get faro apps: %w", err)
	}
	return apps, nil
}

func (c *Client) GetApp(ctx context.Context, baseURL string, stackID int64, appID int64) (App, error) {
	path := fmt.Sprintf("%s/app/%d", pathPrefix, appID)
	var app App
	err := c.doAPIRequest(ctx, baseURL, stackID, http.MethodGet, path, nil, &app)
	if err != nil {
		return App{}, fmt.Errorf("failed to get faro app: %w", err)
	}
	return app, nil
}

func (c *Client) UpdateApp(ctx context.Context, baseURL string, stackID int64, appID int64, appData App) (App, error) {
	path := fmt.Sprintf("%s/app/%d", pathPrefix, appID)
	var app App
	err := c.doAPIRequest(ctx, baseURL, stackID, http.MethodPut, path, &appData, &app)
	if err != nil {
		return App{}, fmt.Errorf("failed to update faro app %q: %w", appData.Name, err)
	}
	return app, nil
}

func (c *Client) DeleteApp(ctx context.Context, baseURL string, stackID int64, appID int64) error {
	path := fmt.Sprintf("%s/app/%d", pathPrefix, appID)
	err := c.doAPIRequest(ctx, baseURL, stackID, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete faro app id=%d: %w", appID, err)
	}
	return nil
}

var (
	ErrNotFound     = fmt.Errorf("not found")
	ErrUnauthorized = fmt.Errorf("request not authorized for stack")
)

func (c *Client) doAPIRequest(ctx context.Context, rawURL string, stackID int64, method string, path string, body any, responseData any) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse frontend o11y API url: %w", err)
	}

	var reqBodyBytes io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBodyBytes = bytes.NewReader(bs)
	}

	req, err := http.NewRequestWithContext(ctx, method, parsedURL.String()+path, reqBodyBytes)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.defaultHeaders {
		req.Header.Add(k, v)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %d:%s", stackID, c.authToken))
	req.Header.Add("X-Scope-OrgID", fmt.Sprintf("%d", stackID))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	bodyContents, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode >= 400:
		return fmt.Errorf("status: %d", resp.StatusCode)
	case responseData == nil || resp.StatusCode == http.StatusNoContent:
		return nil
	}

	err = json.Unmarshal(bodyContents, &responseData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return nil
}

func (a *App) UnmarshalJSON(input []byte) error {
	// We use a type alias here to avoid
	// 1. a stack overflow from recursively calling App.UnmarshalJSON() (see: https://groups.google.com/g/golang-nuts/c/o22NS2yTe2A)
	// 2. needing to pull individual fields out of a map[string]interface{}
	type RawApp App

	raw := &RawApp{}

	err := json.Unmarshal(input, raw)
	if err != nil {
		return err
	}

	corsSet := make(map[string]struct{})
	for _, corsOrigin := range raw.CORSAllowedOrigins {
		normalizedURL := strings.ToLower(corsOrigin.URL)
		if _, ok := corsSet[normalizedURL]; ok {
			continue
		}

		a.CORSAllowedOrigins = append(a.CORSAllowedOrigins, corsOrigin)
		corsSet[corsOrigin.URL] = struct{}{}
	}

	a.ID = raw.ID
	a.Name = raw.Name
	a.Key = raw.Key
	a.ExtraLogLabels = raw.ExtraLogLabels
	a.Settings = raw.Settings
	a.AllowedRate = raw.AllowedRate
	a.CreatedAt = raw.CreatedAt
	a.UpdatedAt = raw.UpdatedAt
	a.DeletedAt = raw.DeletedAt
	a.CollectEndpointURL = raw.CollectEndpointURL

	return nil
}

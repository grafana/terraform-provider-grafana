package gapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

// DashboardMeta represents Grafana dashboard meta.
type DashboardMeta struct {
	IsStarred bool   `json:"isStarred"`
	Slug      string `json:"slug"`
	Folder    int64  `json:"folderId"`
}

// DashboardSaveResponse represents the Grafana API response to creating or saving a dashboard.
type DashboardSaveResponse struct {
	Slug    string `json:"slug"`
	ID      int64  `json:"id"`
	UID     string `json:"uid"`
	Status  string `json:"status"`
	Version int64  `json:"version"`
}

// DashboardSearchResponse represents the Grafana API dashboard search response.
type DashboardSearchResponse struct {
	ID          uint     `json:"id"`
	UID         string   `json:"uid"`
	Title       string   `json:"title"`
	URI         string   `json:"uri"`
	URL         string   `json:"url"`
	Slug        string   `json:"slug"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags"`
	IsStarred   bool     `json:"isStarred"`
	FolderID    uint     `json:"folderId"`
	FolderUID   string   `json:"folderUid"`
	FolderTitle string   `json:"folderTitle"`
	FolderURL   string   `json:"folderUrl"`
}

// Dashboard represents a Grafana dashboard.
type Dashboard struct {
	Meta      DashboardMeta          `json:"meta"`
	Model     map[string]interface{} `json:"dashboard"`
	Folder    int64                  `json:"folderId"`
	Overwrite bool                   `json:"overwrite"`
}

// SaveDashboard is a deprecated method for saving a Grafana dashboard. Use NewDashboard.
// Deprecated: Use NewDashboard instead.
func (c *Client) SaveDashboard(model map[string]interface{}, overwrite bool) (*DashboardSaveResponse, error) {
	wrapper := map[string]interface{}{
		"dashboard": model,
		"overwrite": overwrite,
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return nil, err
	}

	result := &DashboardSaveResponse{}
	err = c.request("POST", "/api/dashboards/db", nil, bytes.NewBuffer(data), &result)
	if err != nil {
		return nil, err
	}

	return result, err
}

// NewDashboard creates a new Grafana dashboard.
func (c *Client) NewDashboard(dashboard Dashboard) (*DashboardSaveResponse, error) {
	data, err := json.Marshal(dashboard)
	if err != nil {
		return nil, err
	}

	result := &DashboardSaveResponse{}
	err = c.request("POST", "/api/dashboards/db", nil, bytes.NewBuffer(data), &result)
	if err != nil {
		return nil, err
	}

	return result, err
}

// Dashboards fetches and returns Grafana dashboards.
func (c *Client) Dashboards() ([]DashboardSearchResponse, error) {
	dashboards := make([]DashboardSearchResponse, 0)
	query := url.Values{}
	// search only dashboards
	query.Add("type", "dash-db")

	err := c.request("GET", "/api/search", query, nil, &dashboards)
	if err != nil {
		return nil, err
	}

	return dashboards, err
}

// Dashboard will be removed.
// Deprecated: Starting from Grafana v5.0. Use DashboardByUID instead.
func (c *Client) Dashboard(slug string) (*Dashboard, error) {
	return c.dashboard(fmt.Sprintf("/api/dashboards/db/%s", slug))
}

// DashboardByUID gets a dashboard by UID.
func (c *Client) DashboardByUID(uid string) (*Dashboard, error) {
	return c.dashboard(fmt.Sprintf("/api/dashboards/uid/%s", uid))
}

func (c *Client) dashboard(path string) (*Dashboard, error) {
	result := &Dashboard{}
	err := c.request("GET", path, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	result.Folder = result.Meta.Folder

	return result, err
}

// DeleteDashboard will be removed.
// Deprecated: Starting from Grafana v5.0. Use DeleteDashboardByUID instead.
func (c *Client) DeleteDashboard(slug string) error {
	return c.deleteDashboard(fmt.Sprintf("/api/dashboards/db/%s", slug))
}

// DeleteDashboardByUID deletes a dashboard by UID.
func (c *Client) DeleteDashboardByUID(uid string) error {
	return c.deleteDashboard(fmt.Sprintf("/api/dashboards/uid/%s", uid))
}

func (c *Client) deleteDashboard(path string) error {
	return c.request("DELETE", path, nil, nil, nil)
}

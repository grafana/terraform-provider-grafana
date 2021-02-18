package gapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DashboardPermission has information such as a dashboard, user, team, role and permission.
type DashboardPermission struct {
	DashboardID  int64  `json:"dashboardId"`
	DashboardUID string `json:"uid"`
	UserID       int64  `json:"userId"`
	TeamID       int64  `json:"teamId"`
	Role         string `json:"role"`
	IsFolder     bool   `json:"isFolder"`
	Inherited    bool   `json:"inherited"`

	// Permission levels are
	// 1 = View
	// 2 = Edit
	// 4 = Admin
	Permission     int64  `json:"permission"`
	PermissionName string `json:"permissionName"`
}

// DashboardPermissions fetches and returns the permissions for the dashboard whose ID it's passed.
func (c *Client) DashboardPermissions(id int64) ([]*DashboardPermission, error) {
	permissions := make([]*DashboardPermission, 0)
	err := c.request("GET", fmt.Sprintf("/api/dashboards/id/%d/permissions", id), nil, nil, &permissions)
	if err != nil {
		return permissions, err
	}

	return permissions, nil
}

// UpdateDashboardPermissions remove existing permissions if items are not included in the request.
func (c *Client) UpdateDashboardPermissions(id int64, items *PermissionItems) error {
	path := fmt.Sprintf("/api/dashboards/id/%d/permissions", id)
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return c.request("POST", path, nil, bytes.NewBuffer(data), nil)
}

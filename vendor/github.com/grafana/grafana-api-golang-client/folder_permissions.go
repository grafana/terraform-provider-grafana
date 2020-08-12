package gapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// FolderPermission has information such as a folder, user, team, role and permission.
type FolderPermission struct {
	Id        int64  `json:"id"`
	FolderUid string `json:"uid"`
	UserId    int64  `json:"userId"`
	TeamId    int64  `json:"teamId"`
	Role      string `json:"role"`
	IsFolder  bool   `json:"isFolder"`

	// Permission levels are
	// 1 = View
	// 2 = Edit
	// 4 = Admin
	Permission     int64  `json:"permission"`
	PermissionName string `json:"permissionName"`

	// optional fields
	FolderId    int64 `json:"folderId,omitempty"`
	DashboardId int64 `json:"dashboardId,omitempty"`
}

// PermissionItems represents Grafana folder permission items.
type PermissionItems struct {
	Items []*PermissionItem `json:"items"`
}

// PermissionItem represents a Grafana folder permission item.
type PermissionItem struct {
	// As you can see the docs, each item has a pair of [Role|TeamId|UserId] and Permission.
	// unnecessary fields are omitted.
	Role       string `json:"role,omitempty"`
	TeamId     int64  `json:"teamId,omitempty"`
	UserId     int64  `json:"userId,omitempty"`
	Permission int64  `json:"permission"`
}

// FolderPermissions fetches and returns the permissions for the folder whose ID it's passed.
func (c *Client) FolderPermissions(fid string) ([]*FolderPermission, error) {
	permissions := make([]*FolderPermission, 0)
	err := c.request("GET", fmt.Sprintf("/api/folders/%s/permissions", fid), nil, nil, &permissions)
	if err != nil {
		return permissions, err
	}

	return permissions, nil
}

// UpdateFolderPermissions remove existing permissions if items are not included in the request.
func (c *Client) UpdateFolderPermissions(fid string, items *PermissionItems) error {
	path := fmt.Sprintf("/api/folders/%s/permissions", fid)
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return c.request("POST", path, nil, bytes.NewBuffer(data), nil)
}

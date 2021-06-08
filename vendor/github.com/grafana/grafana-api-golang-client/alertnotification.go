package gapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// AlertNotification represents a Grafana alert notification.
type AlertNotification struct {
	ID                    int64       `json:"id,omitempty"`
	UID                   string      `json:"uid"`
	Name                  string      `json:"name"`
	Type                  string      `json:"type"`
	IsDefault             bool        `json:"isDefault"`
	DisableResolveMessage bool        `json:"disableResolveMessage"`
	SendReminder          bool        `json:"sendReminder"`
	Frequency             string      `json:"frequency"`
	Settings              interface{} `json:"settings"`
	SecureFields          interface{} `json:"secureFields,omitempty"`
	SecureSettings        interface{} `json:"secureSettings,omitempty"`
}

// AlertNotifications fetches and returns Grafana alert notifications.
func (c *Client) AlertNotifications() ([]AlertNotification, error) {
	alertnotifications := make([]AlertNotification, 0)

	err := c.request("GET", "/api/alert-notifications/", nil, nil, &alertnotifications)
	if err != nil {
		return nil, err
	}

	return alertnotifications, err
}

// AlertNotification fetches and returns a Grafana alert notification.
func (c *Client) AlertNotification(id int64) (*AlertNotification, error) {
	path := fmt.Sprintf("/api/alert-notifications/%d", id)
	result := &AlertNotification{}
	err := c.request("GET", path, nil, nil, result)
	if err != nil {
		return nil, err
	}

	return result, err
}

// NewAlertNotification creates a new Grafana alert notification.
func (c *Client) NewAlertNotification(a *AlertNotification) (int64, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return 0, err
	}
	result := struct {
		ID int64 `json:"id"`
	}{}

	err = c.request("POST", "/api/alert-notifications", nil, bytes.NewBuffer(data), &result)
	if err != nil {
		return 0, err
	}

	return result.ID, err
}

// UpdateAlertNotification updates a Grafana alert notification.
func (c *Client) UpdateAlertNotification(a *AlertNotification) error {
	path := fmt.Sprintf("/api/alert-notifications/%d", a.ID)
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	err = c.request("PUT", path, nil, bytes.NewBuffer(data), nil)

	return err
}

// DeleteAlertNotification deletes a Grafana alert notification.
func (c *Client) DeleteAlertNotification(id int64) error {
	path := fmt.Sprintf("/api/alert-notifications/%d", id)

	return c.request("DELETE", path, nil, nil, nil)
}

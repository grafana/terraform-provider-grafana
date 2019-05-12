package gapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
)

type AlertNotification struct {
	Id        int64       `json:"id,omitempty"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	IsDefault bool        `json:"isDefault"`
	Settings  interface{} `json:"settings"`
}

func (c *Client) AlertNotification(id int64, orgID int64) (*AlertNotification, error) {
	path := fmt.Sprintf("/api/alert-notifications/%d", id)
	req, err := c.newRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Grafana-Org-Id", strconv.FormatInt(orgID, 10))

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &AlertNotification{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *Client) NewAlertNotification(a *AlertNotification, orgID int64) (int64, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return 0, err
	}
	req, err := c.newRequest("POST", "/api/alert-notifications", nil, bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}

	req.Header.Set("X-Grafana-Org-Id", strconv.FormatInt(orgID, 10))

	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != 200 {
		return 0, errors.New(resp.Status)
	}

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	result := struct {
		Id int64 `json:"id"`
	}{}
	err = json.Unmarshal(data, &result)
	return result.Id, err
}

func (c *Client) UpdateAlertNotification(a *AlertNotification, orgID int64) error {
	path := fmt.Sprintf("/api/alert-notifications/%d", a.Id)
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	req, err := c.newRequest("PUT", path, nil, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("X-Grafana-Org-Id", strconv.FormatInt(orgID, 10))

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	return nil
}

func (c *Client) DeleteAlertNotification(id int64, orgID int64) error {
	path := fmt.Sprintf("/api/alert-notifications/%d", id)
	req, err := c.newRequest("DELETE", path, nil, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Grafana-Org-Id", strconv.FormatInt(orgID, 10))

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	return nil
}

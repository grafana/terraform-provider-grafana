package gapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
)

type User struct {
	Id       int64  `json:"id,omitempty"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	IsAdmin  bool   `json:"isAdmin,omitempty"`
}

func (c *Client) Users() ([]User, error) {
	users := make([]User, 0)
	req, err := c.newRequest("GET", "/api/users", nil, nil)
	if err != nil {
		return users, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return users, err
	}
	if resp.StatusCode != 200 {
		return users, errors.New(resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return users, err
	}
	err = json.Unmarshal(data, &users)
	if err != nil {
		return users, err
	}
	return users, err
}

func (c *Client) UserByEmail(email string) (User, error) {
	user := User{}
	query := url.Values{}
	query.Add("loginOrEmail", email)
	req, err := c.newRequest("GET", "/api/users/lookup", query, nil)
	if err != nil {
		return user, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return user, err
	}
	if resp.StatusCode != 200 {
		return user, errors.New(resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return user, err
	}
	tmp := struct {
		Id       int64  `json:"id,omitempty"`
		Email    string `json:"email,omitempty"`
		Name     string `json:"name,omitempty"`
		Login    string `json:"login,omitempty"`
		Password string `json:"password,omitempty"`
		IsAdmin  bool   `json:"isGrafanaAdmin,omitempty"`
	}{}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return user, err
	}
	user = User(tmp)
	return user, err
}

// SwitchCurrentUserOrg will switch the current organisation of the signed in user
func (c *Client) SwitchCurrentUserOrg(orgID int64) error {
	req, err := c.newRequest("POST", fmt.Sprintf("/api/user/using/%d", orgID), nil, nil)

	_, err = c.Do(req)

	return err
}

// SwitchUserOrg will switch the current organisation of the given user ID (via basic auth) to
// the given organisation ID
func (c *Client) SwitchUserOrg(userID, orgID int64) error {
	req, err := c.newRequest("POST", fmt.Sprintf("/api/users/%d/using/%d", userID, orgID), nil, nil)

	_, err = c.Do(req)

	return err
}

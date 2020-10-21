package gapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// PlaylistItem represents a Grafana playlist item.
type PlaylistItem struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Order int    `json:"order"`
	Title string `json:"title"`
}

// Playlist represents a Grafana playlist.
type Playlist struct {
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Interval string         `json:"interval"`
	Items    []PlaylistItem `json:"items"`
}

// Playlist fetches and returns a Grafana playlist.
func (c *Client) Playlist(id int) (*Playlist, error) {
	path := fmt.Sprintf("/api/playlists/%d", id)
	playlist := &Playlist{}
	err := c.request("GET", path, nil, nil, playlist)
	if err != nil {
		return nil, err
	}

	return playlist, nil
}

// NewPlaylist creates a new Grafana playlist.
func (c *Client) NewPlaylist(playlist Playlist) (int, error) {
	data, err := json.Marshal(playlist)
	if err != nil {
		return 0, err
	}

	result := struct {
		ID int
	}{}

	err = c.request("POST", "/api/playlists", nil, bytes.NewBuffer(data), &result)
	if err != nil {
		return 0, err
	}

	return result.ID, nil
}

// UpdatePlaylist updates a Grafana playlist.
func (c *Client) UpdatePlaylist(playlist Playlist) error {
	path := fmt.Sprintf("/api/playlists/%d", playlist.ID)
	data, err := json.Marshal(playlist)
	if err != nil {
		return err
	}

	return c.request("PUT", path, nil, bytes.NewBuffer(data), nil)
}

// DeletePlaylist deletes the Grafana playlist whose ID it's passed.
func (c *Client) DeletePlaylist(id int) error {
	path := fmt.Sprintf("/api/playlists/%d", id)

	return c.request("DELETE", path, nil, nil, nil)
}

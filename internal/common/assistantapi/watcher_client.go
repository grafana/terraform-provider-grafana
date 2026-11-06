package assistantapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

// CreateWatcher creates a new watcher agent. The watcher starts in the draft
// state; unattended calibration is a separate recoverable operation.
func (c *Client) CreateWatcher(ctx context.Context, body WatcherCreate) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	if err := c.doWatcherMutation(ctx, http.MethodPost, "/watcher-agents", body, &resp, nil); err != nil {
		// A public creation request ID can be observed even when the POST
		// response was lost. Never retry creation blindly: older servers may
		// ignore requestId and allocate another Watcher on each POST.
		if body.RequestID != "" && ctx.Err() == nil && IsAmbiguousWriteError(err) {
			watchers, readErr := c.listWatchers(ctx, true)
			if readErr == nil {
				var recovered *Watcher
				for i := range watchers {
					if watchers[i].RequestID == body.RequestID {
						if recovered != nil {
							return Watcher{}, fmt.Errorf("multiple watchers match creation request %q; inspect them before retrying", body.RequestID)
						}
						recovered = &watchers[i]
					}
				}
				if recovered != nil {
					return c.GetWatcher(ctx, recovered.ID)
				}
			}
		}
		return Watcher{}, fmt.Errorf("failed to create watcher (requestId %q): %w", body.RequestID, err)
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

// UpdateWatcher patches the operator-owned watcher configuration.
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

// StartWatcher turns on scheduled runs for a calibrated watcher.
func (c *Client) StartWatcher(ctx context.Context, id string) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	path := "/watcher-agents/" + url.PathEscape(id) + "/start"
	if err := c.doWatcherMutation(ctx, http.MethodPost, path, nil, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to start watcher %q: %w", id, err)
	}
	return resp.Data, nil
}

// PauseWatcher stops future scheduled runs without deleting the watcher.
func (c *Client) PauseWatcher(ctx context.Context, id string) (Watcher, error) {
	var resp apiResponseWrapper[Watcher]
	path := "/watcher-agents/" + url.PathEscape(id) + "/pause"
	if err := c.doWatcherMutation(ctx, http.MethodPost, path, nil, &resp, nil); err != nil {
		return Watcher{}, fmt.Errorf("failed to pause watcher %q: %w", id, err)
	}
	return resp.Data, nil
}

// ListWatchers returns all watchers visible to the caller, following the
// cursor until the API stops returning one.
func (c *Client) ListWatchers(ctx context.Context) ([]Watcher, error) {
	return c.listWatchers(ctx, false)
}

func (c *Client) listWatchers(ctx context.Context, createdByMe bool) ([]Watcher, error) {
	var all []Watcher
	cursor := ""
	for {
		var resp apiResponseWrapper[watcherListData]
		path := "/watcher-agents?page_size=100"
		if createdByMe {
			path += "&createdByMe=true"
		}
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

// GetWatcherCalibration observes setup without initiating work.
func (c *Client) GetWatcherCalibration(ctx context.Context, id string) (WatcherCalibration, error) {
	var resp apiResponseWrapper[WatcherCalibration]
	err := c.doAPIRequest(ctx, http.MethodGet, "/watcher-agents/"+url.PathEscape(id)+"/initial-calibration", nil, &resp, nil)
	return resp.Data, err
}

// CalibrateWatcher starts unattended setup. This POST must not be blindly retried:
// the API deduplicates active attempts but a completed attempt can be replaced.
func (c *Client) CalibrateWatcher(ctx context.Context, id string, body WatcherCalibrationRequest) (WatcherCalibration, error) {
	var resp apiResponseWrapper[WatcherCalibration]
	err := c.doWatcherMutation(ctx, http.MethodPost, "/watcher-agents/"+url.PathEscape(id)+"/initial-calibration", body, &resp, nil)
	return resp.Data, err
}

// Watcher POSTs do not have a public replay key. Preserve the configured HTTP
// transport (TLS/proxy settings), but bypass retryablehttp for these mutations.
// This is a per-call copy; the shared client remains safe for concurrent reads.
func (c *Client) doWatcherMutation(ctx context.Context, method, path string, body, response any, headers map[string]string) error {
	once := *c
	httpClient := *c.client
	if retry, ok := httpClient.Transport.(*retryablehttp.RoundTripper); ok {
		if retry.Client != nil && retry.Client.HTTPClient != nil {
			httpClient.Transport = retry.Client.HTTPClient.Transport
		} else {
			httpClient.Transport = http.DefaultTransport
		}
	}
	once.client = &httpClient
	return once.doAPIRequest(ctx, method, path, body, response, headers)
}

// WatcherURL links to the generated checks without placing credentials or URL
// query parameters into Terraform state.
func (c *Client) WatcherURL(id string) string {
	base := c.baseURL
	base.User = nil
	base.RawQuery = ""
	base.Fragment = ""
	return strings.TrimRight(base.String(), "/") + "/a/grafana-assistant-app/watchers/" + url.PathEscape(id) + "/calibration"
}

// IsAmbiguousWriteError reports whether a transport or server failure may hide
// an accepted mutation. Definitive authorization/validation failures do not.
func IsAmbiguousWriteError(err error) bool {
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrNotFound) {
		return false
	}
	var status *statusError
	if errors.As(err, &status) {
		return status.code >= 500
	}
	return true // transport/read/decode failure can hide an accepted write
}

// Package smapi provides access to the Synthetic Monitoring API.
package smapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/grafana/synthetic-monitoring-api-go-client/model"

	"github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
)

var (
	// ErrAuthorizationTokenRequired is the error returned by client
	// calls that require an authorization token.
	//
	// Authorization tokens can be obtained using Install or Init.
	ErrAuthorizationTokenRequired = errors.New("authorization token required")

	// ErrCannotEncodeJSONRequest is the error returned if it's not
	// possible to encode the request as a JSON object. This error
	// should never happen.
	ErrCannotEncodeJSONRequest = errors.New("cannot encode request")
)

// Client is a Synthetic Monitoring API client.
//
// It should be initialized using the NewClient function in this package.
type Client struct {
	client      *http.Client
	accessToken string
	baseURL     string
}

// NewClient creates a new client for the Synthetic Monitoring API.
//
// The accessToken is optional. If it's not specified, it's necessary to
// use one of the registration calls to obtain one, Install or Init.
//
// If no client is provided, http.DefaultClient will be used.
func NewClient(baseURL, accessToken string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}

	u, err := url.Parse(baseURL + "/api/v1")
	if err != nil {
		return nil
	}

	u.Path = path.Clean(u.Path)

	return &Client{
		client:      client,
		accessToken: accessToken,
		baseURL:     u.String(),
	}
}

// NewDatasourceClient creates a new client for the Synthetic Monitoring API using a Grafana datasource proxy.
//
// The accessToken should be the grafana access token.
//
// If no client is provided, http.DefaultClient will be used.
func NewDatasourceClient(baseURL, accessToken string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}
	u.Path = strings.TrimSuffix(u.Path, "/")

	return &Client{
		client:      client,
		accessToken: accessToken,
		baseURL:     u.String(),
	}
}

// Install takes a stack ID, a hosted metrics instance ID, a hosted logs
// instance ID and a publisher token that can be used to publish data to those
// instances and sets up a new Synthetic Monitoring tenant using those
// parameters.
//
// Note that the client will not any validation on these arguments and it will
// simply pass them to the corresponding API server.
//
// The returned RegistrationInstallResponse will contain the access token used
// to make further calls to the API server. This call will _modify_ the client
// in order to use that access token.
func (h *Client) Install(ctx context.Context, stackID, metricsInstanceID, logsInstanceID int64, publisherToken string) (*model.RegistrationInstallResponse, error) {
	request := model.RegistrationInstallRequest{
		LogsInstanceID:    logsInstanceID,
		MetricsInstanceID: metricsInstanceID,
		StackID:           stackID,
	}

	buf, err := json.Marshal(&request)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling install request: %w", err)
	}

	body := bytes.NewReader(buf)

	headers := defaultHeaders()
	headers.Set("Authorization", "Bearer "+publisherToken)

	resp, err := h.post(ctx, "/register/install", false, headers, body)
	if err != nil {
		return nil, fmt.Errorf("sending install request: %w", err)
	}

	var result model.RegistrationInstallResponse

	if err := validateResponse("registration install request", resp, &result); err != nil {
		return nil, err
	}

	h.accessToken = result.AccessToken

	return &result, nil
}

// Init uses provided admin token to call the deprecated registration init
// entrypoint in order to create a Synthetic Monitoring tenant.
//
// Note that the client will not any validation on the provided token and it
// will simply pass it to the corresponding API server.
//
// The returned RegistrationInitResponse will contain the access token used to
// make further calls to the API server. This call will _modify_ the client in
// order to use that access token.
func (h *Client) Init(ctx context.Context, adminToken string) (*model.RegistrationInitResponse, error) {
	req := struct {
		APIToken string `json:"apiToken"`
	}{
		APIToken: adminToken,
	}

	resp, err := h.postJSON(ctx, "/register/init", false, &req)
	if err != nil {
		return nil, fmt.Errorf("sending init request: %w", err)
	}

	var result model.RegistrationInitResponse

	if err := validateResponse("registration init request", resp, &result); err != nil {
		return nil, err
	}

	h.accessToken = result.AccessToken

	return &result, nil
}

// Save should be called after Init to select the desired hosted metrics and
// hosted logs instance to be used by Synthetic Monitoring probes.
//
// After this call, the tenant is fully configured and can be used to create
// checks and probes.
//
// Just as Init, Save is deprecated.
func (h *Client) Save(ctx context.Context, adminToken string, metricInstanceID, logInstanceID int64) error {
	saveReq := struct {
		AdminToken        string `json:"apiToken"`
		MetricsInstanceID int64  `json:"metricsInstanceId"`
		LogsInstanceID    int64  `json:"logsInstanceId"`
	}{
		AdminToken:        adminToken,
		MetricsInstanceID: metricInstanceID,
		LogsInstanceID:    logInstanceID,
	}

	resp, err := h.postJSON(ctx, "/register/save", true, &saveReq)
	if err != nil {
		return fmt.Errorf("sending save request: %w", err)
	}

	var result struct{}

	if err := validateResponse("registration save request", resp, &result); err != nil {
		return err
	}

	return nil
}

// AddProbe is used to create a new Synthetic Monitoring probe.
//
// The return value includes the assigned probe ID as well as the access token
// that should be used by that probe to communicate with the Synthetic
// Monitoring API.
func (h *Client) AddProbe(ctx context.Context, probe synthetic_monitoring.Probe) (*synthetic_monitoring.Probe, []byte, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, nil, err
	}

	resp, err := h.postJSON(ctx, "/probe/add", true, &probe)
	if err != nil {
		return nil, nil, fmt.Errorf("adding probe: %w", err)
	}

	var result model.ProbeAddResponse

	if err := validateResponse("probe add request", resp, &result); err != nil {
		return nil, nil, err
	}

	return &result.Probe, result.Token, nil
}

// DeleteProbe is used to remove a new Synthetic Monitoring probe.
func (h *Client) DeleteProbe(ctx context.Context, id int64) error {
	if err := h.requireAuthToken(); err != nil {
		return err
	}

	resp, err := h.delete(ctx, fmt.Sprintf("%s%s/%d", h.baseURL, "/probe/delete", id), true)
	if err != nil {
		return fmt.Errorf("sending probe delete request: %w", err)
	}

	var result model.ProbeDeleteResponse

	if err := validateResponse("probe delete request", resp, &result); err != nil {
		return err
	}

	return nil
}

// UpdateProbe is used to update details about an existing Synthetic Monitoring
// probe.
//
// The return value contains the new representation of the probe according the
// Synthetic Monitoring API server.
func (h *Client) UpdateProbe(ctx context.Context, probe synthetic_monitoring.Probe) (*synthetic_monitoring.Probe, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, err
	}

	resp, err := h.postJSON(ctx, "/probe/update", true, &probe)
	if err != nil {
		return nil, fmt.Errorf("sending probe update request: %w", err)
	}

	var result model.ProbeUpdateResponse

	if err := validateResponse("probe update request", resp, &result); err != nil {
		return nil, err
	}

	return &result.Probe, nil
}

// ResetProbeToken requests a _new_ token for the probe.
func (h *Client) ResetProbeToken(ctx context.Context, probe synthetic_monitoring.Probe) (*synthetic_monitoring.Probe, []byte, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, nil, err
	}

	resp, err := h.postJSON(ctx, "/probe/update?reset-token", true, &probe)
	if err != nil {
		return nil, nil, fmt.Errorf("sending probe update request: %w", err)
	}

	var result model.ProbeUpdateResponse

	if err := validateResponse("probe update request", resp, &result); err != nil {
		return nil, nil, err
	}

	return &result.Probe, result.Token, nil
}

// ListProbes returns the list of probes accessible to the authenticated
// tenant.
func (h *Client) ListProbes(ctx context.Context) ([]synthetic_monitoring.Probe, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, err
	}

	resp, err := h.get(ctx, "/probe/list", true, nil)
	if err != nil {
		return nil, fmt.Errorf("sending probe list request: %w", err)
	}

	var result []synthetic_monitoring.Probe

	if err := validateResponse("probe list request", resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// AddCheck creates a new Synthetic Monitoring check in the API server.
//
// The return value contains the assigned ID.
func (h *Client) AddCheck(ctx context.Context, check synthetic_monitoring.Check) (*synthetic_monitoring.Check, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, err
	}

	resp, err := h.postJSON(ctx, "/check/add", true, &check)
	if err != nil {
		return nil, fmt.Errorf("sending check add request: %w", err)
	}

	var result synthetic_monitoring.Check

	if err := validateResponse("check add request", resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateCheck updates an existing check in the API server.
//
// The return value contains the updated check (updated timestamps,
// etc).
func (h *Client) UpdateCheck(ctx context.Context, check synthetic_monitoring.Check) (*synthetic_monitoring.Check, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, err
	}

	resp, err := h.postJSON(ctx, "/check/update", true, &check)
	if err != nil {
		return nil, fmt.Errorf("sending check update request: %w", err)
	}

	var result synthetic_monitoring.Check

	if err := validateResponse("check update request", resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteCheck deletes an existing Synthetic Monitoring check from the API
// server.
func (h *Client) DeleteCheck(ctx context.Context, id int64) error {
	if err := h.requireAuthToken(); err != nil {
		return err
	}

	resp, err := h.delete(ctx, fmt.Sprintf("%s%s/%d", h.baseURL, "/check/delete", id), true)
	if err != nil {
		return fmt.Errorf("sending check delete request: %w", err)
	}

	var result model.CheckDeleteResponse

	if err := validateResponse("check delete request", resp, &result); err != nil {
		return err
	}

	return nil
}

// ListChecks returns the list of Synthetic Monitoring checks for the
// authenticated tenant.
func (h *Client) ListChecks(ctx context.Context) ([]synthetic_monitoring.Check, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, err
	}

	resp, err := h.get(ctx, "/check/list", true, nil)
	if err != nil {
		return nil, fmt.Errorf("sending check list request: %w", err)
	}

	var result []synthetic_monitoring.Check

	if err := validateResponse("check list request", resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateTenant updates the specified tenant in the Synthetic Monitoring
// API. The updated tenant (possibly with updated timestamps) is
// returned.
func (h *Client) UpdateTenant(ctx context.Context, tenant synthetic_monitoring.Tenant) (*synthetic_monitoring.Tenant, error) {
	if err := h.requireAuthToken(); err != nil {
		return nil, err
	}

	resp, err := h.postJSON(ctx, "/tenant/update", true, &tenant)
	if err != nil {
		return nil, fmt.Errorf("sending tenant update request: %w", err)
	}

	var result synthetic_monitoring.Tenant

	if err := validateResponse("tenant update request", resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (h *Client) requireAuthToken() error {
	if h.accessToken == "" {
		return ErrAuthorizationTokenRequired
	}

	return nil
}

func (h *Client) do(ctx context.Context, url, method string, auth bool, headers http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating new HTTP request: %w", err)
	}

	if headers != nil {
		req.Header = headers
	}

	if auth {
		if req.Header == nil {
			req.Header = make(http.Header)
		}
		req.Header.Set("Authorization", "Bearer "+h.accessToken)
	}

	return h.client.Do(req)
}

func (h *Client) get(ctx context.Context, url string, auth bool, headers http.Header) (*http.Response, error) {
	return h.do(ctx, h.baseURL+url, http.MethodGet, auth, headers, nil)
}

func (h *Client) post(ctx context.Context, url string, auth bool, headers http.Header, body io.Reader) (*http.Response, error) {
	return h.do(ctx, h.baseURL+url, http.MethodPost, auth, headers, body)
}

func (h *Client) postJSON(ctx context.Context, url string, auth bool, req interface{}) (*http.Response, error) {
	var body bytes.Buffer

	var headers http.Header
	if req != nil {
		headers = defaultHeaders()

		if err := json.NewEncoder(&body).Encode(&req); err != nil {
			return nil, ErrCannotEncodeJSONRequest
		}
	}

	return h.post(ctx, url, auth, headers, &body)
}

func (h *Client) delete(ctx context.Context, url string, auth bool) (*http.Response, error) {
	return h.do(ctx, url, http.MethodDelete, auth, nil, nil)
}

// HTTPError represents errors returned from the Synthetic Monitoring API
// server.
//
// It implements the error interface, so it can be returned from functions
// interacting with the Synthetic Monitoring API server.
type HTTPError struct {
	Code   int
	Status string
	Action string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s: %s", e.Action, e.Status)
}

func defaultHeaders() http.Header {
	headers := make(http.Header)
	headers.Set("Content-type", "application/json; charset=utf-8")

	return headers
}

func validateResponse(action string, resp *http.Response, result interface{}) error {
	if resp.StatusCode != http.StatusOK {
		return &HTTPError{Code: resp.StatusCode, Status: resp.Status, Action: action}
	}

	if resp.Body != nil {
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)

		if err := dec.Decode(result); err != nil {
			return fmt.Errorf("%s, decoding response: %w", action, err)
		}
	}

	return nil
}

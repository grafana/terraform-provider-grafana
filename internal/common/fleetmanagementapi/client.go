package fleetmanagementapi

import (
	"encoding/base64"
	"net/http"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common/fleetmanagementapi/gen/proto/go/collector/v1/collectorv1connect"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/fleetmanagementapi/gen/proto/go/pipeline/v1/pipelinev1connect"
)

type Client struct {
	CollectorServiceClient collectorv1connect.CollectorServiceClient
	PipelineServiceClient  pipelinev1connect.PipelineServiceClient
}

func NewClient(client *http.Client, auth string, url string, headers map[string]string, userAgent string) *Client {
	httpClient := newHTTPClient(client, auth, headers, userAgent)

	collectorClient := collectorv1connect.NewCollectorServiceClient(httpClient, url)
	pipelineClient := pipelinev1connect.NewPipelineServiceClient(httpClient, url)

	return &Client{
		CollectorServiceClient: collectorClient,
		PipelineServiceClient:  pipelineClient,
	}
}

func newHTTPClient(client *http.Client, auth string, headers map[string]string, userAgent string) *http.Client {
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}

	return &http.Client{
		Transport: &transport{
			auth:          auth,
			headers:       headers,
			userAgent:     userAgent,
			baseTransport: client.Transport,
		},
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
		Timeout:       client.Timeout,
	}
}

type transport struct {
	auth          string
	headers       map[string]string
	userAgent     string
	baseTransport http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	encoded := base64.StdEncoding.EncodeToString([]byte(t.auth))
	clone.Header.Set("Authorization", "Basic "+encoded)

	for key, value := range t.headers {
		clone.Header.Set(key, value)
	}

	if t.userAgent != "" {
		clone.Header.Set("User-Agent", t.userAgent)
	}

	return t.baseTransport.RoundTrip(clone)
}

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

func NewClient(auth string, url string, client *http.Client, userAgent string, headers map[string]string) *Client {
	httpClient := newHTTPClient(client, auth, userAgent, headers)

	collectorClient := collectorv1connect.NewCollectorServiceClient(httpClient, url)
	pipelineClient := pipelinev1connect.NewPipelineServiceClient(httpClient, url)

	return &Client{
		CollectorServiceClient: collectorClient,
		PipelineServiceClient:  pipelineClient,
	}
}

func newHTTPClient(client *http.Client, auth string, userAgent string, headers map[string]string) *http.Client {
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}

	return &http.Client{
		Transport: &transport{
			auth:          auth,
			userAgent:     userAgent,
			headers:       headers,
			baseTransport: client.Transport,
		},
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
		Timeout:       client.Timeout,
	}
}

type transport struct {
	auth          string
	userAgent     string
	headers       map[string]string
	baseTransport http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	encoded := base64.StdEncoding.EncodeToString([]byte(t.auth))
	clone.Header.Set("Authorization", "Basic "+encoded)

	if t.userAgent != "" {
		clone.Header.Set("User-Agent", t.userAgent)
	}

	for key, value := range t.headers {
		clone.Header.Set(key, value)
	}

	return t.baseTransport.RoundTrip(clone)
}

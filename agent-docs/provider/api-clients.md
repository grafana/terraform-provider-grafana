# API Client Architecture

The `*common.Client` struct (`internal/common/client.go`) aggregates 13+ API clients. Each is conditionally created based on which credentials are configured.

## Client Registry

| Field | Go Type | Grafana Service | Auth Method | Required Fields |
|-------|---------|----------------|-------------|-----------------|
| `GrafanaAPI` | `*goapi.GrafanaHTTPAPI` | Core REST (`/api/*`) | Basic or Bearer | `url` + `auth` |
| `GrafanaAppPlatformAPI` | `*k8s.ClientRegistry` | App Platform (`/apis/*`) | Bearer or Basic | `url` + `auth` |
| `GrafanaCloudAPI` | `*gcom.APIClient` | grafana.com Cloud | Bearer | `cloud_access_policy_token` |
| `SMAPI` | `*SMAPI.Client` | Synthetic Monitoring | Bearer | `sm_access_token` |
| `MLAPI` | `*mlapi.Client` | Machine Learning | Bearer (Grafana) | `url` + `auth` |
| `OnCallClient` | `*onCallAPI.Client` | OnCall | Bearer (own or Grafana) | `oncall_access_token` OR `auth` |
| `SLOClient` | `*slo.APIClient` | SLO | Bearer (Grafana) | `url` + `auth` |
| `CloudProviderAPI` | `*cloudproviderapi.Client` | AWS/Azure integration | Bearer | `cloud_provider_access_token` |
| `ConnectionsAPIClient` | `*connectionsapi.Client` | Connections API | Bearer | `connections_api_access_token` |
| `FleetManagementClient` | `*fleetmanagementapi.Client` | Fleet Management | Basic (base64) | `fleet_management_auth` |
| `FrontendO11yAPIClient` | `*frontendo11yapi.Client` | Frontend Observability | Bearer stackID:token | `frontend_o11y_api_access_token` |
| `AssertsAPIClient` | `*assertsapi.APIClient` | Asserts (via plugin proxy) | Bearer (Grafana) | `url` + `auth` |
| `K6APIClient` | `*k6.APIClient` | k6 Cloud | Bearer | `k6_access_token` + `stack_id` |

Note: `MLAPI`, `SLOClient`, `AssertsAPIClient` are always co-created with `GrafanaAPI` (reuse same Grafana auth). `OnCallClient` falls back to Grafana `auth` token if `oncall_access_token` is not set.

## Client Creation Flow

```go
func CreateClients(providerConfig ProviderConfig) (*common.Client, error) {
    c := &common.Client{}

    if !providerConfig.Auth.IsNull() && !providerConfig.URL.IsNull() {
        createGrafanaAPIClient(c, cfg)         // GrafanaAPI
        createGrafanaAppPlatformClient(c, cfg) // GrafanaAppPlatformAPI
        createMLClient(c, cfg)                 // MLAPI
        createSLOClient(c, cfg)                // SLOClient
        createAssertsClient(c, cfg)            // AssertsAPIClient
    }
    if !providerConfig.CloudAccessPolicyToken.IsNull() {
        createCloudClient(c, cfg)              // GrafanaCloudAPI
    }
    if !providerConfig.SMAccessToken.IsNull() {
        createSMClient(c, cfg)                 // SMAPI
    }
    if !providerConfig.OncallAccessToken.IsNull() || ... {
        createOnCallClient(c, cfg)             // OnCallClient
    }
    // ... pattern repeats for each service
}
```

## Authentication Modes

### Main Grafana Client

`parseAuth(auth string)` in `configure_clients.go:519` determines mode from the `auth` field:

```
"admin:password"   → basic auth (username:password, contains ":")
"glsa_xxx..."      → bearer token (single string, no ":")
"anonymous"        → unauthenticated (literal string "anonymous")
```

### App Platform K8s Client

Uses the same URL and auth as `GrafanaAPI` but points to `/apis/...` endpoint:

```go
rest.Config{
    Host:        grafanaURL + "/apis",
    BearerToken: bearerToken,         // or
    Username/Password: basicAuthCreds,
    TLSClientConfig: tlsConfig,       // same TLS settings
}
```

The K8s client and REST OpenAPI client have completely separate transport stacks and Go libraries.

### Per-Service Tokens

Cloud, SM, OnCall, k6, CloudProvider, Connections, Fleet Management, FrontendO11y each accept their own access token. These are independent Bearer tokens — they do not inherit from the main `auth` field (exception: OnCall can fall back).

## TLS Configuration

`parseTLSconfig()` accepts either file paths or raw PEM content for each certificate field:

```
ca_cert            → system cert pool + custom CA
tls_cert + tls_key → mTLS client certificate
insecure_skip_verify → skip TLS verification (dev only)
```

If a field value contains `---BEGIN`, it's treated as a PEM literal (written to temp file). Otherwise treated as a file path.

The same `*tls.Config` is reused for both `GrafanaAPI` and `GrafanaAppPlatformAPI`. Other clients (SMAPI, MLAPI, etc.) use Go's default HTTP client and don't inherit TLS settings.

## Retry Configuration

**`GrafanaAPI`** — uses `TransportConfig.NumRetries` and `RetryStatusCodes` internally (via the OpenAPI client's built-in retry):
```
retries:          3 (default)
retry_status_codes: ["429", "500", "502", "503", "504"] (configurable)
retry_wait:        30s (default)
```

**All other clients** — use `hashicorp/go-retryablehttp`:
```go
retryClient := retryablehttp.NewClient()
retryClient.RetryMax = int(providerConfig.Retries.ValueInt64())
// Note: RetryStatusCodes not respected by retryablehttp — only 429+5xx automatic
```

## Custom HTTP Headers

All clients automatically include:
```
Grafana-Terraform-Provider: true
Grafana-Terraform-Provider-Version: <provider-version>
```

User-defined `http_headers` are merged on top. This applies to `GrafanaAPI` and `GrafanaAppPlatformAPI`. Other clients use their own transport and don't get these headers.

## Wrapper API Clients

Four services use hand-written REST clients in `internal/common/*api/` that follow the same pattern:

```go
type Client struct {
    authToken   string
    apiURL      string
    httpClient  *http.Client
    defaultHeaders map[string]string
}

func (c *Client) doAPIRequest(ctx, method, path, body) ([]byte, error) {
    req := http.NewRequestWithContext(...)
    for k, v := range c.defaultHeaders { req.Header.Set(k, v) }
    req.Header.Set("Authorization", "Bearer "+c.authToken)
    return c.httpClient.Do(req)
}
```

**Exception:** `FleetManagementClient` uses ConnectRPC (gRPC over HTTP/1.1 or HTTP/2) rather than plain REST, with a custom `http.RoundTripper` for header injection.

## GrafanaAppPlatformAPIClientID

```go
c.GrafanaAppPlatformAPIClientID = cfg.UserAgent.ValueString()
// e.g., "terraform-provider-grafana/v4.x.y"
```

This is injected into K8s resource metadata by `setManagerProperties()` in the appplatform resource:
```go
obj.SetCommonMetadata(resource.CommonMetadata{
    ManagedBy: "terraform",
    Identity:  c.GrafanaAppPlatformAPIClientID,
})
```

package provider

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/grafana-app-sdk/k8s"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/k6-cloud-openapi-client-go/k6"
	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/slo-openapi-client/go/slo"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/connectionsapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/fleetmanagementapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/frontendo11yapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/k6providerapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
)

func CreateClients(providerConfig ProviderConfig) (*common.Client, error) {
	var err error
	c := &common.Client{}
	if !providerConfig.Auth.IsNull() && !providerConfig.URL.IsNull() {
		if err = createGrafanaAPIClient(c, providerConfig); err != nil {
			return nil, err
		}
		if err = createGrafanaAppPlatformClient(c, providerConfig); err != nil {
			return nil, err
		}
		if err = createMLClient(c, providerConfig); err != nil {
			return nil, err
		}
		if err = createSLOClient(c, providerConfig); err != nil {
			return nil, err
		}
	}
	if !providerConfig.CloudAccessPolicyToken.IsNull() {
		if err := createCloudClient(c, providerConfig); err != nil {
			return nil, err
		}
	}
	if !providerConfig.SMAccessToken.IsNull() {
		c.SMAPI = SMAPI.NewClient(providerConfig.SMURL.ValueString(), providerConfig.SMAccessToken.ValueString(), getRetryClient(providerConfig))
	}
	if !providerConfig.OncallURL.IsNull() && (!providerConfig.OncallAccessToken.IsNull() || (!providerConfig.Auth.IsNull() && !providerConfig.URL.IsNull())) {
		var onCallClient *onCallAPI.Client
		onCallClient, err = createOnCallClient(providerConfig)
		if err != nil {
			return nil, err
		}
		onCallClient.UserAgent = providerConfig.UserAgent.ValueString()
		c.OnCallClient = onCallClient
	}
	if !providerConfig.CloudProviderAccessToken.IsNull() {
		if err := createCloudProviderClient(c, providerConfig); err != nil {
			return nil, err
		}
	}
	if !providerConfig.ConnectionsAPIAccessToken.IsNull() {
		if err := createConnectionsClient(c, providerConfig); err != nil {
			return nil, err
		}
	}
	if !providerConfig.FleetManagementAuth.IsNull() {
		if err := createFleetManagementClient(c, providerConfig); err != nil {
			return nil, err
		}
	}

	if !providerConfig.FrontendO11yAPIAccessToken.IsNull() || !providerConfig.CloudAccessPolicyToken.IsNull() {
		if err := createFrontendO11yClient(c, providerConfig); err != nil {
			return nil, err
		}
	}

	if !providerConfig.K6AccessToken.IsNull() && !providerConfig.StackID.IsNull() {
		if err := createK6Client(c, providerConfig); err != nil {
			return nil, err
		}
	}

	grafana.StoreDashboardSHA256 = providerConfig.StoreDashboardSha256.ValueBool()

	return c, nil
}

func createGrafanaAPIClient(client *common.Client, providerConfig ProviderConfig) error {
	tlsClientConfig, err := parseTLSconfig(providerConfig)
	if err != nil {
		return err
	}

	tlsConfig, err := tlsClientConfig.TLSConfig()
	if err != nil {
		return err
	}

	client.GrafanaAPIURL = providerConfig.URL.ValueString()
	client.GrafanaAPIURLParsed, err = url.Parse(providerConfig.URL.ValueString())
	if err != nil {
		return fmt.Errorf("failed to parse API url: %v", err.Error())
	}

	if client.GrafanaAPIURLParsed.Scheme == "http" && strings.Contains(client.GrafanaAPIURLParsed.Host, "grafana.net") {
		return fmt.Errorf("http not supported in Grafana Cloud. Use the https scheme")
	}

	apiPath, err := url.JoinPath(client.GrafanaAPIURLParsed.Path, "api")
	if err != nil {
		return fmt.Errorf("failed to join API path: %v", err.Error())
	}

	userInfo, orgID, apiKey, err := parseAuth(providerConfig)
	if err != nil {
		return err
	}

	if orgID > 1 && apiKey != "" {
		return fmt.Errorf("org_id is only supported with basic auth. API keys are already org-scoped")
	}

	cfg := goapi.TransportConfig{
		Host:             client.GrafanaAPIURLParsed.Host,
		BasePath:         apiPath,
		Schemes:          []string{client.GrafanaAPIURLParsed.Scheme},
		NumRetries:       int(providerConfig.Retries.ValueInt64()),
		RetryTimeout:     time.Second * time.Duration(providerConfig.RetryWait.ValueInt64()),
		RetryStatusCodes: setToStringArray(providerConfig.RetryStatusCodes.Elements()),
		TLSConfig:        tlsConfig,
		BasicAuth:        userInfo,
		OrgID:            orgID,
		APIKey:           apiKey,
	}

	if cfg.HTTPHeaders, err = getHTTPHeadersMap(providerConfig); err != nil {
		return err
	}
	client.GrafanaAPI = goapi.NewHTTPClientWithConfig(strfmt.Default, &cfg)
	client.GrafanaAPIConfig = &cfg

	return nil
}

func createGrafanaAppPlatformClient(client *common.Client, cfg ProviderConfig) error {
	rcfg := rest.Config{
		UserAgent: cfg.UserAgent.ValueString(),
		Host:      cfg.URL.ValueString(),
		APIPath:   "/apis",
	}

	tlsClientConfig, err := parseTLSconfig(cfg)
	if err != nil {
		return err
	}

	// Kubernetes really is wonderful, huh.
	// tl;dr it has it's own TLSClientConfig,
	// and it's not compatible with the one from the "crypto/tls" package.
	rcfg.TLSClientConfig = rest.TLSClientConfig{
		Insecure: tlsClientConfig.InsecureSkipVerify,
	}

	if len(tlsClientConfig.CertData) > 0 {
		rcfg.CertData = tlsClientConfig.CertData
	}

	if len(tlsClientConfig.KeyData) > 0 {
		rcfg.KeyData = tlsClientConfig.KeyData
	}

	if len(tlsClientConfig.CAData) > 0 {
		rcfg.CAData = tlsClientConfig.CAData
	}

	userInfo, orgID, apiKey, err := parseAuth(cfg)
	if err != nil {
		return err
	}

	switch {
	case apiKey != "":
		if orgID > 1 {
			return fmt.Errorf("org_id is only supported with basic auth. API keys are already org-scoped")
		}

		rcfg.BearerToken = apiKey
	case userInfo != nil:
		rcfg.Username = userInfo.Username()
		if p, ok := userInfo.Password(); ok {
			rcfg.Password = p
		}
	}

	client.GrafanaOrgID = cfg.OrgID.ValueInt64()
	client.GrafanaStackID = cfg.StackID.ValueInt64()
	client.GrafanaAppPlatformAPIClientID = cfg.UserAgent.ValueString()
	client.GrafanaAppPlatformAPI = k8s.NewClientRegistry(rcfg, k8s.ClientConfig{
		NegotiatedSerializerProvider: func(kind resource.Kind) runtime.NegotiatedSerializer {
			return &k8s.KindNegotiatedSerializer{
				Kind: kind,
			}
		},
	})

	return nil
}

func createMLClient(client *common.Client, providerConfig ProviderConfig) error {
	mlcfg := mlapi.Config{
		BasicAuth:   client.GrafanaAPIConfig.BasicAuth,
		BearerToken: client.GrafanaAPIConfig.APIKey,
		Client:      getRetryClient(providerConfig),
	}
	mlURL := client.GrafanaAPIURL
	if !strings.HasSuffix(mlURL, "/") {
		mlURL += "/"
	}
	mlURL += "api/plugins/grafana-ml-app/resources"
	var err error
	client.MLAPI, err = mlapi.New(mlURL, mlcfg)
	return err
}

func createSLOClient(client *common.Client, providerConfig ProviderConfig) error {
	var err error

	sloConfig := slo.NewConfiguration()
	sloConfig.Host = client.GrafanaAPIURLParsed.Host
	sloConfig.Scheme = client.GrafanaAPIURLParsed.Scheme
	sloConfig.DefaultHeader, err = getHTTPHeadersMap(providerConfig)
	sloConfig.DefaultHeader["Authorization"] = "Bearer " + providerConfig.Auth.ValueString()
	sloConfig.HTTPClient = getRetryClient(providerConfig)
	client.SLOClient = slo.NewAPIClient(sloConfig)

	return err
}

func createCloudClient(client *common.Client, providerConfig ProviderConfig) error {
	openAPIConfig := gcom.NewConfiguration()
	parsedURL, err := url.Parse(providerConfig.CloudAPIURL.ValueString())
	if err != nil {
		return err
	}
	openAPIConfig.Host = parsedURL.Host
	openAPIConfig.Scheme = parsedURL.Scheme
	openAPIConfig.HTTPClient = getRetryClient(providerConfig)
	openAPIConfig.DefaultHeader["Authorization"] = "Bearer " + providerConfig.CloudAccessPolicyToken.ValueString()
	httpHeaders, err := getHTTPHeadersMap(providerConfig)
	if err != nil {
		return err
	}
	for k, v := range httpHeaders {
		openAPIConfig.DefaultHeader[k] = v
	}
	client.GrafanaCloudAPI = gcom.NewAPIClient(openAPIConfig)

	return nil
}

func createOnCallClient(providerConfig ProviderConfig) (*onCallAPI.Client, error) {
	authToken := providerConfig.OncallAccessToken.ValueString()
	if authToken == "" {
		// prefer OncallAccessToken if it was set, otherwise use Grafana auth (service account) token
		authToken = providerConfig.Auth.ValueString()
	}
	return onCallAPI.NewWithGrafanaURL(providerConfig.OncallURL.ValueString(), authToken, providerConfig.URL.ValueString())
}

func createCloudProviderClient(client *common.Client, providerConfig ProviderConfig) error {
	providerHeaders, err := getHTTPHeadersMap(providerConfig)
	if err != nil {
		return fmt.Errorf("failed to get provider default HTTP headers: %w", err)
	}

	apiClient, err := cloudproviderapi.NewClient(
		providerConfig.CloudProviderAccessToken.ValueString(),
		providerConfig.CloudProviderURL.ValueString(),
		getRetryClient(providerConfig),
		providerHeaders,
	)
	if err != nil {
		return err
	}
	client.CloudProviderAPI = apiClient
	return nil
}

func createFrontendO11yClient(client *common.Client, providerConfig ProviderConfig) error {
	providerHeaders, err := getHTTPHeadersMap(providerConfig)
	if err != nil {
		return fmt.Errorf("failed to get provider default HTTP headers: %w", err)
	}

	var token string
	if providerConfig.FrontendO11yAPIAccessToken.IsNull() {
		token = providerConfig.CloudAccessPolicyToken.ValueString()
	} else {
		token = providerConfig.FrontendO11yAPIAccessToken.ValueString()
	}

	cURL, err := url.Parse(providerConfig.CloudAPIURL.ValueString())
	if err != nil {
		return err
	}

	cHostParts := strings.Split(cURL.Host, ".")
	if len(cHostParts) < 2 {
		return fmt.Errorf("invalid cloud url")
	}
	// https://grafana.com -> grafana.net
	cHost := fmt.Sprintf("%s.net", cHostParts[len(cHostParts)-2])

	apiClient, err := frontendo11yapi.NewClient(
		cHost,
		token,
		getRetryClient(providerConfig),
		providerConfig.UserAgent.ValueString(),
		providerHeaders,
	)
	if err != nil {
		return err
	}
	client.FrontendO11yAPIClient = apiClient
	return nil
}

func createConnectionsClient(client *common.Client, providerConfig ProviderConfig) error {
	providerHeaders, err := getHTTPHeadersMap(providerConfig)
	if err != nil {
		return fmt.Errorf("failed to get provider default HTTP headers: %w", err)
	}

	apiClient, err := connectionsapi.NewClient(
		providerConfig.ConnectionsAPIAccessToken.ValueString(),
		providerConfig.ConnectionsAPIURL.ValueString(),
		getRetryClient(providerConfig),
		providerConfig.UserAgent.ValueString(),
		providerHeaders,
	)
	if err != nil {
		return err
	}
	client.ConnectionsAPIClient = apiClient
	return nil
}

func createK6Client(client *common.Client, providerConfig ProviderConfig) error {
	k6Cfg := k6.NewConfiguration()
	if !providerConfig.K6URL.IsNull() {
		k6Cfg.Servers = []k6.ServerConfiguration{
			{URL: providerConfig.K6URL.ValueString()},
		}
	}

	k6Cfg.HTTPClient = getRetryClient(providerConfig)

	httpHeaders, err := getHTTPHeadersMap(providerConfig)
	if err != nil {
		return err
	}
	for k, v := range httpHeaders {
		k6Cfg.DefaultHeader[k] = v
	}

	client.K6APIClient = k6.NewAPIClient(k6Cfg)
	client.K6APIConfig = &k6providerapi.K6APIConfig{
		Token:   providerConfig.K6AccessToken.ValueString(),
		StackID: int32(providerConfig.StackID.ValueInt64()),
	}
	return nil
}

func createFleetManagementClient(client *common.Client, providerConfig ProviderConfig) error {
	providerHeaders, err := getHTTPHeadersMap(providerConfig)
	if err != nil {
		return fmt.Errorf("failed to get provider default HTTP headers: %w", err)
	}

	client.FleetManagementClient = fleetmanagementapi.NewClient(
		providerConfig.FleetManagementAuth.ValueString(),
		providerConfig.FleetManagementURL.ValueString(),
		getRetryClient(providerConfig),
		providerConfig.UserAgent.ValueString(),
		providerHeaders,
	)

	return nil
}

// Sets a custom HTTP Header on all requests coming from the Grafana Terraform Provider to Grafana-Terraform-Provider: true
// in addition to any headers set within the `http_headers` field or the `GRAFANA_HTTP_HEADERS` environment variable
func getHTTPHeadersMap(providerConfig ProviderConfig) (map[string]string, error) {
	headers := map[string]string{
		"Grafana-Terraform-Provider":         "true",
		"Grafana-Terraform-Provider-Version": providerConfig.Version.ValueString(),
	}
	for k, v := range providerConfig.HTTPHeaders.Elements() {
		if vString, ok := v.(types.String); ok {
			headers[k] = vString.ValueString()
		} else {
			return nil, fmt.Errorf("invalid header value for %s: %v", k, v)
		}
	}

	return headers, nil
}

func createTempFileIfLiteral(value string) (path string, tempFile bool, err error) {
	if value == "" {
		return "", false, nil
	}

	if _, err := os.Stat(value); errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENAMETOOLONG) {
		// value is not a file path, assume it's a literal
		f, err := os.CreateTemp("", "grafana-provider-tls")
		if err != nil {
			return "", false, err
		}
		if _, err := f.WriteString(value); err != nil {
			return "", false, err
		}
		if err := f.Close(); err != nil {
			return "", false, err
		}
		return f.Name(), true, nil
	}

	return value, false, nil
}

func parseAuth(providerConfig ProviderConfig) (*url.Userinfo, int64, string, error) {
	auth := strings.SplitN(providerConfig.Auth.ValueString(), ":", 2)
	var orgID int64 = 1

	if len(auth) == 2 {
		user := strings.TrimSpace(auth[0])
		pass := strings.TrimSpace(auth[1])
		return url.UserPassword(user, pass), orgID, "", nil
	} else if auth[0] != "anonymous" {
		apiKey := strings.TrimSpace(auth[0])
		return nil, 0, apiKey, nil
	}
	return nil, 0, "", nil
}

type TLSConfig struct {
	CAData             []byte
	CertData           []byte
	KeyData            []byte
	InsecureSkipVerify bool
}

func (t TLSConfig) TLSConfig() (*tls.Config, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to get system certificate pool: %w", err)
	}

	res := &tls.Config{
		// gosec is stating the obvious here.
		// G402: TLS InsecureSkipVerify may be true. (gosec)
		// nolint: gosec
		InsecureSkipVerify: t.InsecureSkipVerify,
		RootCAs:            pool,
	}

	if len(t.CAData) > 0 {
		if !res.RootCAs.AppendCertsFromPEM(t.CAData) {
			return nil, fmt.Errorf("failed to append CA data")
		}
	}

	if len(t.CertData) > 0 && len(t.KeyData) > 0 {
		cert, err := tls.X509KeyPair(t.CertData, t.KeyData)
		if err != nil {
			return nil, err
		}

		res.Certificates = []tls.Certificate{cert}
	}

	return res, nil
}

func parseTLSconfig(providerConfig ProviderConfig) (*TLSConfig, error) {
	res := &TLSConfig{
		InsecureSkipVerify: providerConfig.InsecureSkipVerify.ValueBool(),
	}

	tlsKeyFile, tempFile, err := createTempFileIfLiteral(providerConfig.TLSKey.ValueString())
	if err != nil {
		return nil, err
	}
	if tempFile {
		defer os.Remove(tlsKeyFile)
	}
	tlsCertFile, tempFile, err := createTempFileIfLiteral(providerConfig.TLSCert.ValueString())
	if err != nil {
		return nil, err
	}
	if tempFile {
		defer os.Remove(tlsCertFile)
	}
	caCertFile, tempFile, err := createTempFileIfLiteral(providerConfig.CACert.ValueString())
	if err != nil {
		return nil, err
	}
	if tempFile {
		defer os.Remove(caCertFile)
	}

	if caCertFile != "" {
		ca, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		res.CAData = ca
	}

	if tlsCertFile != "" {
		certData, err := os.ReadFile(tlsCertFile)
		if err != nil {
			return nil, err
		}
		res.CertData = certData
	}

	if tlsKeyFile != "" {
		keyData, err := os.ReadFile(tlsKeyFile)
		if err != nil {
			return nil, err
		}
		res.KeyData = keyData
	}

	return res, nil
}

func setToStringArray(set []attr.Value) []string {
	var result []string
	for _, v := range set {
		result = append(result, v.(types.String).ValueString())
	}
	return result
}

func getRetryClient(providerConfig ProviderConfig) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = int(providerConfig.Retries.ValueInt64())
	if wait := providerConfig.RetryWait.ValueInt64(); wait > 0 {
		retryClient.RetryWaitMin = time.Second * time.Duration(wait)
		retryClient.RetryWaitMax = time.Second * time.Duration(wait)
	}
	return retryClient.StandardClient()
}

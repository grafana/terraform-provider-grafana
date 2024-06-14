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
	"time"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/machine-learning-go-client/mlapi"
	slo "github.com/grafana/slo-openapi-client/go"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"

	"github.com/go-openapi/strfmt"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func CreateClients(providerConfig ProviderConfig) (*common.Client, error) {
	var err error
	c := &common.Client{}
	if !providerConfig.Auth.IsNull() && !providerConfig.URL.IsNull() {
		if err = createGrafanaAPIClient(c, providerConfig); err != nil {
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
	if !providerConfig.OncallAccessToken.IsNull() {
		var onCallClient *onCallAPI.Client
		onCallClient, err = createOnCallClient(providerConfig)
		if err != nil {
			return nil, err
		}
		onCallClient.UserAgent = providerConfig.UserAgent.ValueString()
		c.OnCallClient = onCallClient
	}
	if !providerConfig.CloudProviderURL.IsNull() && !providerConfig.CloudProviderAccessToken.IsNull() {
		if err := createCloudProviderClient(c, providerConfig); err != nil {
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

	client.GrafanaAPIURL = providerConfig.URL.ValueString()
	client.GrafanaAPIURLParsed, err = url.Parse(providerConfig.URL.ValueString())
	if err != nil {
		return fmt.Errorf("failed to parse API url: %v", err.Error())
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
		TLSConfig:        tlsClientConfig,
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

func createMLClient(client *common.Client, providerConfig ProviderConfig) error {
	mlcfg := mlapi.Config{
		BasicAuth:   client.GrafanaAPIConfig.BasicAuth,
		BearerToken: client.GrafanaAPIConfig.APIKey,
		Client:      getRetryClient(providerConfig),
		NumRetries:  client.GrafanaAPIConfig.NumRetries,
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
	sloConfig := slo.NewConfiguration()
	sloConfig.Host = client.GrafanaAPIURLParsed.Host
	sloConfig.Scheme = client.GrafanaAPIURLParsed.Scheme
	sloConfig.DefaultHeader["Authorization"] = "Bearer " + providerConfig.Auth.ValueString()
	sloConfig.DefaultHeader["Grafana-Terraform-Provider"] = "true"
	sloConfig.HTTPClient = getRetryClient(providerConfig)
	client.SLOClient = slo.NewAPIClient(sloConfig)
	return nil
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
	return onCallAPI.New(providerConfig.OncallURL.ValueString(), providerConfig.OncallAccessToken.ValueString())
}

func createCloudProviderClient(client *common.Client, providerConfig ProviderConfig) error {
	apiClient, err := cloudproviderapi.NewClient(
		providerConfig.CloudProviderAccessToken.ValueString(),
		providerConfig.CloudProviderURL.ValueString(),
	)
	if err != nil {
		return err
	}
	client.CloudProviderAPI = apiClient
	return nil
}

// Sets a custom HTTP Header on all requests coming from the Grafana Terraform Provider to Grafana-Terraform-Provider: true
// in addition to any headers set within the `http_headers` field or the `GRAFANA_HTTP_HEADERS` environment variable
func getHTTPHeadersMap(providerConfig ProviderConfig) (map[string]string, error) {
	headers := map[string]string{"Grafana-Terraform-Provider": "true"}
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

	if _, err := os.Stat(value); errors.Is(err, os.ErrNotExist) {
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
		return url.UserPassword(auth[0], auth[1]), orgID, "", nil
	} else if auth[0] != "anonymous" {
		return nil, 0, auth[0], nil
	}
	return nil, 0, "", nil
}

func parseTLSconfig(providerConfig ProviderConfig) (*tls.Config, error) {
	tlsClientConfig := &tls.Config{}

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

	insecure := providerConfig.InsecureSkipVerify.ValueBool()
	if caCertFile != "" {
		ca, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		tlsClientConfig.RootCAs = pool
	}
	if tlsKeyFile != "" && tlsCertFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
		if err != nil {
			return nil, err
		}
		tlsClientConfig.Certificates = []tls.Certificate{cert}
	}
	if insecure {
		tlsClientConfig.InsecureSkipVerify = true
	}

	return tlsClientConfig, nil
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

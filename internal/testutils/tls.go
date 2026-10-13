package testutils

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"syscall"
)

// GrafanaTLSTransport returns an HTTP transport honouring the TLS settings of the
// acceptance test environment (GRAFANA_CA_CERT, GRAFANA_TLS_CERT, GRAFANA_TLS_KEY,
// GRAFANA_INSECURE_SKIP_VERIFY). Tests that call Grafana without going through the
// provider's API client need it: that client configures TLS on its own transport only.
func GrafanaTLSTransport() (*http.Transport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	caCert, err := readPEMEnv("GRAFANA_CA_CERT")
	if err != nil {
		return nil, err
	}
	clientCert, err := readPEMEnv("GRAFANA_TLS_CERT")
	if err != nil {
		return nil, err
	}
	clientKey, err := readPEMEnv("GRAFANA_TLS_KEY")
	if err != nil {
		return nil, err
	}
	insecure, err := boolEnv("GRAFANA_INSECURE_SKIP_VERIFY")
	if err != nil {
		return nil, err
	}
	if caCert == nil && clientCert == nil && clientKey == nil && !insecure {
		return transport, nil
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to get system certificate pool: %w", err)
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
		// The provider supports skipping verification, so tests talking to the same
		// instance must too.
		InsecureSkipVerify: insecure, //nolint:gosec // G402: mirrors the provider's insecure_skip_verify.
	}

	if caCert != nil && !tlsConfig.RootCAs.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append GRAFANA_CA_CERT to the certificate pool")
	}
	if clientCert != nil && clientKey != nil {
		cert, err := tls.X509KeyPair(clientCert, clientKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport.TLSClientConfig = tlsConfig
	return transport, nil
}

// readPEMEnv reads the PEM data an env var points at, or returns the value itself
// when it is not a file path. Mirrors how the provider accepts both forms.
func readPEMEnv(name string) ([]byte, error) {
	value := os.Getenv(name)
	if value == "" {
		return nil, nil
	}
	//nolint:gosec // G703: the path comes from a trusted test env var, not user input.
	if _, err := os.Stat(value); errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENAMETOOLONG) {
		// Not a path we can read: assume the value is the PEM data itself.
		return []byte(value), nil
	}
	//nolint:gosec // G703: the path comes from a trusted test env var, not user input.
	data, err := os.ReadFile(value)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", name, err)
	}
	return data, nil
}

// boolEnv parses a boolean env var, treating an unset var as false.
func boolEnv(name string) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s: %w", name, err)
	}
	return parsed, nil
}

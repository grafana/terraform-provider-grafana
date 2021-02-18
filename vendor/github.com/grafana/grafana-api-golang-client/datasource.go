package gapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DataSource represents a Grafana data source.
type DataSource struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Access string `json:"access"`

	Database string `json:"database,omitempty"`
	User     string `json:"user,omitempty"`
	// Deprecated: Use secureJsonData.password instead.
	Password string `json:"password,omitempty"`

	OrgID     int64 `json:"orgId,omitempty"`
	IsDefault bool  `json:"isDefault"`

	BasicAuth     bool   `json:"basicAuth"`
	BasicAuthUser string `json:"basicAuthUser,omitempty"`
	// Deprecated: Use secureJsonData.basicAuthPassword instead.
	BasicAuthPassword string `json:"basicAuthPassword,omitempty"`

	JSONData       JSONData       `json:"jsonData,omitempty"`
	SecureJSONData SecureJSONData `json:"secureJsonData,omitempty"`
}

// JSONData is a representation of the datasource `jsonData` property
type JSONData struct {
	// Used by all datasources
	TLSAuth           bool `json:"tlsAuth,omitempty"`
	TLSAuthWithCACert bool `json:"tlsAuthWithCACert,omitempty"`
	TLSSkipVerify     bool `json:"tlsSkipVerify,omitempty"`

	// Used by Graphite
	GraphiteVersion string `json:"graphiteVersion,omitempty"`

	// Used by Prometheus, Elasticsearch, InfluxDB, MySQL, PostgreSQL and MSSQL
	TimeInterval string `json:"timeInterval,omitempty"`

	// Used by Elasticsearch
	EsVersion       int64  `json:"esVersion,omitempty"`
	TimeField       string `json:"timeField,omitempty"`
	Interval        string `json:"inteval,omitempty"`
	LogMessageField string `json:"logMessageField,omitempty"`
	LogLevelField   string `json:"logLevelField,omitempty"`

	// Used by Cloudwatch
	AuthType                string `json:"authType,omitempty"`
	AssumeRoleArn           string `json:"assumeRoleArn,omitempty"`
	DefaultRegion           string `json:"defaultRegion,omitempty"`
	CustomMetricsNamespaces string `json:"customMetricsNamespaces,omitempty"`
	Profile                 string `json:"profile,omitempty"`

	// Used by OpenTSDB
	TsdbVersion    string `json:"tsdbVersion,omitempty"`
	TsdbResolution string `json:"tsdbResolution,omitempty"`

	// Used by MSSQL
	Encrypt string `json:"encrypt,omitempty"`

	// Used by PostgreSQL
	Sslmode         string `json:"sslmode,omitempty"`
	PostgresVersion int64  `json:"postgresVersion,omitempty"`
	Timescaledb     bool   `json:"timescaledb,omitempty"`

	// Used by MySQL, PostgreSQL and MSSQL
	MaxOpenConns    int64 `json:"maxOpenConns,omitempty"`
	MaxIdleConns    int64 `json:"maxIdleConns,omitempty"`
	ConnMaxLifetime int64 `json:"connMaxLifetime,omitempty"`

	// Used by Prometheus
	HTTPMethod   string `json:"httpMethod,omitempty"`
	QueryTimeout string `json:"queryTimeout,omitempty"`

	// Used by Stackdriver
	AuthenticationType string `json:"authenticationType,omitempty"`
	ClientEmail        string `json:"clientEmail,omitempty"`
	DefaultProject     string `json:"defaultProject,omitempty"`
	TokenURI           string `json:"tokenUri,omitempty"`
}

// SecureJSONData is a representation of the datasource `secureJsonData` property
type SecureJSONData struct {
	// Used by all datasources
	TLSCACert         string `json:"tlsCACert,omitempty"`
	TLSClientCert     string `json:"tlsClientCert,omitempty"`
	TLSClientKey      string `json:"tlsClientKey,omitempty"`
	Password          string `json:"password,omitempty"`
	BasicAuthPassword string `json:"basicAuthPassword,omitempty"`

	// Used by Cloudwatch
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`

	// Used by Stackdriver
	PrivateKey string `json:"privateKey,omitempty"`
}

// NewDataSource creates a new Grafana data source.
func (c *Client) NewDataSource(s *DataSource) (int64, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return 0, err
	}

	result := struct {
		ID int64 `json:"id"`
	}{}

	err = c.request("POST", "/api/datasources", nil, bytes.NewBuffer(data), &result)
	if err != nil {
		return 0, err
	}

	return result.ID, err
}

// UpdateDataSource updates a Grafana data source.
func (c *Client) UpdateDataSource(s *DataSource) error {
	path := fmt.Sprintf("/api/datasources/%d", s.ID)
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return c.request("PUT", path, nil, bytes.NewBuffer(data), nil)
}

// DataSource fetches and returns the Grafana data source whose ID it's passed.
func (c *Client) DataSource(id int64) (*DataSource, error) {
	path := fmt.Sprintf("/api/datasources/%d", id)
	result := &DataSource{}
	err := c.request("GET", path, nil, nil, result)
	if err != nil {
		return nil, err
	}

	return result, err
}

// DeleteDataSource deletes the Grafana data source whose ID it's passed.
func (c *Client) DeleteDataSource(id int64) error {
	path := fmt.Sprintf("/api/datasources/%d", id)

	return c.request("DELETE", path, nil, nil, nil)
}

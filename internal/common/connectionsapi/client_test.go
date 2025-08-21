package connectionsapi_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/connectionsapi"
)

func Test_NewClient(t *testing.T) {
	defaultHeaders := map[string]string{}

	t.Run("successfully creates a new client", func(t *testing.T) {
		client, err := connectionsapi.NewClient("some token", "https://valid.url", &http.Client{}, "some-user-agent", defaultHeaders)

		assert.NotNil(t, client)
		assert.NoError(t, err)
	})

	t.Run("returns error for invalid url", func(t *testing.T) {
		_, err := connectionsapi.NewClient("some token", " https://leading.space", &http.Client{}, "some-user-agent", defaultHeaders)

		assert.Error(t, err)
		assert.Equal(t, `failed to parse connections API url: parse " https://leading.space": first path segment in URL cannot contain colon`, err.Error())
	})
}

func TestClient_CreateMetricsEndpointScrapeJob(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))
			assert.Equal(t, "/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/test_job", r.URL.Path)
			requestBody, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `
			{
				"enabled":true,
				"authentication_method":"basic",
				"basic_password":"my-password",
				"basic_username":"my-username",
				"url":"https://my-example-url.com:9000/metrics",
				"scrape_interval_seconds":120
			}`, string(requestBody))

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`
			{
				"status":"success",
				"data":{
					"enabled":true,
					"authentication_method":"basic",
					"basic_username":"my-username",
					"basic_password":"my-password",
					"url":"https://my-example-url.com:9000/metrics",
					"scrape_interval_seconds":120,
					"flavor":"default"
				}
			}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		actualJob, err := c.CreateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "test_job", connectionsapi.MetricsEndpointScrapeJob{
			Enabled:                     true,
			AuthenticationMethod:        "basic",
			AuthenticationBasicUsername: "my-username",
			AuthenticationBasicPassword: "my-password",
			URL:                         "https://my-example-url.com:9000/metrics",
			ScrapeIntervalSeconds:       120,
		})
		assert.NoError(t, err)

		assert.Equal(t, connectionsapi.MetricsEndpointScrapeJob{
			Enabled:                     true,
			AuthenticationMethod:        "basic",
			AuthenticationBasicUsername: "my-username",
			AuthenticationBasicPassword: "my-password",
			URL:                         "https://my-example-url.com:9000/metrics",
			ScrapeIntervalSeconds:       120,
		}, actualJob)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.CreateMetricsEndpointScrapeJob(context.Background(), "some stack id", "test_job", connectionsapi.MetricsEndpointScrapeJob{})
		require.NoError(t, err)
	})

	t.Run("returns error when connections API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.CreateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "test_job", connectionsapi.MetricsEndpointScrapeJob{})

		assert.Error(t, err)
		assert.Equal(t, `failed to create metrics endpoint scrape job "test_job": status: 500`, err.Error())
	})

	t.Run("returns ErrUnauthorized when connections API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.CreateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "test_job", connectionsapi.MetricsEndpointScrapeJob{})

		assert.Error(t, err)
		assert.Equal(t, `failed to create metrics endpoint scrape job "test_job": request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, connectionsapi.ErrUnauthorized))
	})

	t.Run("returns error when client fails to Do request", func(t *testing.T) {
		c, err := connectionsapi.NewClient("some token", "some random url", &http.Client{}, "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.CreateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name", connectionsapi.MetricsEndpointScrapeJob{})

		assert.Error(t, err)
		assert.Equal(t, `failed to create metrics endpoint scrape job "job-name": failed to do request: Post "some%20random%20url/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/job-name": unsupported protocol scheme ""`, err.Error())
	})
}

func TestClient_GetMetricsEndpointScrapeJob(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))
			assert.Equal(t, "/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/test_job", r.URL.Path)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
			{
				"status":"success",
				"data":{
					"name":"test_job",
					"enabled":true,
					"authentication_method":"basic",
					"basic_username":"my-username",
					"basic_password":"my-password",
					"url":"https://my-example-url.com:9000/metrics",
					"scrape_interval_seconds":120,
					"flavor":"default"
				}
			}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		actualJob, err := c.GetMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "test_job")
		assert.NoError(t, err)

		assert.Equal(t, connectionsapi.MetricsEndpointScrapeJob{
			Enabled:                     true,
			AuthenticationMethod:        "basic",
			AuthenticationBasicUsername: "my-username",
			AuthenticationBasicPassword: "my-password",
			URL:                         "https://my-example-url.com:9000/metrics",
			ScrapeIntervalSeconds:       120,
		}, actualJob)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetMetricsEndpointScrapeJob(context.Background(), "some stack id", "test_job")
		require.NoError(t, err)
	})

	t.Run("returns ErrorNotFound when connections API responds 404", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.ErrorIs(t, err, connectionsapi.ErrNotFound)
		assert.Equal(t, `failed to get metrics endpoint scrape job "job-name": not found`, err.Error())
	})

	t.Run("returns ErrUnauthorized when connections API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.Equal(t, `failed to get metrics endpoint scrape job "job-name": request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, connectionsapi.ErrUnauthorized))
	})

	t.Run("returns error when connections API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.Equal(t, `failed to get metrics endpoint scrape job "job-name": status: 500`, err.Error())
	})

	t.Run("returns error when client fails to Do request", func(t *testing.T) {
		c, err := connectionsapi.NewClient("some token", "some random url", &http.Client{}, "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.Equal(t, `failed to get metrics endpoint scrape job "job-name": failed to do request: Get "some%20random%20url/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/job-name": unsupported protocol scheme ""`, err.Error())
	})
}

func TestClient_UpdateMetricsEndpointScrapeJob(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))
			assert.Equal(t, "/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/test_job", r.URL.Path)
			requestBody, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `
			{
				"enabled":true,
				"authentication_method":"bearer",
				"bearer_token":"some token",
				"url":"https://updated-url.com:9000/metrics",
				"scrape_interval_seconds":120
			}`, string(requestBody))

			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`
			{
				"status":"success",
				"data":{
					"enabled":true,
					"authentication_method":"bearer",
					"bearer_token":"some token",
					"url":"https://updated-url.com:9000/metrics",
					"scrape_interval_seconds":120,
					"flavor":"default"
				}
			}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		actualJob, err := c.UpdateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "test_job",
			connectionsapi.MetricsEndpointScrapeJob{
				Enabled:                   true,
				AuthenticationMethod:      "bearer",
				AuthenticationBearerToken: "some token",
				URL:                       "https://updated-url.com:9000/metrics",
				ScrapeIntervalSeconds:     120,
			})
		assert.NoError(t, err)

		assert.Equal(t, connectionsapi.MetricsEndpointScrapeJob{
			Enabled:                   true,
			AuthenticationMethod:      "bearer",
			AuthenticationBearerToken: "some token",
			URL:                       "https://updated-url.com:9000/metrics",
			ScrapeIntervalSeconds:     120,
		}, actualJob)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateMetricsEndpointScrapeJob(context.Background(), "some stack id", "test_job", connectionsapi.MetricsEndpointScrapeJob{})
		require.NoError(t, err)
	})

	t.Run("returns ErrorNotFound when connections API responds 404", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name", connectionsapi.MetricsEndpointScrapeJob{})

		assert.Error(t, err)
		assert.ErrorIs(t, err, connectionsapi.ErrNotFound)
		assert.Equal(t, `failed to update metrics endpoint scrape job "job-name": not found`, err.Error())
	})

	t.Run("returns Unauthorized when connections API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name", connectionsapi.MetricsEndpointScrapeJob{})

		assert.Error(t, err)
		assert.Equal(t, `failed to update metrics endpoint scrape job "job-name": request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, connectionsapi.ErrUnauthorized))
	})

	t.Run("returns error when connections API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name", connectionsapi.MetricsEndpointScrapeJob{})

		assert.Error(t, err)
		assert.Equal(t, `failed to update metrics endpoint scrape job "job-name": status: 500`, err.Error())
	})

	t.Run("returns error when client fails to Do request", func(t *testing.T) {
		c, err := connectionsapi.NewClient("some token", "some random url", &http.Client{}, "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name", connectionsapi.MetricsEndpointScrapeJob{})

		assert.Error(t, err)
		assert.Equal(t, `failed to update metrics endpoint scrape job "job-name": failed to do request: Put "some%20random%20url/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/job-name": unsupported protocol scheme ""`, err.Error())
	})
}

func TestClient_DeleteMetricsEndpointScrapeJob(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/test_job", r.URL.Path)

			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))

			w.WriteHeader(http.StatusOK)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "test_job")

		assert.NoError(t, err)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteMetricsEndpointScrapeJob(context.Background(), "some stack id", "test_job")
		require.NoError(t, err)
	})

	t.Run("returns ErrorNotFound when connections API responds 404", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.ErrorIs(t, err, connectionsapi.ErrNotFound)
		assert.Equal(t, `failed to delete metrics endpoint scrape job "job-name": not found`, err.Error())
	})

	t.Run("returns ErrUnauthorized when connections API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.Equal(t, `failed to delete metrics endpoint scrape job "job-name": request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, connectionsapi.ErrUnauthorized))
	})

	t.Run("returns error when connections API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := connectionsapi.NewClient("some token", svr.URL, svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.Equal(t, `failed to delete metrics endpoint scrape job "job-name": status: 500`, err.Error())
	})

	t.Run("returns error when client fails to Do request", func(t *testing.T) {
		c, err := connectionsapi.NewClient("some token", "some random url", &http.Client{}, "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteMetricsEndpointScrapeJob(context.Background(), "some-stack-id", "job-name")

		assert.Error(t, err)
		assert.Equal(t, `failed to delete metrics endpoint scrape job "job-name": failed to do request: Delete "some%20random%20url/api/v1/stacks/some-stack-id/metrics-endpoint/jobs/job-name": unsupported protocol scheme ""`, err.Error())
	})
}

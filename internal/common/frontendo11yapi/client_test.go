package frontendo11yapi_test

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

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
)

func Test_NewClient(t *testing.T) {
	defaultHeaders := map[string]string{}

	t.Run("successfully creates a new client", func(t *testing.T) {
		client, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", &http.Client{}, "some-user-agent", defaultHeaders)

		assert.NotNil(t, client)
		assert.NoError(t, err)
	})
}

func TestClient_CreateApp(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))
			assert.Equal(t, "/api/v1/app", r.URL.Path)
			requestBody, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `
			{
				"name": "Test App",
    			"corsOrigins": [{
    			    "url": "*"
    			}],
				"settings": {
					"geolcation.enabled": "1"
				}
			}`, string(requestBody))

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`
			{
				"id": 1,
				"name": "Test App",
    			"appKey": "foobar",
    			"corsOrigins": [{
    			    "id": 1,
    			    "url": "*"
    			}],
				"settings": {
					"geolcation.enabled": "1"
				},
				"allowedRate": 0
			}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		acutalApp, err := c.CreateApp(context.Background(), svr.URL, 1, frontendo11yapi.App{
			Name: "Test App",
			CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
				{
					URL: "*",
				},
			},
			Settings: map[string]string{
				"geolcation.enabled": "1",
			},
		})
		assert.NoError(t, err)

		assert.Equal(t, frontendo11yapi.App{
			ID:   1,
			Name: "Test App",
			Key:  "foobar",
			CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
				{
					ID:  1,
					URL: "*",
				},
			},
			Settings: map[string]string{
				"geolcation.enabled": "1",
			},
			AllowedRate: 0,
		}, acutalApp)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer 1:some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.CreateApp(context.Background(), svr.URL, 1, frontendo11yapi.App{})
		require.NoError(t, err)
	})

	t.Run("returns error when API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.CreateApp(context.Background(), svr.URL, 1, frontendo11yapi.App{Name: "test app"})

		assert.Error(t, err)
		assert.Equal(t, `failed to create faro app "test app": status: 500`, err.Error())
	})

	t.Run("returns ErrUnauthorized when API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.CreateApp(context.Background(), svr.URL, 1, frontendo11yapi.App{Name: "test app"})

		assert.Error(t, err)
		assert.Equal(t, `failed to create faro app "test app": request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, frontendo11yapi.ErrUnauthorized))
	})
}

func TestClient_GetApp(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))
			assert.Equal(t, "/api/v1/app/1", r.URL.Path)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
			{
				"id": 1,
				"name": "Test App",
    			"appKey": "foobar",
    			"corsOrigins": [{
    			    "id": 1,
    			    "url": "*"
    			}],
				"allowedRate": 0
			}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		acutalApp, err := c.GetApp(context.Background(), svr.URL, 1, 1)
		assert.NoError(t, err)

		assert.Equal(t, frontendo11yapi.App{
			ID:   1,
			Name: "Test App",
			Key:  "foobar",
			CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
				{
					ID:  1,
					URL: "*",
				},
			},
			AllowedRate: 0,
		}, acutalApp)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer 1:some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetApp(context.Background(), svr.URL, 1, 1)
		require.NoError(t, err)
	})

	t.Run("returns ErrUnauthorized when frontend o11y API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetApp(context.Background(), svr.URL, 1, 1)

		assert.Error(t, err)
		assert.Equal(t, `failed to get faro app: request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, frontendo11yapi.ErrUnauthorized))
	})

	t.Run("returns error when frontend o11y API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetApp(context.Background(), svr.URL, 1, 1)

		assert.Error(t, err)
		assert.Equal(t, `failed to get faro app: status: 500`, err.Error())
	})
}

func TestClient_GetApps(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))
			assert.Equal(t, "/api/v1/app", r.URL.Path)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
			[
				{
					"id": 1,
					"name": "Test App",
    				"appKey": "foobar",
    				"corsOrigins": [{
    				    "id": 1,
    				    "url": "*"
    				}],
					"allowedRate": 0
				},
				{
					"id": 2,
					"name": "Test App 2",
    				"appKey": "foobar2",
    				"corsOrigins": [{
    				    "id": 1,
    				    "url": "*"
    				}],
					"allowedRate": 0
				}
			]`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		acutalApps, err := c.GetApps(context.Background(), svr.URL, 1)
		assert.NoError(t, err)

		assert.Equal(t, []frontendo11yapi.App{
			{
				ID:   1,
				Name: "Test App",
				Key:  "foobar",
				CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
					{
						ID:  1,
						URL: "*",
					},
				},
				AllowedRate: 0,
			},
			{
				ID:   2,
				Name: "Test App 2",
				Key:  "foobar2",
				CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
					{
						ID:  1,
						URL: "*",
					},
				},
				AllowedRate: 0,
			},
		}, acutalApps)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer 1:some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `[]`)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetApps(context.Background(), svr.URL, 1)
		require.NoError(t, err)
	})

	t.Run("returns ErrUnauthorized when frontend o11y API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetApps(context.Background(), svr.URL, 1)

		assert.Error(t, err)
		assert.Equal(t, `failed to get faro apps: request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, frontendo11yapi.ErrUnauthorized))
	})

	t.Run("returns error when frontend o11y API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.GetApps(context.Background(), svr.URL, 1)

		assert.Error(t, err)
		assert.Equal(t, `failed to get faro apps: status: 500`, err.Error())
	})
}

func TestClient_UpdateApp(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))
			assert.Equal(t, "/api/v1/app/1", r.URL.Path)
			requestBody, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `
			{
				"name": "New Name",
    			"corsOrigins": [{
					"url": "https://grafana.com"
    			}],
				"settings": {
					"geolcation.enabled": "0"
				}
			}`, string(requestBody))

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`
			{
				"id": 1,
				"name": "New Name",
    			"appKey": "foobar",
    			"corsOrigins": [{
    			    "id": 2,
					"url": "https://grafana.com"
    			}],
				"settings": {
					"geolcation.enabled": "0"
				},
				"allowedRate": 0
			}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		acutalApp, err := c.UpdateApp(context.Background(), svr.URL, 1, 1, frontendo11yapi.App{
			Name: "New Name",
			CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
				{
					URL: "https://grafana.com",
				},
			},
			Settings: map[string]string{
				"geolcation.enabled": "0",
			},
		})
		assert.NoError(t, err)

		assert.Equal(t, frontendo11yapi.App{
			ID:   1,
			Name: "New Name",
			Key:  "foobar",
			CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
				{
					ID:  2,
					URL: "https://grafana.com",
				},
			},
			AllowedRate: 0,
			Settings: map[string]string{
				"geolcation.enabled": "0",
			},
		}, acutalApp)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer 1:some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateApp(context.Background(), svr.URL, 1, 1, frontendo11yapi.App{})
		require.NoError(t, err)
	})

	t.Run("returns ErrorNotFound when frontend o11y API responds 404", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateApp(context.Background(), svr.URL, 1, 1, frontendo11yapi.App{Name: "Test App"})

		assert.Error(t, err)
		assert.ErrorIs(t, err, frontendo11yapi.ErrNotFound)
		assert.Equal(t, `failed to update faro app "Test App": not found`, err.Error())
	})

	t.Run("returns Unauthorized when frontend o11y API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateApp(context.Background(), svr.URL, 1, 1, frontendo11yapi.App{Name: "Test App"})

		assert.Error(t, err)
		assert.Equal(t, `failed to update faro app "Test App": request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, frontendo11yapi.ErrUnauthorized))
	})

	t.Run("returns error when frontend o11y API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		_, err = c.UpdateApp(context.Background(), svr.URL, 1, 1, frontendo11yapi.App{Name: "Test App"})

		assert.Error(t, err)
		assert.Equal(t, `failed to update faro app "Test App": status: 500`, err.Error())
	})
}

func TestClient_DeleteApp(t *testing.T) {
	defaultHeaders := map[string]string{"Grafana-Terraform-Provider": "True"}
	t.Run("successfully sends request and receives response", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/app/1", r.URL.Path)

			assert.Equal(t, "True", r.Header.Get("Grafana-Terraform-Provider"))

			w.WriteHeader(http.StatusOK)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteApp(context.Background(), svr.URL, 1, 1)

		assert.NoError(t, err)
	})

	t.Run("sets auth token, content type, user agent", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer 1:some token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "some-user-agent", r.Header.Get("User-Agent"))
			_, _ = fmt.Fprintf(w, `{}`)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteApp(context.Background(), svr.URL, 1, 1)
		require.NoError(t, err)
	})

	t.Run("returns ErrorNotFound when frontend o11y API responds 404", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteApp(context.Background(), svr.URL, 1, 1)

		assert.Error(t, err)
		assert.ErrorIs(t, err, frontendo11yapi.ErrNotFound)
		assert.Equal(t, `failed to delete faro app id=1: not found`, err.Error())
	})

	t.Run("returns ErrUnauthorized when frontend o11y API responds 401", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteApp(context.Background(), svr.URL, 1, 1)

		assert.Error(t, err)
		assert.Equal(t, `failed to delete faro app id=1: request not authorized for stack`, err.Error())
		assert.True(t, errors.Is(err, frontendo11yapi.ErrUnauthorized))
	})

	t.Run("returns error when frontend o11y API responds 500", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"some error"}`))
		}))
		defer svr.Close()

		c, err := frontendo11yapi.NewClient("grafana-dev.com", "some token", svr.Client(), "some-user-agent", defaultHeaders)
		require.NoError(t, err)
		err = c.DeleteApp(context.Background(), svr.URL, 1, 1)

		assert.Error(t, err)
		assert.Equal(t, `failed to delete faro app id=1: status: 500`, err.Error())
	})
}

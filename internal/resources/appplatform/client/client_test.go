package client_test

import (
	"context"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/client"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

type commonClient struct {
	grafanaGetFunc func(subpath string) ([]byte, error)
}

func (c *commonClient) GrafanaGet(_ context.Context, subpath string) ([]byte, error) {
	return c.grafanaGetFunc(subpath)
}

func TestResolvePluralUsesDiscovery(t *testing.T) {
	req := require.New(t)

	apiCallsCount := 0
	c := client.New(rest.Config{APIPath: "/apis"}, &commonClient{
		grafanaGetFunc: func(subpath string) ([]byte, error) {
			apiCallsCount++
			req.Equal("/apis/iam.grafana.app/v0alpha1", subpath)
			return []byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`), nil
		},
	})

	plural, err := c.ResolvePlural(context.TODO(), "iam.grafana.app", "v0alpha1", "Team")
	req.NoError(err)
	req.Equal("teams", plural)
	req.Equal(apiCallsCount, 1)

	// this call shouldn't reach the API
	plural, err = c.ResolvePlural(context.TODO(), "iam.grafana.app", "v0alpha1", "Team")
	req.NoError(err)
	req.Equal("teams", plural)
	req.Equal(apiCallsCount, 1)
}

func TestResolvePluralRejectsClusterScopedKind(t *testing.T) {
	req := require.New(t)

	c := client.New(rest.Config{APIPath: "/apis"}, &commonClient{
		grafanaGetFunc: func(subpath string) ([]byte, error) {
			return []byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":false}]}`), nil
		},
	})

	_, err := c.ResolvePlural(context.TODO(), "iam.grafana.app", "v0alpha1", "Team")
	req.Error(err)
	require.Contains(t, err.Error(), "cluster-scoped")
}

func TestDiscoverGrafanaStackID(t *testing.T) {
	req := require.New(t)

	apiCallsCount := 0
	c := client.New(rest.Config{APIPath: "/apis"}, &commonClient{
		grafanaGetFunc: func(subpath string) ([]byte, error) {
			apiCallsCount++
			req.Equal("/bootdata", subpath)
			return []byte(`{"settings":{"namespace":"stacks-42"}}`), nil
		},
	})

	stackID, err := c.DiscoverGrafanaStackID(context.TODO())
	req.NoError(err)
	req.Equal(int64(42), stackID)
	req.Equal(apiCallsCount, 1)

	// this call shouldn't reach the API
	stackID, err = c.DiscoverGrafanaStackID(context.TODO())
	req.NoError(err)
	req.Equal(int64(42), stackID)
	req.Equal(apiCallsCount, 1)
}

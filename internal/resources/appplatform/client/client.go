package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-app-sdk/k8s"
	"github.com/grafana/grafana-app-sdk/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

const (
	discoveryRequestTimeout = 10 * time.Second
)

var _ resource.ClientGenerator = (*Client)(nil)

type CommonClient interface {
	GrafanaGet(ctx context.Context, subpath string) ([]byte, error)
}

type discoveredAPIResource struct {
	Plural     string
	Namespaced bool
}

type Client struct {
	commonClient      CommonClient
	k8sClientRegistry resource.ClientGenerator

	discoveredResources      map[string]discoveredAPIResource
	discoveredResourcesMutex sync.RWMutex
}

func New(rcfg rest.Config, commonClient CommonClient) *Client {
	return &Client{
		k8sClientRegistry: k8s.NewClientRegistry(rcfg, k8s.ClientConfig{
			NegotiatedSerializerProvider: func(kind resource.Kind) runtime.NegotiatedSerializer {
				return &k8s.KindNegotiatedSerializer{Kind: kind}
			},
		}),
		commonClient:        commonClient,
		discoveredResources: map[string]discoveredAPIResource{},
	}
}

func (g *Client) ClientFor(kind resource.Kind) (resource.Client, error) {
	return g.k8sClientRegistry.ClientFor(kind)
}

func (g *Client) GetCustomRouteClient(version k8sschema.GroupVersion, defaultNamespace string) (resource.CustomRouteClient, error) {
	return g.k8sClientRegistry.GetCustomRouteClient(version, defaultNamespace)
}

func (g *Client) DiscoveryClient() (resource.DiscoveryClient, error) {
	return g.k8sClientRegistry.DiscoveryClient()
}

func (g *Client) ResolvePlural(ctx context.Context, apiGroup, version, kind string) (string, error) {
	discovered, err := g.discoverAPIResource(ctx, apiGroup, version, kind)
	if err != nil {
		return "", err
	}
	if !discovered.Namespaced {
		return "", fmt.Errorf("%s/%s %s is cluster-scoped; this MVP only supports namespaced resources", apiGroup, version, kind)
	}

	return discovered.Plural, nil
}

func (g *Client) discoverAPIResource(ctx context.Context, apiGroup, version, kind string) (discoveredAPIResource, error) {
	discoveryCtx, cancel := context.WithTimeout(ctx, discoveryRequestTimeout)
	defer cancel()

	resourceKey := fmt.Sprintf("%s/%s/%s", apiGroup, version, kind)

	g.discoveredResourcesMutex.RLock()
	if r, ok := g.discoveredResources[resourceKey]; ok {
		g.discoveredResourcesMutex.RUnlock()
		return r, nil
	}
	g.discoveredResourcesMutex.RUnlock()

	g.discoveredResourcesMutex.Lock()
	defer g.discoveredResourcesMutex.Unlock()

	body, err := g.commonClient.GrafanaGet(discoveryCtx, fmt.Sprintf("/apis/%s/%s", apiGroup, version))
	if err != nil {
		return discoveredAPIResource{}, err
	}

	var resources metav1.APIResourceList
	if err := json.Unmarshal(body, &resources); err != nil {
		return discoveredAPIResource{}, fmt.Errorf("failed to decode discovery response: %w", err)
	}

	for _, candidate := range resources.APIResources {
		if strings.Contains(candidate.Name, "/") {
			continue
		}
		if candidate.Kind != kind {
			continue
		}

		discovered := discoveredAPIResource{
			Plural:     candidate.Name,
			Namespaced: candidate.Namespaced,
		}

		g.discoveredResources[resourceKey] = discovered

		return discovered, nil
	}

	return discoveredAPIResource{}, fmt.Errorf("no discovery entry found for kind %q", kind)
}

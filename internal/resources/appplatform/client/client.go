package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	authlib "github.com/grafana/authlib/types"
	"github.com/grafana/grafana-app-sdk/k8s"
	"github.com/grafana/grafana-app-sdk/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

const (
	bootdataRequestTimeout  = 10 * time.Second
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

	discoveredStackID      int64
	discoveredStackIDErr   error
	discoveredStackIDMutex sync.RWMutex
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
		discoveredStackID:   -1,
	}
}

func (c *Client) ClientFor(kind resource.Kind) (resource.Client, error) {
	return c.k8sClientRegistry.ClientFor(kind)
}

func (c *Client) GetCustomRouteClient(version k8sschema.GroupVersion, defaultNamespace string) (resource.CustomRouteClient, error) {
	return c.k8sClientRegistry.GetCustomRouteClient(version, defaultNamespace)
}

func (c *Client) DiscoveryClient() (resource.DiscoveryClient, error) {
	return c.k8sClientRegistry.DiscoveryClient()
}

func (c *Client) DiscoverGrafanaStackID(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, bootdataRequestTimeout)
	defer cancel()

	c.discoveredStackIDMutex.RLock()
	if c.discoveredStackID != -1 {
		c.discoveredStackIDMutex.RUnlock()
		return c.discoveredStackID, c.discoveredStackIDErr
	}
	c.discoveredStackIDMutex.RUnlock()

	c.discoveredStackIDMutex.Lock()
	defer c.discoveredStackIDMutex.Unlock()

	c.discoveredStackID = 0

	body, err := c.commonClient.GrafanaGet(ctx, "/bootdata")
	if err != nil {
		c.discoveredStackIDErr = err
		return c.discoveredStackID, c.discoveredStackIDErr
	}

	var payload struct {
		Settings struct {
			Namespace string `json:"namespace"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.discoveredStackIDErr = fmt.Errorf("failed to decode /bootdata response: %w", err)
		return c.discoveredStackID, c.discoveredStackIDErr
	}

	namespace := strings.TrimSpace(payload.Settings.Namespace)
	if namespace == "" {
		c.discoveredStackIDErr = fmt.Errorf("bootdata returned an empty namespace")
		return c.discoveredStackID, c.discoveredStackIDErr
	}

	parsed, err := authlib.ParseNamespace(namespace)
	if err != nil {
		c.discoveredStackIDErr = fmt.Errorf("failed to parse namespace %q: %w", namespace, err)
		return c.discoveredStackID, c.discoveredStackIDErr
	}

	c.discoveredStackID = parsed.StackID

	if c.discoveredStackID == 0 {
		c.discoveredStackIDErr = fmt.Errorf("bootdata namespace is not a Grafana Cloud stack namespace %q", namespace)
	}

	return c.discoveredStackID, c.discoveredStackIDErr
}

func (c *Client) ResolvePlural(ctx context.Context, apiGroup, version, kind string) (string, error) {
	discovered, err := c.discoverAPIResource(ctx, apiGroup, version, kind)
	if err != nil {
		return "", err
	}
	if !discovered.Namespaced {
		return "", fmt.Errorf("%s/%s %s is cluster-scoped; this MVP only supports namespaced resources", apiGroup, version, kind)
	}

	return discovered.Plural, nil
}

func (c *Client) discoverAPIResource(ctx context.Context, apiGroup, version, kind string) (discoveredAPIResource, error) {
	discoveryCtx, cancel := context.WithTimeout(ctx, discoveryRequestTimeout)
	defer cancel()

	resourceKey := fmt.Sprintf("%s/%s/%s", apiGroup, version, kind)

	c.discoveredResourcesMutex.RLock()
	if r, ok := c.discoveredResources[resourceKey]; ok {
		c.discoveredResourcesMutex.RUnlock()
		return r, nil
	}
	c.discoveredResourcesMutex.RUnlock()

	c.discoveredResourcesMutex.Lock()
	defer c.discoveredResourcesMutex.Unlock()

	body, err := c.commonClient.GrafanaGet(discoveryCtx, fmt.Sprintf("/apis/%s/%s", apiGroup, version))
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

		c.discoveredResources[resourceKey] = discovered

		return discovered, nil
	}

	return discoveredAPIResource{}, fmt.Errorf("no discovery entry found for kind %q", kind)
}

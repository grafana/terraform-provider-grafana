package client

import "github.com/grafana/grafana-app-sdk/resource"

// Registry
type Registry interface {
	ClientFor(resource.Kind) (resource.Client, error)
}

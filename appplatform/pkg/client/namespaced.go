package client

import (
	"context"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana-app-sdk/resource"
)

// NamespacedClient
type NamespacedClient[T resource.Object, L resource.ListObject] struct {
	cli       *ResourceClient[T, L]
	namespace string
}

// NewNamespaced
func NewNamespaced[T resource.Object, L resource.ListObject](
	cli *ResourceClient[T, L], namespaceID int64, orgMode bool,
) *NamespacedClient[T, L] {
	var ns string
	if orgMode {
		ns = claims.OrgNamespaceFormatter(namespaceID)
	} else {
		ns = claims.CloudNamespaceFormatter(namespaceID)
	}

	return &NamespacedClient[T, L]{
		cli:       cli,
		namespace: ns,
	}
}

// List
func (c *NamespacedClient[T, L]) List(ctx context.Context, opts resource.ListOptions) (L, error) {
	return c.cli.List(ctx, c.namespace, opts)
}

// Watch
func (c *NamespacedClient[T, L]) Watch(ctx context.Context, opts resource.WatchOptions) (resource.WatchResponse, error) {
	return c.cli.Watch(ctx, c.namespace, opts)
}

// Get
func (c *NamespacedClient[T, L]) Get(ctx context.Context, uid string) (T, error) {
	return c.cli.Get(ctx, resource.Identifier{
		Namespace: c.namespace,
		Name:      uid,
	})
}

// Create
func (c *NamespacedClient[T, L]) Create(ctx context.Context, obj T, opts resource.CreateOptions) (T, error) {
	obj.SetNamespace(c.namespace)
	return c.cli.Create(ctx, obj, opts)
}

// Update
func (c *NamespacedClient[T, L]) Update(ctx context.Context, obj T, opts resource.UpdateOptions) (T, error) {
	obj.SetNamespace(c.namespace)
	return c.cli.Update(ctx, obj, opts)
}

// Patch
func (c *NamespacedClient[T, L]) Patch(
	ctx context.Context, uid string, req resource.PatchRequest, opts resource.PatchOptions,
) (T, error) {
	return c.cli.Patch(ctx, resource.Identifier{
		Namespace: c.namespace,
		Name:      uid,
	}, req, opts)
}

// Delete
func (c *NamespacedClient[T, L]) Delete(ctx context.Context, uid string, opts resource.DeleteOptions) error {
	return c.cli.Delete(ctx, resource.Identifier{
		Namespace: c.namespace,
		Name:      uid,
	}, opts)
}

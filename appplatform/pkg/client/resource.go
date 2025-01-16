package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/apimachinery/pkg/runtime"
)

// MetaAccessorGetter
type MetaAccessorGetter func(raw runtime.Object) (utils.GrafanaMetaAccessor, error)

// ResourceClient
//
// TODO: automatically set meta values.
// TODO: this will require providing generators / default values to the initializer.
type ResourceClient[T resource.Object, L resource.ListObject] struct {
	cli  resource.Client
	kind resource.Kind
}

// NewResourceClient
func NewResourceClient[T resource.Object, L resource.ListObject](
	cli resource.Client, kind resource.Kind,
) *ResourceClient[T, L] {
	return &ResourceClient[T, L]{
		cli:  cli,
		kind: kind,
	}
}

// List
func (c *ResourceClient[T, L]) List(ctx context.Context, namespace string, opts resource.ListOptions) (L, error) {
	var res L

	v, err := c.cli.List(ctx, namespace, opts)
	if err != nil {
		return res, err
	}

	res, ok := v.(L)
	if !ok {
		return res, fmt.Errorf("expected %T, got %T", res, v)
	}

	return res, nil
}

// Watch
func (c *ResourceClient[T, L]) Watch(
	ctx context.Context, namespace string, opts resource.WatchOptions,
) (resource.WatchResponse, error) {
	return c.cli.Watch(ctx, namespace, opts)
}

// Get
func (c *ResourceClient[T, L]) Get(ctx context.Context, id resource.Identifier) (T, error) {
	var res T

	v, err := c.cli.Get(ctx, id)
	if err != nil {
		return res, err
	}

	res, ok := v.(T)
	if !ok {
		return res, fmt.Errorf("expected %T, got %T", res, v)
	}

	return res, nil
}

// Create
func (c *ResourceClient[T, L]) Create(ctx context.Context, obj T, opts resource.CreateOptions) (T, error) {
	obj.SetGroupVersionKind(c.kind.GroupVersionKind())

	var res T

	json, err := json.Marshal(obj)
	if err != nil {
		return res, err
	}

	tflog.Debug(ctx, "ResourceClient.Create", map[string]any{
		"identifier": obj.GetStaticMetadata().Identifier(),
		"obj":        string(json),
	})

	v, err := c.cli.Create(ctx, obj.GetStaticMetadata().Identifier(), obj, opts)
	if err != nil {
		return res, err
	}

	res, ok := v.(T)
	if !ok {
		return res, fmt.Errorf("expected %T, got %T", res, v)
	}

	return res, nil
}

// Update
func (c *ResourceClient[T, L]) Update(ctx context.Context, obj T, opts resource.UpdateOptions) (T, error) {
	obj.SetGroupVersionKind(c.kind.GroupVersionKind())

	var res T
	v, err := c.cli.Update(ctx, obj.GetStaticMetadata().Identifier(), obj, opts)
	if err != nil {
		return res, err
	}

	res, ok := v.(T)
	if !ok {
		return res, fmt.Errorf("expected %T, got %T", res, v)
	}

	return res, nil
}

// Patch
func (c *ResourceClient[T, L]) Patch(
	ctx context.Context, id resource.Identifier, req resource.PatchRequest, opts resource.PatchOptions,
) (T, error) {
	var res T

	v, err := c.cli.Patch(ctx, id, req, opts)
	if err != nil {
		return res, err
	}

	res, ok := v.(T)
	if !ok {
		return res, fmt.Errorf("expected %T, got %T", res, v)
	}

	return res, nil
}

// Delete
func (c *ResourceClient[T, L]) Delete(ctx context.Context, id resource.Identifier) error {
	return c.cli.Delete(ctx, id)
}

// Package appplatform: K8s-style dashboard handler for the legacy grafana_dashboard resource.
// When config_json is a K8s resource (apiVersion + kind + metadata + spec), the legacy
// resource uses the Grafana App Platform (v1beta1) dashboard API instead of /api/dashboards/db.
package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v1beta1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

const (
	k8sDashboardIDPrefix   = "v1beta1:"
	k8sDashboardAPIVersion = "dashboard.grafana.app/v1beta1"
	k8sDashboardKind       = "Dashboard"
)

// IsK8sDashboardConfig returns true if configJSON looks like a K8s dashboard
// resource (has apiVersion and spec; apiVersion contains v1beta1).
func IsK8sDashboardConfig(configJSON string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(configJSON), &m); err != nil {
		return false
	}
	apiVer, _ := m["apiVersion"].(string)
	_, hasSpec := m["spec"].(map[string]any)
	return hasSpec && strings.Contains(apiVer, "v1beta1")
}

// ParseK8sDashboardConfig parses a K8s-style dashboard config_json and returns
// uid (from metadata.uid or metadata.name), folder_uid, and the spec object.
func ParseK8sDashboardConfig(configJSON string) (uid, folderUID string, spec map[string]any, err error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(configJSON), &m); err != nil {
		return "", "", nil, err
	}
	meta, _ := m["metadata"].(map[string]any)
	if meta != nil {
		if u, ok := meta["uid"].(string); ok && u != "" {
			uid = u
		}
		if uid == "" {
			if n, ok := meta["name"].(string); ok {
				uid = n
			}
		}
		if f, ok := meta["folder_uid"].(string); ok {
			folderUID = f
		}
	}
	spec, _ = m["spec"].(map[string]any)
	if spec == nil {
		return "", "", nil, fmt.Errorf("missing spec in K8s dashboard config")
	}
	if uid == "" {
		return "", "", nil, fmt.Errorf("missing metadata.uid and metadata.name in K8s dashboard config")
	}
	return uid, folderUID, spec, nil
}

// ParseK8sDashboardID returns namespace and uid from a state id "v1beta1:namespace:uid".
// If id is not a v1beta1 id, returns "", "", false.
func ParseK8sDashboardID(id string) (namespace, uid string, ok bool) {
	if !strings.HasPrefix(id, k8sDashboardIDPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(id, k8sDashboardIDPrefix)
	idx := strings.Index(rest, ":")
	if idx <= 0 {
		return "", "", false
	}
	return rest[:idx], rest[idx+1:], true
}

func getDashboardClient(meta *common.Client, orgID int64) (*resource.NamespacedClient[*v1beta1.Dashboard, *v1beta1.DashboardList], string, error) {
	rcli, err := meta.GrafanaAppPlatformAPI.ClientFor(v1beta1.DashboardKind())
	if err != nil {
		return nil, "", err
	}
	// Use resource org when provided; otherwise provider default. Stack takes precedence over org.
	useOrg := orgID
	if useOrg == 0 {
		useOrg = meta.GrafanaOrgID
	}
	ns, errMsg := namespaceForClient(useOrg, meta.GrafanaStackID)
	if errMsg != "" {
		return nil, "", fmt.Errorf("%s", errMsg)
	}
	typed := resource.NewTypedClient[*v1beta1.Dashboard, *v1beta1.DashboardList](rcli, v1beta1.DashboardKind())
	namespaced := resource.NewNamespaced(typed, ns)
	return namespaced, ns, nil
}

// CreateDashboardFromK8s creates a dashboard via the App Platform API from K8s-style
// config. Returns config_json for state (K8s shape), and id "v1beta1:namespace:uid".
func CreateDashboardFromK8s(ctx context.Context, meta *common.Client, orgID int64, uid, folderUID string, spec map[string]any, overwrite bool) (configJSON, id string, diags diag.Diagnostics) {
	client, namespace, err := getDashboardClient(meta, orgID)
	if err != nil {
		return "", "", diag.FromErr(err)
	}

	specCopy := make(map[string]any)
	for k, v := range spec {
		specCopy[k] = v
	}
	delete(specCopy, "version")
	delete(specCopy, "id") // legacy field; v1beta1 API rejects it

	var dashSpec v1beta1.DashboardSpec
	dashSpec.Object = specCopy

	obj := v1beta1.Dashboard{}
	obj.Spec = dashSpec
	metaAcc, err := utils.MetaAccessor(&obj)
	if err != nil {
		return "", "", diag.FromErr(err)
	}
	metaAcc.SetName(uid)
	metaAcc.SetFolder(folderUID)
	if err := setManagerProperties(&obj, meta.GrafanaAppPlatformAPIClientID); err != nil {
		return "", "", diag.FromErr(err)
	}

	res, err := client.Create(ctx, &obj, resource.CreateOptions{})
	if err != nil {
		return "", "", diag.FromErr(err)
	}

	configJSON, _ = dashboardObjectToK8sJSON(res)
	id = k8sDashboardIDPrefix + namespace + ":" + res.GetName()
	return configJSON, id, nil
}

// ReadDashboardFromK8s reads a dashboard from the App Platform API. id must be "v1beta1:namespace:uid".
// Returns config_json for state, uid, folderUID, version, url.
func ReadDashboardFromK8s(ctx context.Context, meta *common.Client, id string) (configJSON, uid, folderUID, version, url string, diags diag.Diagnostics) {
	namespace, uid, ok := ParseK8sDashboardID(id)
	if !ok {
		return "", "", "", "", "", diag.Errorf("invalid v1beta1 dashboard id: %s", id)
	}

	rcli, err := meta.GrafanaAppPlatformAPI.ClientFor(v1beta1.DashboardKind())
	if err != nil {
		return "", "", "", "", "", diag.FromErr(err)
	}
	typed := resource.NewTypedClient[*v1beta1.Dashboard, *v1beta1.DashboardList](rcli, v1beta1.DashboardKind())
	namespaced := resource.NewNamespaced(typed, namespace)

	res, err := namespaced.Get(ctx, uid)
	if err != nil {
		return "", "", "", "", "", diag.FromErr(err)
	}

	configJSON, _ = dashboardObjectToK8sJSON(res)
	metaAcc, _ := utils.MetaAccessor(res)
	folderUID = metaAcc.GetFolder()
	version = res.GetResourceVersion()
	url = metaAcc.GetSelfLink()
	return configJSON, res.GetName(), folderUID, version, url, nil
}

// UpdateDashboardFromK8s updates a dashboard via the App Platform API. id must be "v1beta1:namespace:uid".
func UpdateDashboardFromK8s(ctx context.Context, meta *common.Client, id string, uid, folderUID string, spec map[string]any, overwrite bool) (configJSON string, diags diag.Diagnostics) {
	namespace, _, ok := ParseK8sDashboardID(id)
	if !ok {
		return "", diag.Errorf("invalid v1beta1 dashboard id: %s", id)
	}

	rcli, err := meta.GrafanaAppPlatformAPI.ClientFor(v1beta1.DashboardKind())
	if err != nil {
		return "", diag.FromErr(err)
	}
	typed := resource.NewTypedClient[*v1beta1.Dashboard, *v1beta1.DashboardList](rcli, v1beta1.DashboardKind())
	namespaced := resource.NewNamespaced(typed, namespace)

	existing, err := namespaced.Get(ctx, uid)
	if err != nil {
		return "", diag.FromErr(err)
	}

	specCopy := make(map[string]any)
	for k, v := range spec {
		specCopy[k] = v
	}
	delete(specCopy, "version")
	delete(specCopy, "id") // legacy field; v1beta1 API rejects it

	var dashSpec v1beta1.DashboardSpec
	dashSpec.Object = specCopy
	if err := existing.SetSpec(dashSpec); err != nil {
		return "", diag.FromErr(err)
	}
	metaAcc, _ := utils.MetaAccessor(existing)
	metaAcc.SetFolder(folderUID)
	if err := setManagerProperties(existing, meta.GrafanaAppPlatformAPIClientID); err != nil {
		return "", diag.FromErr(err)
	}

	opts := resource.UpdateOptions{ResourceVersion: existing.GetResourceVersion()}
	if overwrite {
		opts.ResourceVersion = ""
	}
	res, err := namespaced.Update(ctx, existing, opts)
	if err != nil {
		return "", diag.FromErr(err)
	}

	configJSON, _ = dashboardObjectToK8sJSON(res)
	return configJSON, nil
}

// DeleteDashboardFromK8s deletes a dashboard via the App Platform API. id must be "v1beta1:namespace:uid".
func DeleteDashboardFromK8s(ctx context.Context, meta *common.Client, id string) diag.Diagnostics {
	namespace, uid, ok := ParseK8sDashboardID(id)
	if !ok {
		return diag.Errorf("invalid v1beta1 dashboard id: %s", id)
	}

	rcli, err := meta.GrafanaAppPlatformAPI.ClientFor(v1beta1.DashboardKind())
	if err != nil {
		return diag.FromErr(err)
	}
	typed := resource.NewTypedClient[*v1beta1.Dashboard, *v1beta1.DashboardList](rcli, v1beta1.DashboardKind())
	namespaced := resource.NewNamespaced(typed, namespace)

	if err := namespaced.Delete(ctx, uid, resource.DeleteOptions{}); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func dashboardObjectToK8sJSON(obj *v1beta1.Dashboard) (string, error) {
	specCopy := make(map[string]any)
	for k, v := range obj.Spec.Object {
		specCopy[k] = v
	}
	delete(specCopy, "version")
	delete(specCopy, "id")
	metaAcc, _ := utils.MetaAccessor(obj)
	out := map[string]any{
		"apiVersion": k8sDashboardAPIVersion,
		"kind":       k8sDashboardKind,
		"metadata": map[string]any{
			"name":       obj.GetName(),
			"uid":        string(obj.GetUID()),
			"folder_uid": metaAcc.GetFolder(),
		},
		"spec": specCopy,
	}
	b, err := json.Marshal(out)
	return string(b), err
}

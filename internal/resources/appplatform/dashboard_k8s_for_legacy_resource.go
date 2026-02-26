// Package appplatform: K8s-style dashboard handler for the legacy grafana_dashboard resource.
// When config_json is a K8s resource (apiVersion + kind + metadata + spec), the legacy
// resource uses the Grafana App Platform dashboard API instead of /api/dashboards/db.
// This implementation is version-agnostic: it uses k8s.io/client-go/dynamic with
// unstructured objects so any dashboard API version (v0alpha1, v1beta1, v1, ...) works.
package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	k8sDashboardKind     = "Dashboard"
	k8sDashboardResource = "dashboards"
)

// IsK8sDashboardConfig returns true if configJSON looks like a K8s dashboard
// resource (has apiVersion under dashboard.grafana.app and a spec object).
func IsK8sDashboardConfig(configJSON string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(configJSON), &m); err != nil {
		return false
	}
	apiVer, _ := m["apiVersion"].(string)
	_, hasSpec := m["spec"].(map[string]any)
	return hasSpec && strings.HasPrefix(apiVer, "dashboard.grafana.app/")
}

// ParseK8sDashboardConfig parses a K8s-style dashboard config_json and returns
// the apiVersion, uid (from metadata.name), folder_uid, and the spec object.
func ParseK8sDashboardConfig(configJSON string) (apiVersion, uid, folderUID string, spec map[string]any, err error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(configJSON), &m); err != nil {
		return "", "", "", nil, err
	}
	apiVersion, _ = m["apiVersion"].(string)
	if apiVersion == "" {
		return "", "", "", nil, fmt.Errorf("missing apiVersion in K8s dashboard config")
	}
	meta, _ := m["metadata"].(map[string]any)
	if meta != nil {
		if n, ok := meta["name"].(string); ok && n != "" {
			uid = n
		}
		if uid == "" {
			if u, ok := meta["uid"].(string); ok && u != "" {
				uid = u
			}
		}
		if f, ok := meta["folder_uid"].(string); ok {
			folderUID = f
		}
	}
	spec, _ = m["spec"].(map[string]any)
	if spec == nil {
		return "", "", "", nil, fmt.Errorf("missing spec in K8s dashboard config")
	}
	if uid == "" {
		return "", "", "", nil, fmt.Errorf("missing metadata.name in K8s dashboard config")
	}
	return apiVersion, uid, folderUID, spec, nil
}

// getDynamicDashboardClient creates a dynamic K8s client scoped to the dashboard
// resource in the appropriate namespace for the given org/stack.
func getDynamicDashboardClient(meta *common.Client, apiVersion string, orgID int64) (dynamic.ResourceInterface, string, error) {
	if meta.GrafanaAppPlatformRestConfig == nil {
		return nil, "", fmt.Errorf("Grafana App Platform REST config is not configured")
	}

	gv, err := k8sschema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, "", fmt.Errorf("invalid apiVersion %q: %w", apiVersion, err)
	}

	useOrg := orgID
	if useOrg == 0 {
		useOrg = meta.GrafanaOrgID
	}
	ns, errMsg := namespaceForClient(useOrg, meta.GrafanaStackID)
	if errMsg != "" {
		return nil, "", fmt.Errorf("%s", errMsg)
	}

	dynClient, err := dynamic.NewForConfig(meta.GrafanaAppPlatformRestConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create dynamic K8s client: %w", err)
	}

	return dynClient.Resource(gv.WithResource(k8sDashboardResource)).Namespace(ns), ns, nil
}

// prepareUnstructuredDashboard builds a clean unstructured dashboard object
// ready for a Create or Update call, following the same cleanup pattern as
// Grafana's saveDashboardViaK8s.
func prepareUnstructuredDashboard(apiVersion, uid, folderUID string, spec map[string]any, clientID string) (*unstructured.Unstructured, error) {
	specCopy := make(map[string]any, len(spec))
	for k, v := range spec {
		specCopy[k] = v
	}
	delete(specCopy, "version")
	delete(specCopy, "id")

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": apiVersion,
			"kind":       k8sDashboardKind,
			"metadata":   map[string]any{},
			"spec":       specCopy,
		},
	}

	if uid != "" {
		obj.SetName(uid)
	} else {
		obj.SetGenerateName("a")
	}

	delete(obj.Object, "status")

	metaAcc, err := utils.MetaAccessor(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to access metadata: %w", err)
	}
	metaAcc.SetFolder(folderUID)
	metaAcc.SetResourceVersionInt64(0)
	metaAcc.SetUID("")
	obj.SetResourceVersion("")
	obj.SetFinalizers(nil)
	obj.SetManagedFields(nil)

	metaAcc.SetManagerProperties(utils.ManagerProperties{
		Kind:     utils.ManagerKindTerraform,
		Identity: clientID,
	})

	return obj, nil
}

// CreateDashboardFromK8s creates a dashboard via the App Platform API from K8s-style
// config. Returns config_json for state (K8s shape) and the resulting dashboard UID.
func CreateDashboardFromK8s(ctx context.Context, meta *common.Client, orgID int64, apiVersion, uid, folderUID string, spec map[string]any, overwrite bool) (configJSON, resultUID string, diags diag.Diagnostics) {
	client, _, err := getDynamicDashboardClient(meta, apiVersion, orgID)
	if err != nil {
		return "", "", diag.FromErr(err)
	}

	obj, err := prepareUnstructuredDashboard(apiVersion, uid, folderUID, spec, meta.GrafanaAppPlatformAPIClientID)
	if err != nil {
		return "", "", diag.FromErr(err)
	}

	res, err := client.Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return "", "", diag.FromErr(err)
	}

	configJSON, err = unstructuredToK8sJSON(res)
	if err != nil {
		return "", "", diag.FromErr(err)
	}
	return configJSON, res.GetName(), nil
}

// UpdateDashboardFromK8s updates a dashboard via the App Platform API.
func UpdateDashboardFromK8s(ctx context.Context, meta *common.Client, orgID int64, apiVersion, uid, folderUID string, spec map[string]any, overwrite bool) (configJSON string, diags diag.Diagnostics) {
	client, _, err := getDynamicDashboardClient(meta, apiVersion, orgID)
	if err != nil {
		return "", diag.FromErr(err)
	}

	existing, err := client.Get(ctx, uid, metav1.GetOptions{})
	if err != nil {
		return "", diag.FromErr(err)
	}

	obj, err := prepareUnstructuredDashboard(apiVersion, uid, folderUID, spec, meta.GrafanaAppPlatformAPIClientID)
	if err != nil {
		return "", diag.FromErr(err)
	}

	if !overwrite {
		obj.SetResourceVersion(existing.GetResourceVersion())
	}

	res, err := client.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return "", diag.FromErr(err)
	}

	configJSON, err = unstructuredToK8sJSON(res)
	if err != nil {
		return "", diag.FromErr(err)
	}
	return configJSON, nil
}

// DeleteDashboardFromK8s deletes a dashboard via the App Platform API.
func DeleteDashboardFromK8s(ctx context.Context, meta *common.Client, orgID int64, apiVersion, uid string) diag.Diagnostics {
	client, _, err := getDynamicDashboardClient(meta, apiVersion, orgID)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.Delete(ctx, uid, metav1.DeleteOptions{}); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// unstructuredToK8sJSON converts a K8s API response (unstructured) to the clean
// K8s-style JSON stored in config_json state.
func unstructuredToK8sJSON(obj *unstructured.Unstructured) (string, error) {
	metaAcc, err := utils.MetaAccessor(obj)
	if err != nil {
		return "", err
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec != nil {
		delete(spec, "version")
		delete(spec, "id")
	}

	out := map[string]any{
		"apiVersion": obj.GetAPIVersion(),
		"kind":       k8sDashboardKind,
		"metadata": map[string]any{
			"name":       obj.GetName(),
			"uid":        string(obj.GetUID()),
			"folder_uid": metaAcc.GetFolder(),
		},
		"spec": spec,
	}
	b, err := json.Marshal(out)
	return string(b), err
}

// ReconstructK8sConfigJSON wraps a legacy dashboard body (from the /api read)
// into K8s envelope format, using the apiVersion from the prior state's config_json.
func ReconstructK8sConfigJSON(existingConfig string, dashboardBody map[string]any, folderUID string) string {
	var existing map[string]any
	if err := json.Unmarshal([]byte(existingConfig), &existing); err != nil {
		return existingConfig
	}
	apiVersion, _ := existing["apiVersion"].(string)
	if apiVersion == "" {
		return existingConfig
	}

	spec := make(map[string]any, len(dashboardBody))
	for k, v := range dashboardBody {
		spec[k] = v
	}
	delete(spec, "id")
	delete(spec, "version")
	delete(spec, "uid")

	out := map[string]any{
		"apiVersion": apiVersion,
		"kind":       k8sDashboardKind,
		"metadata": map[string]any{
			"name":       dashboardBody["uid"],
			"folder_uid": folderUID,
		},
		"spec": spec,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return existingConfig
	}
	return string(b)
}

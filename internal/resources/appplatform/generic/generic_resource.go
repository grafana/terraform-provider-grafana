package generic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	apicommon "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfrsc "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

const (
	genericResourceTypeName = "grafana_apps_generic_resource"
	bootdataRequestTimeout  = 10 * time.Second
	discoveryRequestTimeout = 10 * time.Second
)

var (
	errGenericResourceRecreated = errors.New("resource was recreated with a different UID")
)

var (
	supportedGenericManifestFields = map[string]struct{}{
		"apiVersion": {},
		"kind":       {},
		"metadata":   {},
		"spec":       {},
		"status":     {},
	}
)

type genericResource struct {
	client *common.Client
}

type GenericResourceModel struct {
	ID             types.String  `tfsdk:"id"`
	Manifest       types.Dynamic `tfsdk:"manifest"`
	Secure         types.Dynamic `tfsdk:"secure"`
	SecureVersion  types.Int64   `tfsdk:"secure_version"`
	AllowUIUpdates types.Bool    `tfsdk:"allow_ui_updates"`
}

type resolvedGenericResource struct {
	APIGroup  string
	Version   string
	Kind      string
	Plural    string
	Namespace string
	Name      string
	Object    *genericUntypedObject
}

type genericIdentity struct {
	APIGroup string
	Kind     string
	Name     string
}

type discoveredAPIResource struct {
	Plural     string
	Namespaced bool
}

type genericUntypedObject struct {
	sdkresource.UntypedObject
	rawMetadata map[string]any
}

func newGenericUntypedObject(name string, metadata map[string]any, spec map[string]any) (*genericUntypedObject, error) {
	objectMeta, err := objectMetaFromNormalizedMetadata(name, metadata)
	if err != nil {
		return nil, err
	}

	return &genericUntypedObject{
		UntypedObject: sdkresource.UntypedObject{
			ObjectMeta: objectMeta,
			Spec:       cloneMap(spec),
		},
		rawMetadata: cloneMap(metadata),
	}, nil
}

func (o *genericUntypedObject) DeepCopyObject() runtime.Object {
	return o.Copy()
}

func (o *genericUntypedObject) Copy() sdkresource.Object {
	copied, _ := o.UntypedObject.Copy().(*sdkresource.UntypedObject)
	if copied == nil {
		copied = &sdkresource.UntypedObject{}
	}

	return &genericUntypedObject{
		UntypedObject: *copied,
		rawMetadata:   cloneMap(o.rawMetadata),
	}
}

func (o *genericUntypedObject) UnmarshalJSON(data []byte) error {
	if err := o.UntypedObject.UnmarshalJSON(data); err != nil {
		return err
	}

	var payload struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	o.rawMetadata = cloneMap(payload.Metadata)
	return nil
}

func (o *genericUntypedObject) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"kind":       o.Kind,
		"apiVersion": o.APIVersion,
		"metadata":   objectMetadataForWire(o.rawMetadata, o.ObjectMeta),
		"spec":       o.Spec,
	}
	for key, value := range o.Subresources {
		payload[key] = value
	}
	return json.Marshal(payload)
}

func GenericResource() appplatform.NamedResource {
	return appplatform.NamedResource{
		Resource: &genericResource{},
		Name:     genericResourceTypeName,
		Category: common.CategoryGrafanaApps,
	}
}

func (r *genericResource) Metadata(_ context.Context, _ tfrsc.MetadataRequest, resp *tfrsc.MetadataResponse) {
	resp.TypeName = genericResourceTypeName
}

func (r *genericResource) Schema(_ context.Context, _ tfrsc.SchemaRequest, resp *tfrsc.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages arbitrary Grafana App Platform resources when a typed Terraform resource is not yet available.",
		MarkdownDescription: `
Manages arbitrary Grafana App Platform resources when a typed Terraform resource is not yet available. The resource is still experimental; diffing semantics are subject to change - feedback welcome in https://github.com/grafana/terraform-provider-grafana/issues.

This resource accepts a Kubernetes-style ` + "`manifest`" + ` as the single source of truth for the resource definition. Use HCL ` + "`merge()`" + ` if you need to inject Terraform variables into a static manifest file.

Only namespaced App Platform kinds are supported. The provider autodiscovers the namespace from ` + "`/bootdata`" + ` on every operation. If autodiscovery does not find a cloud stack namespace, the provider falls back to the explicit ` + "`stack_id`" + ` and then ` + "`org_id`" + ` provider settings.

Top-level manifest fields are limited to ` + "`apiVersion`" + `, ` + "`kind`" + `, ` + "`metadata`" + `, ` + "`spec`" + `, and the ignored ` + "`status`" + ` field. If ` + "`metadata.namespace`" + ` is configured, it must match the provider-selected namespace.

Inside ` + "`manifest.metadata`" + `, both Kubernetes ` + "`name`" + ` and ` + "`uid`" + ` are accepted as input aliases for the object identifier.

The top-level ` + "`secure`" + ` argument is write-only and requires Terraform 1.11 or later. Each configured key must set exactly one of ` + "`create`" + ` or ` + "`name`" + `, and Terraform only re-sends those secure values when ` + "`secure_version`" + ` changes.

Reads refresh managed drift from the API. Metadata drift is limited to the metadata keys you configured; ` + "`spec`" + ` is authoritative, so extra remote spec fields are refreshed into state and will drift until Terraform restores the configured object.

Import format:

` + "```text\n" + `terraform import grafana_apps_generic_resource.example <api_group>/<version>/<kind>/<object_name>
` + "```" + `

Import stores a normalized manifest without noisy server-managed metadata such as ` + "`resourceVersion`" + ` or ` + "`managedFields`" + `. Because ` + "`secure`" + ` is write-only, imported configurations still need you to add ` + "`secure`" + ` and ` + "`secure_version`" + ` manually afterward.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The API resource UUID assigned by Grafana. This is not used for import; import uses the object name stored in `metadata.uid`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"manifest": schema.DynamicAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Kubernetes-style manifest, typically from `yamldecode(file(...))` or `jsondecode(file(...))`. Must contain `apiVersion`, `kind`, `metadata` (with `name` or `uid`), and `spec`. Use HCL `merge()` to inject Terraform variables. If you start from an exported manifest, remove noisy server-managed metadata such as `resourceVersion`, `generation`, and `managedFields`, or import the resource first and use the normalized state shape. If `metadata.namespace` is set, it must match the namespace selected from provider `org_id` or `stack_id` / autodiscovery. Top-level manifest fields are limited to `apiVersion`, `kind`, `metadata`, `spec`, and the ignored `status` field. The `secure` field must not be set here; use the top-level `secure` argument instead.",
			},
			"secure": schema.DynamicAttribute{
				Optional:    true,
				WriteOnly:   true,
				Description: "Write-only secure values map. Each key must contain exactly one of `create` or `name`; empty objects are invalid.",
			},
			"secure_version": schema.Int64Attribute{
				Optional:    true,
				Description: "Set this to 1 when using `secure`, then increment it whenever you want Terraform to re-apply secure values.",
			},
			"allow_ui_updates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the resource can be edited from the Grafana UI. Defaults to `false` — Terraform-managed resources are locked from UI edits unless you opt in. Set to `true` to allow UI modifications; not supported by all resources.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *genericResource) ConfigValidators(_ context.Context) []tfrsc.ConfigValidator {
	return []tfrsc.ConfigValidator{
		genericConfigValidator{},
		genericSecureVersionValidator{},
	}
}

func (r *genericResource) ModifyPlan(ctx context.Context, req tfrsc.ModifyPlanRequest, resp *tfrsc.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || !req.Plan.Raw.IsKnown() {
		return
	}

	planModel, diags := getGenericResourceModelFromData(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if req.State.Raw.IsNull() || !req.State.Raw.IsKnown() {
		return
	}

	stateModel, diags := getGenericResourceModelFromData(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if genericIdentityReplacementRequired(planModel, stateModel) {
		resp.RequiresReplace = append(resp.RequiresReplace,
			path.Root("manifest"),
		)
	}
}

func (r *genericResource) Configure(_ context.Context, req tfrsc.ConfigureRequest, resp *tfrsc.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if client.GrafanaAppPlatformAPI == nil {
		resp.Diagnostics.AddError(
			"Grafana App Platform API client not configured",
			"The grafana provider must be configured with `url` and `auth` to use App Platform resources.",
		)
		return
	}

	r.client = client
}

func (r *genericResource) Create(ctx context.Context, req tfrsc.CreateRequest, resp *tfrsc.CreateResponse) {
	model, diags := getGenericResourceModelFromData(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resolved, diags := r.resolveResource(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := setManagerProperties(resolved.Object, r.client.GrafanaAppPlatformAPIClientID, genericAllowUIUpdates(model)); err != nil {
		resp.Diagnostics.AddError("Failed to configure manager metadata", err.Error())
		return
	}

	resp.Diagnostics.Append(r.applySecureFromConfig(ctx, req.Config, resolved.Object)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.clientForResolved(resolved)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := client.Create(ctx, resolved.Object, sdkresource.CreateOptions{})
	if err != nil {
		resp.Diagnostics.Append(appplatform.ErrorToDiagnostics(appplatform.ResourceActionCreate, resolved.Name, genericResourceTypeName, err)...)
		return
	}

	resp.Diagnostics.Append(r.setComputedState(&model, created)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *genericResource) Read(ctx context.Context, req tfrsc.ReadRequest, resp *tfrsc.ReadResponse) {
	model, diags := getGenericResourceModelFromData(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resolved, diags := r.resolveResource(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.clientForResolved(resolved)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := client.Get(ctx, resolved.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.Append(appplatform.ErrorToDiagnostics(appplatform.ResourceActionRead, resolved.Name, genericResourceTypeName, err)...)
		return
	}
	if resourceUIDChanged(model.ID, current) {
		addResourceReplacedOutsideTerraformError(&resp.Diagnostics)
		return
	}

	resp.Diagnostics.Append(r.refreshManagedState(ctx, &model, current, resolved)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *genericResource) Update(ctx context.Context, req tfrsc.UpdateRequest, resp *tfrsc.UpdateResponse) {
	planModel, diags := getGenericResourceModelFromData(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateModel, diags := getGenericResourceModelFromData(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planIdentity, identityDiags := resolveGenericIdentity(ctx, planModel)
	if identityDiags.HasError() {
		resp.Diagnostics.Append(identityDiags...)
		return
	}

	stateIdentity, identityDiags := resolveGenericIdentity(ctx, stateModel)
	if identityDiags.HasError() {
		resp.Diagnostics.Append(identityDiags...)
		return
	}

	if planIdentity != stateIdentity {
		resp.Diagnostics.AddError(
			"Resource identity changed",
			"Changing the API group, `kind`, or resource name requires replacing the resource. Version changes are applied in-place.",
		)
		return
	}

	resolved, diags := r.resolveResource(ctx, planModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateResolved, diags := r.resolveResource(ctx, stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clearManagedAnnotations := metadataStringMapExplicitlyEmpty(planModel, "annotations")

	if err := setManagerProperties(resolved.Object, r.client.GrafanaAppPlatformAPIClientID, genericAllowUIUpdates(planModel)); err != nil {
		resp.Diagnostics.AddError("Failed to configure manager metadata", err.Error())
		return
	}

	if secureVersionChanged(planModel.SecureVersion, stateModel.SecureVersion) {
		resp.Diagnostics.Append(r.applySecureFromConfig(ctx, req.Config, resolved.Object)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	client, diags := r.clientForResolved(resolved)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updated *genericUntypedObject
	err := retryOnConflict(ctx, conflictRetryAttempts, conflictRetryDelay, func(attempt int) error {
		current, err := client.Get(ctx, resolved.Name)
		if err != nil {
			return err
		}
		if resourceUIDChanged(stateModel.ID, current) {
			return errGenericResourceRecreated
		}

		merged := mergeManagedObject(current, stateResolved.Object, resolved.Object, clearManagedAnnotations)
		resourceVersion := current.GetResourceVersion()
		merged.SetResourceVersion(resourceVersion)

		updated, err = client.Update(ctx, merged, sdkresource.UpdateOptions{
			ResourceVersion: resourceVersion,
		})
		return err
	})
	if err != nil {
		if errors.Is(err, errGenericResourceRecreated) {
			addResourceReplacedOutsideTerraformError(&resp.Diagnostics)
			return
		}

		resp.Diagnostics.Append(appplatform.ErrorToDiagnostics(appplatform.ResourceActionUpdate, resolved.Name, genericResourceTypeName, err)...)
		return
	}

	resp.Diagnostics.Append(r.setComputedState(&planModel, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &planModel)...)
}

func (r *genericResource) Delete(ctx context.Context, req tfrsc.DeleteRequest, resp *tfrsc.DeleteResponse) {
	model, diags := getGenericResourceModelFromData(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resolved, diags := r.resolveResource(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.clientForResolved(resolved)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteOptions := sdkresource.DeleteOptions{}
	if !model.ID.IsNull() && !model.ID.IsUnknown() {
		deleteOptions.Preconditions.UID = strings.TrimSpace(model.ID.ValueString())
	}

	if err := retryOnConflict(ctx, conflictRetryAttempts, conflictRetryDelay, func(_ int) error {
		return client.Delete(ctx, resolved.Name, deleteOptions)
	}); err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		if apierrors.IsConflict(err) && deleteOptions.Preconditions.UID != "" {
			addResourceReplacedOutsideTerraformError(&resp.Diagnostics)
			return
		}

		resp.Diagnostics.Append(appplatform.ErrorToDiagnostics(appplatform.ResourceActionDelete, resolved.Name, genericResourceTypeName, err)...)
	}
}

func (r *genericResource) ImportState(ctx context.Context, req tfrsc.ImportStateRequest, resp *tfrsc.ImportStateResponse) {
	id, diags := parseGenericImportID(req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, diags := r.resolveNamespace(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plural, err := r.resolvePlural(ctx, id.APIGroup, id.Version, id.Kind)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to resolve API route",
			err.Error(),
		)
		return
	}

	resolved := resolvedGenericResource{
		APIGroup:  id.APIGroup,
		Version:   id.Version,
		Kind:      id.Kind,
		Plural:    plural,
		Namespace: namespace,
		Name:      id.Name,
	}

	client, diags := r.clientForResolved(resolved)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := client.Get(ctx, id.Name)
	if err != nil {
		resp.Diagnostics.Append(appplatform.ErrorToDiagnostics(appplatform.ResourceActionRead, id.Name, genericResourceTypeName, err)...)
		return
	}

	model := GenericResourceModel{
		Secure:         types.DynamicNull(),
		SecureVersion:  types.Int64Null(),
		AllowUIUpdates: types.BoolValue(readAllowUIUpdatesFromObject(obj)),
	}

	model.Manifest, diags = goToDynamicValue(ctx, importedManifest(obj, id.APIGroup, id.Version, id.Kind))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.setComputedState(&model, obj)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *genericResource) resolveResource(ctx context.Context, model GenericResourceModel) (resolvedGenericResource, diag.Diagnostics) {
	input, diags := resolveGenericInput(ctx, model)
	if diags.HasError() {
		return resolvedGenericResource{}, diags
	}

	namespace, namespaceDiags := r.resolveNamespace(ctx)
	diags.Append(namespaceDiags...)
	if diags.HasError() {
		return resolvedGenericResource{}, diags
	}

	plural, err := r.resolvePlural(ctx, input.APIGroup, input.Version, input.Kind)
	if err != nil {
		diags.AddError(
			"Failed to resolve API route",
			fmt.Sprintf(
				"Could not resolve the plural route for %s/%s %s: %s.",
				input.APIGroup,
				input.Version,
				input.Kind,
				err.Error(),
			),
		)
		return resolvedGenericResource{}, diags
	}

	if configuredNamespace := strings.TrimSpace(input.Object.GetNamespace()); configuredNamespace != "" && configuredNamespace != namespace {
		namespacePath := input.NamespacePath
		if namespacePath.Equal(path.Empty()) {
			namespacePath = path.Root("manifest").AtMapKey("metadata").AtMapKey("namespace")
		}
		diags.AddAttributeError(
			namespacePath,
			"Namespace does not match provider context",
			fmt.Sprintf(
				"The configured `metadata.namespace` must match the provider-selected namespace %q for this Grafana stack or org.",
				namespace,
			),
		)
		return resolvedGenericResource{}, diags
	}
	input.Object.SetNamespace(namespace)

	return resolvedGenericResource{
		APIGroup:  input.APIGroup,
		Version:   input.Version,
		Kind:      input.Kind,
		Plural:    plural,
		Namespace: namespace,
		Name:      input.Name,
		Object:    input.Object,
	}, diags
}

func (r *genericResource) clientForResolved(resolved resolvedGenericResource) (*sdkresource.NamespacedClient[*genericUntypedObject, *sdkresource.UntypedList], diag.Diagnostics) {
	return r.clientForResolvedWithNamespace(resolved, resolved.Namespace)
}

func (r *genericResource) clientForResolvedWithNamespace(
	resolved resolvedGenericResource,
	namespace string,
) (*sdkresource.NamespacedClient[*genericUntypedObject, *sdkresource.UntypedList], diag.Diagnostics) {
	var diags diag.Diagnostics

	kind := sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			resolved.APIGroup,
			resolved.Version,
			&genericUntypedObject{},
			&sdkresource.UntypedList{},
			sdkresource.WithKind(resolved.Kind),
			sdkresource.WithPlural(resolved.Plural),
			sdkresource.WithScope(sdkresource.NamespacedScope),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: sdkresource.NewPassthroughJSONCodec(),
		},
	}

	cli, err := r.client.GrafanaAppPlatformAPI.ClientFor(kind)
	if err != nil {
		diags.AddError("Failed to create Grafana App Platform client", err.Error())
		return nil, diags
	}

	return sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*genericUntypedObject, *sdkresource.UntypedList](cli, kind),
		namespace,
	), diags
}

func (r *genericResource) resolveNamespace(ctx context.Context) (string, diag.Diagnostics) {
	return appplatform.ResolveNamespace(ctx, r.client)
}

func (r *genericResource) resolvePlural(ctx context.Context, apiGroup, version, kind string) (string, error) {
	discovered, err := r.discoverAPIResource(ctx, apiGroup, version, kind)
	if err != nil {
		return "", err
	}
	if !discovered.Namespaced {
		return "", fmt.Errorf("%s/%s %s is cluster-scoped; this MVP only supports namespaced resources", apiGroup, version, kind)
	}

	return discovered.Plural, nil
}

func normalizeMetadataMapKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	normalized := make([]string, 0, len(keys))
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func configuredNamespacePath(manifestMetadata map[string]any) path.Path {
	if _, ok := manifestMetadata["namespace"]; ok {
		return path.Root("manifest").AtMapKey("metadata").AtMapKey("namespace")
	}
	return path.Empty()
}

func (r *genericResource) discoverAPIResource(ctx context.Context, apiGroup, version, kind string) (discoveredAPIResource, error) {
	return r.discoverAPIResourceWithTimeout(ctx, apiGroup, version, kind, discoveryRequestTimeout)
}

func (r *genericResource) discoverAPIResourceWithTimeout(
	ctx context.Context,
	apiGroup string,
	version string,
	kind string,
	timeout time.Duration,
) (discoveredAPIResource, error) {
	discoveryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body, err := appplatform.GrafanaGet(discoveryCtx, r.client, fmt.Sprintf("/apis/%s/%s", apiGroup, version))
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
		return discovered, nil
	}

	return discoveredAPIResource{}, fmt.Errorf("no discovery entry found for kind %q", kind)
}

func (r *genericResource) applySecureFromConfig(ctx context.Context, cfg tfsdk.Config, dst *genericUntypedObject) diag.Diagnostics {
	var (
		secure types.Dynamic
		diags  diag.Diagnostics
	)

	diags.Append(cfg.GetAttribute(ctx, path.Root("secure"), &secure)...)
	if diags.HasError() || secure.IsNull() || secure.IsUnknown() {
		return diags
	}

	secureFields, err := genericSecureFieldValues(secure)
	if err != nil {
		diags.AddAttributeError(
			path.Root("secure"),
			"Invalid secure configuration",
			err.Error(),
		)
		return diags
	}

	values, valueDiags := parseGenericConfiguredSecureValues(secureFields)
	diags.Append(valueDiags...)
	if diags.HasError() || len(values) == 0 {
		return diags
	}

	if err := dst.SetSubresource("secure", secureSubresourcePayload(values)); err != nil {
		diags.AddError("Failed to encode secure subresource", err.Error())
	}

	return diags
}

func (r *genericResource) setComputedState(
	model *GenericResourceModel,
	obj *genericUntypedObject,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(string(obj.GetUID()))

	model.Manifest = normalizeDynamicState(model.Manifest)
	model.Secure = types.DynamicNull()

	// Set allow_ui_updates: preserve config value if set, otherwise default to false.
	if model.AllowUIUpdates.IsNull() || model.AllowUIUpdates.IsUnknown() {
		model.AllowUIUpdates = types.BoolValue(false)
	}

	return diags
}

func (r *genericResource) refreshManagedState(
	ctx context.Context,
	model *GenericResourceModel,
	obj *genericUntypedObject,
	resolved resolvedGenericResource,
) diag.Diagnostics {
	var diags diag.Diagnostics

	currentManifest, manifestDiags := dynamicMapValue(ctx, "manifest", model.Manifest)
	diags.Append(manifestDiags...)
	if diags.HasError() {
		return diags
	}

	// Build the refreshed manifest preserving only config-scoped keys.
	refreshedManifest := refreshManifestState(currentManifest, resolved, obj)

	model.ID = types.StringValue(string(obj.GetUID()))

	model.Manifest, manifestDiags = dynamicStateFromMap(ctx, model.Manifest, refreshedManifest)
	diags.Append(manifestDiags...)
	model.Secure = types.DynamicNull()

	// Refresh allow_ui_updates from the live object's manager properties.
	model.AllowUIUpdates = types.BoolValue(readAllowUIUpdatesFromObject(obj))

	return diags
}

func getGenericResourceModelFromData(ctx context.Context, src resourceData) (GenericResourceModel, diag.Diagnostics) {
	var (
		model GenericResourceModel
		diags diag.Diagnostics
	)

	diags.Append(src.GetAttribute(ctx, path.Root("id"), &model.ID)...)
	diags.Append(src.GetAttribute(ctx, path.Root("manifest"), &model.Manifest)...)
	diags.Append(src.GetAttribute(ctx, path.Root("secure"), &model.Secure)...)
	diags.Append(src.GetAttribute(ctx, path.Root("secure_version"), &model.SecureVersion)...)
	diags.Append(src.GetAttribute(ctx, path.Root("allow_ui_updates"), &model.AllowUIUpdates)...)

	return model, diags
}

type genericResolvedInput struct {
	APIGroup      string
	Version       string
	Kind          string
	Name          string
	Object        *genericUntypedObject
	NamespacePath path.Path
}

func resolveGenericInput(ctx context.Context, model GenericResourceModel) (genericResolvedInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	manifest, manifestDiags := dynamicMapValue(ctx, "manifest", model.Manifest)
	diags.Append(manifestDiags...)
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	if _, exists := manifest["secure"]; exists {
		diags.AddAttributeError(
			path.Root("manifest"),
			"Invalid manifest",
			"The `secure` field must not be set inside `manifest`. Use the top-level write-only `secure` argument instead.",
		)
		return genericResolvedInput{}, diags
	}

	validationDiags := validateSupportedGenericManifest(manifest)
	diags.Append(validationDiags...)
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	apiGroup, version, apiVersionDiags := manifestAPIVersion(manifest)
	diags.Append(apiVersionDiags...)
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	kind, kindDiags := manifestStringField(manifest, "kind", path.Root("manifest").AtMapKey("kind"))
	diags.Append(kindDiags...)
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	metadataFromManifest, manifestDiags := mapField(manifest, "metadata", path.Root("manifest").AtMapKey("metadata"))
	diags.Append(manifestDiags...)
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	specFromManifest, manifestDiags := mapField(manifest, "spec", path.Root("manifest").AtMapKey("spec"))
	diags.Append(manifestDiags...)
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	name, normalizedMetadata, metadataDiags := normalizeMetadata(metadataFromManifest, path.Root("manifest").AtMapKey("metadata"))
	diags.Append(metadataDiags...)
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	switch {
	case apiGroup == "":
		diags.AddAttributeError(path.Root("manifest").AtMapKey("apiVersion"), "Missing API group", "Provide `manifest.apiVersion`.")
	case version == "":
		diags.AddAttributeError(path.Root("manifest").AtMapKey("apiVersion"), "Missing API version", "Provide `manifest.apiVersion`.")
	case kind == "":
		diags.AddAttributeError(path.Root("manifest").AtMapKey("kind"), "Missing kind", "Provide `manifest.kind`.")
	case name == "":
		diags.AddAttributeError(path.Root("manifest").AtMapKey("metadata"), "Missing metadata identifier", "Provide `manifest.metadata.name` or `manifest.metadata.uid`.")
	}
	if diags.HasError() {
		return genericResolvedInput{}, diags
	}

	obj, err := newGenericUntypedObject(name, normalizedMetadata, specFromManifest)
	if err != nil {
		diags.AddAttributeError(
			path.Root("manifest").AtMapKey("metadata"),
			"Invalid metadata",
			fmt.Sprintf("Failed to encode metadata for the API request: %s.", err.Error()),
		)
		return genericResolvedInput{}, diags
	}

	return genericResolvedInput{
		APIGroup:      apiGroup,
		Version:       version,
		Kind:          kind,
		Name:          name,
		Object:        obj,
		NamespacePath: configuredNamespacePath(metadataFromManifest),
	}, diags
}

func resolveGenericIdentity(ctx context.Context, model GenericResourceModel) (genericIdentity, diag.Diagnostics) {
	resolved, diags := resolveGenericInput(ctx, model)
	if diags.HasError() {
		return genericIdentity{}, diags
	}

	return genericIdentity{
		APIGroup: resolved.APIGroup,
		Kind:     resolved.Kind,
		Name:     resolved.Name,
	}, nil
}

func resolveGenericIdentityForPlan(model GenericResourceModel) (genericIdentity, bool) {
	apiGroup, ok := identityGroupForPlan(model)
	if !ok {
		return genericIdentity{}, false
	}

	kind, ok := identityKindForPlan(model)
	if !ok {
		return genericIdentity{}, false
	}

	name, ok := identityNameForPlan(model)
	if !ok {
		return genericIdentity{}, false
	}

	if apiGroup == "" || kind == "" || name == "" {
		return genericIdentity{}, false
	}

	return genericIdentity{
		APIGroup: apiGroup,
		Kind:     kind,
		Name:     name,
	}, true
}

func genericIdentityReplacementRequired(planModel GenericResourceModel, stateModel GenericResourceModel) bool {
	stateIdentity, stateIdentityKnown := resolveGenericIdentityForPlan(stateModel)
	if !stateIdentityKnown {
		return false
	}

	planIdentity, planIdentityKnown := resolveGenericIdentityForPlan(planModel)
	if planIdentityKnown {
		return planIdentity != stateIdentity
	}

	return genericIdentityHasUnknownInputs(planModel)
}

func genericIdentityHasUnknownInputs(model GenericResourceModel) bool {
	if _, status := stringFieldAtPath(model.Manifest, "kind"); status == attrPathUnknown {
		return true
	}

	uid, status := stringFieldAtPath(model.Manifest, "metadata", "uid")
	if status == attrPathUnknown {
		return true
	}
	if status == attrPathKnown && uid != "" {
		return false
	}

	_, status = stringFieldAtPath(model.Manifest, "metadata", "name")
	return status == attrPathUnknown
}

func identityGroupForPlan(model GenericResourceModel) (string, bool) {
	apiGroup, _, ok := manifestGroupVersionForPlan(model.Manifest)
	return apiGroup, ok && apiGroup != ""
}

func identityKindForPlan(model GenericResourceModel) (string, bool) {
	kind, status := stringFieldAtPath(model.Manifest, "kind")
	return kind, status == attrPathKnown && kind != ""
}

func identityNameForPlan(model GenericResourceModel) (string, bool) {
	if uid, status := stringFieldAtPath(model.Manifest, "metadata", "uid"); status == attrPathUnknown {
		return "", false
	} else if uid != "" {
		return uid, true
	}

	name, status := stringFieldAtPath(model.Manifest, "metadata", "name")
	return name, status == attrPathKnown && name != ""
}

func manifestGroupVersionForPlan(manifest types.Dynamic) (string, string, bool) {
	apiVersion, status := stringFieldAtPath(manifest, "apiVersion")
	if status != attrPathKnown || apiVersion == "" {
		return "", "", false
	}

	gv, err := k8sschema.ParseGroupVersion(apiVersion)
	if err != nil {
		return "", "", false
	}

	return gv.Group, gv.Version, true
}

func manifestAPIVersion(manifest map[string]any) (string, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(manifest) == 0 {
		return "", "", diags
	}

	raw, ok := manifest["apiVersion"]
	if !ok || raw == nil {
		return "", "", diags
	}

	apiVersion, ok := raw.(string)
	if !ok {
		diags.AddAttributeError(
			path.Root("manifest").AtMapKey("apiVersion"),
			"Invalid apiVersion",
			fmt.Sprintf("Expected `manifest.apiVersion` to be a string, got %T.", raw),
		)
		return "", "", diags
	}

	gv, err := k8sschema.ParseGroupVersion(apiVersion)
	if err != nil {
		diags.AddAttributeError(
			path.Root("manifest").AtMapKey("apiVersion"),
			"Invalid apiVersion",
			err.Error(),
		)
		return "", "", diags
	}

	return gv.Group, gv.Version, diags
}

func manifestStringField(manifest map[string]any, key string, p path.Path) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	raw, ok := manifest[key]
	if !ok || raw == nil {
		return "", diags
	}

	str, ok := raw.(string)
	if !ok {
		diags.AddAttributeError(
			p,
			fmt.Sprintf("Invalid %s", key),
			fmt.Sprintf("Expected `%s` to be a string, got %T.", key, raw),
		)
		return "", diags
	}

	return str, diags
}

func mapField(parent map[string]any, key string, p path.Path) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	raw, ok := parent[key]
	if !ok || raw == nil {
		return map[string]any{}, diags
	}

	cast, ok := raw.(map[string]any)
	if !ok {
		diags.AddAttributeError(
			p,
			fmt.Sprintf("Invalid %s", key),
			fmt.Sprintf("Expected `%s` to be an object, got %T.", key, raw),
		)
		return nil, diags
	}

	return cloneMap(cast), diags
}

func normalizeMetadata(metadata map[string]any, p path.Path) (string, map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	normalized := map[string]any{}

	name, _ := stringFieldAlias(metadata, "name")
	uid, _ := stringFieldAlias(metadata, "uid")
	if name != "" && uid != "" && name != uid {
		diags.AddAttributeError(
			p,
			"Conflicting metadata identifier",
			"`metadata.name` and `metadata.uid` must either match or only one of them may be set.",
		)
		return "", nil, diags
	}
	if uid != "" {
		name = uid
	}

	for key, value := range metadata {
		switch key {
		case "name", "uid":
			continue
		case "labels", "annotations":
			normalizedMap, mapDiags := normalizeStringMapField(metadata, key, p.AtMapKey(key))
			diags.Append(mapDiags...)
			if diags.HasError() {
				return "", nil, diags
			}
			if normalizedMap != nil {
				normalized[key] = normalizedMap
			}
		default:
			normalized[key] = cloneValue(value)
		}
	}

	return name, normalized, diags
}

func normalizeStringMapField(values map[string]any, fieldName string, p path.Path) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	raw, ok := values[fieldName]
	if !ok || raw == nil {
		return nil, diags
	}

	input, ok := raw.(map[string]any)
	if !ok {
		diags.AddAttributeError(
			p,
			fmt.Sprintf("Invalid %s", fieldName),
			fmt.Sprintf("Expected `metadata.%s` to be an object of strings, got %T.", fieldName, raw),
		)
		return nil, diags
	}

	normalized := make(map[string]any, len(input))
	for key, item := range input {
		str, ok := item.(string)
		if !ok {
			diags.AddAttributeError(
				p.AtMapKey(key),
				fmt.Sprintf("Invalid %s value", fieldName),
				fmt.Sprintf("Expected `metadata.%s[%q]` to be a string, got %T.", fieldName, key, item),
			)
			continue
		}

		normalized[key] = str
	}

	return normalized, diags
}

func validateSupportedGenericManifest(manifest map[string]any) diag.Diagnostics {
	var diags diag.Diagnostics

	for key := range manifest {
		if _, ok := supportedGenericManifestFields[key]; ok {
			continue
		}

		diags.AddAttributeError(
			path.Root("manifest").AtMapKey(key),
			"Unsupported manifest field",
			fmt.Sprintf(
				"Only `manifest.apiVersion`, `manifest.kind`, `manifest.metadata`, `manifest.spec`, and the ignored server-managed `manifest.status` field are supported by `grafana_apps_generic_resource`. Remove `%s` from the manifest or switch to a typed resource if you need that field.",
				key,
			),
		)
	}

	return diags
}

func metadataStringMapExplicitlyEmpty(model GenericResourceModel, field string) bool {
	keys, present, known := stringMapKeysAtPath(model.Manifest, "metadata", field)
	return present && known && len(keys) == 0
}

type attrPathState int

const (
	attrPathMissing attrPathState = iota
	attrPathKnown
	attrPathUnknown
)

func stringMapKeysAtPath(value attr.Value, keys ...string) ([]string, bool, bool) {
	fieldValue, status := attrValueAtPath(value, keys...)
	switch status {
	case attrPathMissing:
		return nil, false, true
	case attrPathUnknown:
		return nil, true, false
	}

	fieldValue = unwrapDynamicAttrValue(fieldValue)
	switch v := fieldValue.(type) {
	case types.Object:
		return normalizeMetadataMapKeys(attrMapKeys(v.Attributes())), true, true
	case types.Map:
		return normalizeMetadataMapKeys(attrMapKeys(v.Elements())), true, true
	default:
		return nil, true, false
	}
}

func stringFieldAtPath(value attr.Value, keys ...string) (string, attrPathState) {
	fieldValue, status := attrValueAtPath(value, keys...)
	if status != attrPathKnown {
		return "", status
	}

	fieldValue = unwrapDynamicAttrValue(fieldValue)
	stringValue, ok := fieldValue.(types.String)
	if !ok {
		return "", attrPathMissing
	}
	if stringValue.IsUnknown() {
		return "", attrPathUnknown
	}
	if stringValue.IsNull() {
		return "", attrPathMissing
	}

	return strings.TrimSpace(stringValue.ValueString()), attrPathKnown
}

func attrValueAtPath(value attr.Value, keys ...string) (attr.Value, attrPathState) {
	current := value
	for _, key := range keys {
		current = unwrapDynamicAttrValue(current)
		if current == nil || current.IsNull() {
			return nil, attrPathMissing
		}
		if current.IsUnknown() {
			return nil, attrPathUnknown
		}

		fields, err := dynamicFields(current)
		if err != nil {
			return nil, attrPathMissing
		}

		next, ok := fields[key]
		if !ok {
			return nil, attrPathMissing
		}
		current = next
	}

	current = unwrapDynamicAttrValue(current)
	if current == nil || current.IsNull() {
		return nil, attrPathMissing
	}
	if current.IsUnknown() {
		return nil, attrPathUnknown
	}

	return current, attrPathKnown
}

func unwrapDynamicAttrValue(value attr.Value) attr.Value {
	current := value
	for {
		dynamicValue, ok := current.(types.Dynamic)
		if !ok {
			return current
		}
		if dynamicValue.IsNull() || dynamicValue.IsUnknown() || dynamicValue.UnderlyingValue() == nil {
			return dynamicValue
		}
		current = dynamicValue.UnderlyingValue()
	}
}

func attrMapKeys(fields map[string]attr.Value) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	return keys
}

func dynamicMapValue(ctx context.Context, fieldName string, value types.Dynamic) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return map[string]any{}, diags
	}

	goValue, valueDiags := attrValueToGo(ctx, path.Root(fieldName), value)
	diags.Append(valueDiags...)
	if diags.HasError() {
		return nil, diags
	}

	if goValue == nil {
		return map[string]any{}, diags
	}

	m, ok := goValue.(map[string]any)
	if !ok {
		diags.AddAttributeError(
			path.Root(fieldName),
			"Invalid value",
			fmt.Sprintf("Expected `%s` to be an object, got %T.", fieldName, goValue),
		)
		return nil, diags
	}

	return m, diags
}

func attrValueToGo(ctx context.Context, p path.Path, value attr.Value) (any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value == nil || value.IsNull() {
		return nil, diags
	}

	if value.IsUnknown() {
		diags.AddAttributeError(p, "Unknown value not supported", fmt.Sprintf("`%s` contains unknown values that cannot be sent to the API.", p.String()))
		return nil, diags
	}

	switch v := value.(type) {
	case types.Dynamic:
		return attrValueToGo(ctx, p, v.UnderlyingValue())
	case types.String:
		return v.ValueString(), diags
	case types.Bool:
		return v.ValueBool(), diags
	case types.Int64:
		return v.ValueInt64(), diags
	case types.Number:
		numberText := v.ValueBigFloat().Text('f', -1)
		if integer, ok := new(big.Int).SetString(numberText, 10); ok && integer.IsInt64() {
			return integer.Int64(), diags
		}
		return json.Number(numberText), diags
	case types.Object:
		result := make(map[string]any, len(v.Attributes()))
		for key, nested := range v.Attributes() {
			nestedValue, nestedDiags := attrValueToGo(ctx, p.AtMapKey(key), nested)
			diags.Append(nestedDiags...)
			if diags.HasError() {
				return nil, diags
			}
			result[key] = nestedValue
		}
		return result, diags
	case types.Map:
		result := make(map[string]any, len(v.Elements()))
		for key, nested := range v.Elements() {
			nestedValue, nestedDiags := attrValueToGo(ctx, p.AtMapKey(key), nested)
			diags.Append(nestedDiags...)
			if diags.HasError() {
				return nil, diags
			}
			result[key] = nestedValue
		}
		return result, diags
	case types.List:
		items := make([]any, 0, len(v.Elements()))
		for idx, nested := range v.Elements() {
			nestedValue, nestedDiags := attrValueToGo(ctx, p.AtListIndex(idx), nested)
			diags.Append(nestedDiags...)
			if diags.HasError() {
				return nil, diags
			}
			items = append(items, nestedValue)
		}
		return items, diags
	case types.Tuple:
		items := make([]any, 0, len(v.Elements()))
		for idx, nested := range v.Elements() {
			nestedValue, nestedDiags := attrValueToGo(ctx, p.AtListIndex(idx), nested)
			diags.Append(nestedDiags...)
			if diags.HasError() {
				return nil, diags
			}
			items = append(items, nestedValue)
		}
		return items, diags
	case types.Set:
		items := make([]any, 0, len(v.Elements()))
		for _, nested := range v.Elements() {
			nestedValue, nestedDiags := attrValueToGo(ctx, p, nested)
			diags.Append(nestedDiags...)
			if diags.HasError() {
				return nil, diags
			}
			items = append(items, nestedValue)
		}
		return items, diags
	default:
		diags.AddAttributeError(
			p,
			"Unsupported value type",
			fmt.Sprintf("The provider cannot convert %T at `%s` into JSON.", value, p.String()),
		)
		return nil, diags
	}
}

func goToDynamicValue(ctx context.Context, value any) (types.Dynamic, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrValue, valueDiags := goToAttrValue(ctx, value)
	diags.Append(valueDiags...)
	if diags.HasError() {
		return types.DynamicNull(), diags
	}

	return types.DynamicValue(attrValue), diags
}

func goToAttrValue(ctx context.Context, value any) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch v := value.(type) {
	case nil:
		return types.DynamicNull(), diags
	case string:
		return types.StringValue(v), diags
	case bool:
		return types.BoolValue(v), diags
	case int:
		return types.Int64Value(int64(v)), diags
	case int64:
		return types.Int64Value(v), diags
	case json.Number:
		if integer, err := v.Int64(); err == nil {
			return types.Int64Value(integer), diags
		}

		floatValue, _, err := big.ParseFloat(v.String(), 10, 256, big.ToNearestEven)
		if err != nil {
			diags.AddError("Failed to parse number", err.Error())
			return nil, diags
		}
		return types.NumberValue(floatValue), diags
	case float64:
		if integer, accuracy := big.NewFloat(v).Int64(); accuracy == big.Exact {
			return types.Int64Value(integer), diags
		}
		return types.NumberValue(big.NewFloat(v)), diags
	case []any:
		values := make([]attr.Value, 0, len(v))
		elemTypes := make([]attr.Type, 0, len(v))
		for _, item := range v {
			itemValue, itemDiags := goToAttrValue(ctx, item)
			diags.Append(itemDiags...)
			if diags.HasError() {
				return nil, diags
			}
			values = append(values, types.DynamicValue(itemValue))
			elemTypes = append(elemTypes, types.DynamicType)
		}

		tupleValue, tupleDiags := types.TupleValue(elemTypes, values)
		diags.Append(tupleDiags...)
		return tupleValue, diags
	case map[string]any:
		attrTypes := make(map[string]attr.Type, len(v))
		attrValues := make(map[string]attr.Value, len(v))
		for key, item := range v {
			itemValue, itemDiags := goToAttrValue(ctx, item)
			diags.Append(itemDiags...)
			if diags.HasError() {
				return nil, diags
			}
			attrValues[key] = itemValue
			attrTypes[key] = itemValue.Type(ctx)
		}

		objectValue, objectDiags := types.ObjectValue(attrTypes, attrValues)
		diags.Append(objectDiags...)
		return objectValue, diags
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			diags.AddError("Failed to marshal value", err.Error())
			return nil, diags
		}

		var normalized any
		if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
			diags.AddError("Failed to normalize value", err.Error())
			return nil, diags
		}

		return goToAttrValue(ctx, normalized)
	}
}

func dynamicFields(value attr.Value) (map[string]attr.Value, error) {
	switch v := value.(type) {
	case types.Dynamic:
		if v.IsNull() || v.IsUnknown() || v.UnderlyingValue() == nil {
			return map[string]attr.Value{}, nil
		}
		return dynamicFields(v.UnderlyingValue())
	case types.Object:
		return v.Attributes(), nil
	case types.Map:
		return v.Elements(), nil
	default:
		return nil, fmt.Errorf("expected an object, got %T", value)
	}
}

func normalizeDynamicState(value types.Dynamic) types.Dynamic {
	if value.IsUnknown() {
		return types.DynamicNull()
	}
	return value
}

func stringFieldAlias(values map[string]any, key string) (string, bool) {
	raw, ok := values[key]
	if !ok || raw == nil {
		return "", false
	}

	str, ok := raw.(string)
	if !ok {
		return "", false
	}

	return str, true
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}

	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = cloneValue(value)
	}
	return dst
}

func cloneValue(src any) any {
	switch v := src.(type) {
	case map[string]any:
		return cloneMap(v)
	case []any:
		items := make([]any, len(v))
		for idx, item := range v {
			items[idx] = cloneValue(item)
		}
		return items
	default:
		return v
	}
}

func stringMapValue(value any) map[string]string {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	result := make(map[string]string, len(m))
	for key, item := range m {
		if item == nil {
			continue
		}
		result[key] = fmt.Sprintf("%v", item)
	}
	return result
}

type genericImportID struct {
	APIGroup string
	Version  string
	Kind     string
	Name     string
}

func parseGenericImportID(raw string) (genericImportID, diag.Diagnostics) {
	var diags diag.Diagnostics

	parts := strings.Split(raw, "/")
	if len(parts) != 4 {
		diags.AddError(
			"Invalid import ID",
			"Expected `<api_group>/<version>/<kind>/<object_name>`.",
		)
		return genericImportID{}, diags
	}

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			diags.AddError(
				"Invalid import ID",
				"Expected `<api_group>/<version>/<kind>/<object_name>`.",
			)
			return genericImportID{}, diags
		}
	}

	return genericImportID{
		APIGroup: parts[0],
		Version:  parts[1],
		Kind:     parts[2],
		Name:     parts[3],
	}, diags
}

func importedManifest(obj *genericUntypedObject, apiGroup, version, kind string) map[string]any {
	manifest := map[string]any{
		"apiVersion": k8sschema.GroupVersion{Group: apiGroup, Version: version}.String(),
		"kind":       kind,
	}
	if metadata := importedMetadataMapFromObject(obj); len(metadata) > 0 {
		manifest["metadata"] = metadata
	}
	if spec := importedSpec(obj); len(spec) > 0 {
		manifest["spec"] = spec
	}

	return manifest
}

func importedMetadataMapFromObject(obj *genericUntypedObject) map[string]any {
	metadata := metadataMapFromObject(obj)

	// Re-add the object name since metadataMapFromObject strips it.
	if name := strings.TrimSpace(obj.GetName()); name != "" {
		metadata["name"] = name
	}

	for _, key := range []string{
		"namespace",
		"resourceVersion",
		"creationTimestamp",
		"generation",
		"managedFields",
		"deletionTimestamp",
		"deletionGracePeriodSeconds",
	} {
		delete(metadata, key)
	}

	return metadata
}

func objectMetadataForWire(rawMetadata map[string]any, objectMeta metav1.ObjectMeta) map[string]any {
	result := cloneMap(rawMetadata)
	if result == nil {
		result = map[string]any{}
	}

	for key, value := range metadataMapFromObjectMeta(objectMeta) {
		result[key] = cloneValue(value)
	}

	return result
}

func metadataMapFromObjectMeta(objectMeta metav1.ObjectMeta) map[string]any {
	payload, err := json.Marshal(objectMeta)
	if err != nil {
		return map[string]any{}
	}

	result := map[string]any{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return map[string]any{}
	}

	return result
}

func metadataMapFromObject(obj *genericUntypedObject) map[string]any {
	if obj == nil {
		return map[string]any{}
	}

	result := objectMetadataForWire(obj.rawMetadata, obj.ObjectMeta)

	delete(result, "name")
	delete(result, "uid")

	annotations := mapValueOrEmpty(result["annotations"])
	for _, key := range []string{utils.AnnoKeyManagerIdentity, utils.AnnoKeyManagerKind, utils.AnnoKeyManagerAllowsEdits} {
		delete(annotations, key)
	}
	if len(annotations) > 0 {
		result["annotations"] = annotations
	} else {
		delete(result, "annotations")
	}

	return result
}

func addResourceReplacedOutsideTerraformError(diags *diag.Diagnostics) {
	diags.AddError(
		"Resource replaced outside Terraform",
		"The resource with this identifier was deleted and recreated with a different UID outside Terraform. Re-import the replacement object if you want Terraform to manage it, or delete it deliberately before retrying.",
	)
}

func importedSpec(obj *genericUntypedObject) map[string]any {
	if len(obj.Spec) == 0 {
		return nil
	}
	return cloneMap(obj.Spec)
}

// refreshConfigScopedSpec builds a refreshed spec by updating config keys with
// live server values. Only keys present in the config spec are included so that
// server-added defaults (e.g. tags=null, extra timeSettings fields) do not
// leak into state and cause false drift on the next plan.
func refreshConfigScopedSpec(configSpec map[string]any, liveSpec map[string]any) map[string]any {
	if len(configSpec) == 0 {
		return map[string]any{}
	}

	refreshed := make(map[string]any, len(configSpec))
	for key, configValue := range configSpec {
		liveValue, exists := liveSpec[key]
		if !exists {
			// Key was in config but server removed it — preserve config value
			// so Terraform detects the drift and re-applies.
			refreshed[key] = cloneValue(configValue)
			continue
		}

		// For nested maps, recurse to preserve config structure.
		configMap, configIsMap := configValue.(map[string]any)
		liveMap, liveIsMap := liveValue.(map[string]any)
		if configIsMap && liveIsMap {
			nested := refreshConfigScopedSpec(configMap, liveMap)
			// Also add live keys that aren't in config at this depth,
			// so server-added nested fields cause drift.
			for liveKey, liveNestedValue := range liveMap {
				if _, exists := nested[liveKey]; !exists {
					nested[liveKey] = cloneValue(liveNestedValue)
				}
			}
			refreshed[key] = nested
			continue
		}

		refreshed[key] = cloneValue(liveValue)
	}

	return refreshed
}

func stringMapToAny(input map[string]string) map[string]any {
	if len(input) == 0 {
		return nil
	}

	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func objectMetaFromNormalizedMetadata(name string, metadata map[string]any) (metav1.ObjectMeta, error) {
	encoded := cloneMap(metadata)
	if encoded == nil {
		encoded = map[string]any{}
	}
	encoded["name"] = name

	payload, err := json.Marshal(encoded)
	if err != nil {
		return metav1.ObjectMeta{}, err
	}

	var objectMeta metav1.ObjectMeta
	if err := json.Unmarshal(payload, &objectMeta); err != nil {
		return metav1.ObjectMeta{}, err
	}

	return objectMeta, nil
}

type genericConfigValidator struct{}

func (v genericConfigValidator) Description(context.Context) string {
	return "Validates the generic resource identity and manifest shape."
}

func (v genericConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v genericConfigValidator) ValidateResource(ctx context.Context, req tfrsc.ValidateConfigRequest, resp *tfrsc.ValidateConfigResponse) {
	model, diags := getGenericResourceModelFromData(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !genericConfigHasUnknownInputs(model) {
		_, diags = resolveGenericInput(ctx, model)
		resp.Diagnostics.Append(diags...)
	}

	if !attrValueHasUnknown(model.Secure) {
		resp.Diagnostics.Append(validateGenericSecureConfigValue(model.Secure)...)
	}
}

func genericConfigHasUnknownInputs(model GenericResourceModel) bool {
	return attrValueHasUnknown(model.Manifest)
}

func attrValueHasUnknown(value attr.Value) bool {
	if value == nil || value.IsNull() {
		return false
	}

	if value.IsUnknown() {
		return true
	}

	switch v := value.(type) {
	case types.Dynamic:
		if v.UnderlyingValue() == nil {
			return false
		}
		return attrValueHasUnknown(v.UnderlyingValue())
	case types.Object:
		for _, nested := range v.Attributes() {
			if attrValueHasUnknown(nested) {
				return true
			}
		}
	case types.Map:
		for _, nested := range v.Elements() {
			if attrValueHasUnknown(nested) {
				return true
			}
		}
	case types.List:
		for _, nested := range v.Elements() {
			if attrValueHasUnknown(nested) {
				return true
			}
		}
	case types.Tuple:
		for _, nested := range v.Elements() {
			if attrValueHasUnknown(nested) {
				return true
			}
		}
	case types.Set:
		for _, nested := range v.Elements() {
			if attrValueHasUnknown(nested) {
				return true
			}
		}
	}

	return false
}

func validateGenericSecureConfigValue(secure attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics
	if secure == nil || secure.IsNull() {
		return diags
	}

	fields, err := genericSecureFieldValues(secure)
	if err != nil {
		diags.AddAttributeError(
			path.Root("secure"),
			"Invalid secure configuration",
			err.Error(),
		)
		return diags
	}

	_, valueDiags := parseGenericConfiguredSecureValues(fields)
	diags.Append(valueDiags...)
	return diags
}

func genericSecureFieldValues(value attr.Value) (map[string]attr.Value, error) {
	switch v := value.(type) {
	case types.Dynamic:
		if v.IsNull() || v.IsUnknown() || v.UnderlyingValue() == nil {
			return map[string]attr.Value{}, nil
		}
		return genericSecureFieldValues(v.UnderlyingValue())
	default:
		return secureFieldValues(value)
	}
}

func parseGenericConfiguredSecureValues(fields map[string]attr.Value) (apicommon.InlineSecureValues, diag.Diagnostics) {
	var diags diag.Diagnostics

	secureValues := make(apicommon.InlineSecureValues)
	for fieldName, fieldValue := range fields {
		parsedValue, shouldSet, fieldDiags := parseInlineSecureValue(fieldName, fieldValue)
		diags.Append(fieldDiags...)
		if fieldDiags.HasError() {
			continue
		}
		if !shouldSet {
			diags.AddError(
				"failed to parse secure values",
				fmt.Sprintf("secure field %q object must set exactly one of `name` or `create`", fieldName),
			)
			continue
		}

		secureValues[fieldName] = parsedValue
	}

	return secureValues, diags
}

func validateGenericSecureVersionRequirement(secure attr.Value, secureVersion types.Int64, description string) diag.Diagnostics {
	var diags diag.Diagnostics
	if !genericSecureVersionRequiredValue(secure) || secureVersion.IsUnknown() || !secureVersion.IsNull() {
		return diags
	}

	diags.AddAttributeError(
		path.Root("secure_version"),
		"Missing secure version",
		description,
	)

	return diags
}

func genericSecureVersionRequiredValue(secure attr.Value) bool {
	if secure == nil || secure.IsNull() {
		return false
	}

	if secure.IsUnknown() {
		return true
	}

	attrs, err := genericSecureFieldValues(secure)
	if err != nil {
		return false
	}

	for _, value := range attrs {
		if hasSecureValueInput(value) {
			return true
		}
	}

	return false
}

func dynamicStateFromMap(ctx context.Context, current types.Dynamic, value map[string]any) (types.Dynamic, diag.Diagnostics) {
	if current.IsNull() && len(value) == 0 {
		return types.DynamicNull(), nil
	}

	return goToDynamicValue(ctx, value)
}

func refreshManifestState(currentManifest map[string]any, resolved resolvedGenericResource, obj *genericUntypedObject) map[string]any {
	if len(currentManifest) == 0 {
		return map[string]any{}
	}

	state := map[string]any{}

	if _, ok := currentManifest["apiVersion"]; ok {
		state["apiVersion"] = k8sschema.GroupVersion{Group: resolved.APIGroup, Version: resolved.Version}.String()
	}
	if _, ok := currentManifest["kind"]; ok {
		state["kind"] = resolved.Kind
	}

	currentManifestMetadata, _ := mapValue(currentManifest["metadata"])
	manifestMetadataState := refreshConfiguredManifestMetadataState(currentManifestMetadata, metadataMapFromObject(obj), obj.GetName())
	if len(manifestMetadataState) > 0 {
		state["metadata"] = manifestMetadataState
	}

	// Build spec: start from config keys (preserving nested types), then add
	// server-only top-level keys for drift detection - anything
	// added compared to config should cause drift.
	liveSpec := importedSpec(obj)
	configSpec, configHasSpec := mapValue(currentManifest["spec"])
	if configHasSpec || len(liveSpec) > 0 {
		refreshedSpec := refreshConfigScopedSpec(configSpec, liveSpec)
		// Add top-level keys that the server added (not in config).
		for key, value := range liveSpec {
			if _, exists := refreshedSpec[key]; !exists {
				refreshedSpec[key] = cloneValue(value)
			}
		}
		state["spec"] = refreshedSpec
	}

	if status, ok := currentManifest["status"]; ok {
		state["status"] = status
	}

	return state
}

func refreshConfiguredManifestMetadataState(current map[string]any, actual map[string]any, actualName string) map[string]any {
	if len(current) == 0 {
		return map[string]any{}
	}

	refreshed := make(map[string]any)
	for key, currentValue := range current {
		switch key {
		case "name", "uid":
			if strings.TrimSpace(actualName) != "" {
				refreshed[key] = actualName
				continue
			}
		}

		actualValue, exists := actual[key]

		if !exists {
			if explicitEmptyCollection(currentValue) {
				refreshed[key] = cloneValue(currentValue)
			}
			continue
		}

		// For map-valued metadata fields (annotations, labels, custom),
		// scope to config keys only so server-added keys don't leak into
		// state and cause false drift.
		currentMap, currentIsMap := currentValue.(map[string]any)
		actualMap, actualIsMap := actualValue.(map[string]any)
		if currentIsMap && actualIsMap {
			refreshed[key] = refreshConfigScopedSpec(currentMap, actualMap)
			continue
		}

		refreshed[key] = cloneValue(actualValue)
	}

	return refreshed
}

func explicitEmptyCollection(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		return len(v) == 0
	case []any:
		return len(v) == 0
	default:
		return false
	}
}

func mapValue(value any) (map[string]any, bool) {
	if value == nil {
		return map[string]any{}, false
	}

	cast, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}, false
	}

	return cloneMap(cast), true
}

func mapValueOrEmpty(value any) map[string]any {
	cast, _ := mapValue(value)
	return cast
}

func mergeManagedObject(
	current *genericUntypedObject,
	previousManaged *genericUntypedObject,
	nextManaged *genericUntypedObject,
	clearManagedAnnotations bool,
) *genericUntypedObject {
	merged, _ := current.Copy().(*genericUntypedObject)
	if merged == nil {
		merged = &genericUntypedObject{}
	}

	merged.Subresources = nil
	merged.SetName(nextManaged.GetName())
	mergedMetadataMap := mergeManagedMetadata(
		metadataMapFromObject(current),
		metadataMapFromObject(previousManaged),
		metadataMapFromObject(nextManaged),
		clearManagedAnnotations,
	)
	mergedMetadata, err := objectMetaFromNormalizedMetadata(nextManaged.GetName(), mergedMetadataMap)
	if err == nil {
		mergedMetadata.UID = current.GetUID()
		if mergedMetadata.GetNamespace() == "" {
			mergedMetadata.SetNamespace(nextManaged.GetNamespace())
		}
		merged.ObjectMeta = mergedMetadata
		merged.rawMetadata = cloneMap(mergedMetadataMap)
	}
	merged.SetLabels(mergeManagedStringMap(current.GetLabels(), previousManaged.GetLabels(), nextManaged.GetLabels()))
	merged.SetAnnotations(mergeManagedAnnotations(
		current.GetAnnotations(),
		previousManaged.GetAnnotations(),
		nextManaged.GetAnnotations(),
		clearManagedAnnotations,
	))
	merged.Spec = cloneMap(nextManaged.Spec)

	if secureSubresource, ok := nextManaged.GetSubresource("secure"); ok {
		_ = merged.SetSubresource("secure", secureSubresource)
	}

	return merged
}

func mergeManagedMetadata(
	current map[string]any,
	previousManaged map[string]any,
	nextManaged map[string]any,
	clearManagedAnnotations bool,
) map[string]any {
	merged := cloneMap(current)
	if merged == nil {
		merged = map[string]any{}
	}

	for key := range previousManaged {
		switch key {
		case "labels", "annotations":
			continue
		default:
			delete(merged, key)
		}
	}

	for key, value := range nextManaged {
		switch key {
		case "labels", "annotations":
			continue
		default:
			merged[key] = cloneValue(value)
		}
	}

	mergedLabels := mergeManagedStringMap(
		stringMapValue(current["labels"]),
		stringMapValue(previousManaged["labels"]),
		stringMapValue(nextManaged["labels"]),
	)
	if mergedLabels == nil {
		delete(merged, "labels")
	} else {
		merged["labels"] = stringMapToAny(mergedLabels)
	}

	mergedAnnotations := mergeManagedAnnotations(
		stringMapValue(current["annotations"]),
		stringMapValue(previousManaged["annotations"]),
		stringMapValue(nextManaged["annotations"]),
		clearManagedAnnotations,
	)
	if mergedAnnotations == nil {
		delete(merged, "annotations")
	} else {
		merged["annotations"] = stringMapToAny(mergedAnnotations)
	}

	return merged
}

func mergeManagedStringMap(current map[string]string, previousManaged map[string]string, nextManaged map[string]string) map[string]string {
	// Three-way merge: preserve unconfigured server keys.
	// Start from current (server state), then apply managed changes.
	merged := make(map[string]string, len(current)+len(nextManaged))
	for key, value := range current {
		merged[key] = value
	}

	if nextManaged != nil {
		// Remove keys that were in previous config but not in next (user removed them).
		for key := range previousManaged {
			if _, stillConfigured := nextManaged[key]; !stillConfigured {
				delete(merged, key)
			}
		}
		// Add/update configured keys.
		for key, value := range nextManaged {
			merged[key] = value
		}
	} else {
		// No managed labels at all — remove everything that was previously managed.
		for key := range previousManaged {
			delete(merged, key)
		}
	}

	if len(merged) == 0 {
		return nil
	}
	return merged
}

func mergeManagedAnnotations(
	current map[string]string,
	previousManaged map[string]string,
	nextManaged map[string]string,
	clearManagedAnnotations bool,
) map[string]string {
	nextUserAnnotations := stripManagerAnnotations(nextManaged)
	if clearManagedAnnotations && nextUserAnnotations == nil {
		nextUserAnnotations = map[string]string{}
	}

	merged := mergeManagedStringMap(
		stripManagerAnnotations(current),
		stripManagerAnnotations(previousManaged),
		nextUserAnnotations,
	)
	return mergeStringMaps(merged, preferredManagerAnnotations(current, nextManaged))
}

func preferredManagerAnnotations(current map[string]string, next map[string]string) map[string]string {
	if nextManagerAnnotations := managerAnnotations(next); len(nextManagerAnnotations) > 0 {
		return nextManagerAnnotations
	}
	return managerAnnotations(current)
}

func stripManagerAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}

	filtered := make(map[string]string)
	for key, value := range annotations {
		if isManagerAnnotation(key) {
			continue
		}
		filtered[key] = value
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func managerAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}

	filtered := make(map[string]string)
	for key, value := range annotations {
		if !isManagerAnnotation(key) {
			continue
		}
		filtered[key] = value
	}
	return filtered
}

func isManagerAnnotation(key string) bool {
	return key == utils.AnnoKeyManagerIdentity || key == utils.AnnoKeyManagerKind || key == utils.AnnoKeyManagerAllowsEdits
}

func mergeStringMaps(primary map[string]string, secondary map[string]string) map[string]string {
	if len(primary) == 0 && len(secondary) == 0 {
		return nil
	}

	merged := make(map[string]string, len(primary)+len(secondary))
	for key, value := range primary {
		merged[key] = value
	}
	for key, value := range secondary {
		merged[key] = value
	}
	return merged
}

func resourceUIDChanged(stateID types.String, current *genericUntypedObject) bool {
	if stateID.IsNull() || stateID.IsUnknown() {
		return false
	}

	expectedUID := strings.TrimSpace(stateID.ValueString())
	if expectedUID == "" {
		return false
	}

	currentUID := strings.TrimSpace(string(current.GetUID()))
	return currentUID != "" && currentUID != expectedUID
}

type genericSecureVersionValidator struct{}

func (v genericSecureVersionValidator) Description(context.Context) string {
	return "Requires `secure_version` when `secure` values are configured."
}

func (v genericSecureVersionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v genericSecureVersionValidator) ValidateResource(ctx context.Context, req tfrsc.ValidateConfigRequest, resp *tfrsc.ValidateConfigResponse) {
	var secure types.Dynamic
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("secure"), &secure)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var secureVersion types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("secure_version"), &secureVersion)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(
		validateGenericSecureVersionRequirement(
			secure,
			secureVersion,
			"Set `secure_version = 1` when using `secure`, then increment it whenever you want Terraform to re-apply secure values.",
		)...,
	)
}

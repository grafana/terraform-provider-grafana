package generic

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	apicommon "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	conflictRetryAttempts = 5
	conflictRetryDelay    = 200 * time.Millisecond
)

func retryOnConflict(
	ctx context.Context,
	attempts int,
	delay time.Duration,
	run func(attempt int) error,
) error {
	var err error

	for attempt := 0; attempt < attempts; attempt++ {
		err = run(attempt)
		if err == nil || !apierrors.IsConflict(err) || attempt == attempts-1 {
			return err
		}

		if err := waitForRetry(ctx, delay); err != nil {
			return err
		}
	}

	return err
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func secureSubresourcePayload(secure map[string]apicommon.InlineSecureValue) map[string]any {
	out := map[string]any{}
	for key, value := range secure {
		subresource := map[string]any{}
		if value.Create != "" {
			subresource["create"] = string(value.Create)
		}
		if value.Name != "" {
			subresource["name"] = value.Name
		}
		if len(subresource) == 0 {
			continue
		}
		out[key] = subresource
	}

	return out
}

func setManagerProperties(obj sdkresource.Object, clientID string, allowUIUpdates bool) error {
	meta, err := utils.MetaAccessor(obj)
	if err != nil {
		return fmt.Errorf("failed to configure resource metadata: %w", err)
	}

	desired := utils.ManagerProperties{
		Kind:        utils.ManagerKindTerraform,
		Identity:    clientID,
		AllowsEdits: allowUIUpdates,
	}

	existing, found := meta.GetManagerProperties()
	if !found || existing != desired {
		meta.SetManagerProperties(desired)
	}

	return nil
}

type resourceData interface {
	GetAttribute(ctx context.Context, path path.Path, target interface{}) diag.Diagnostics
}

func secureVersionChanged(current, previous types.Int64) bool {
	switch {
	case current.IsUnknown() || previous.IsUnknown():
		return true
	case current.IsNull() && previous.IsNull():
		return false
	case current.IsNull() != previous.IsNull():
		return true
	default:
		return current.ValueInt64() != previous.ValueInt64()
	}
}

func hasSecureValueInput(value attr.Value) bool {
	if value == nil || value.IsNull() {
		return false
	}

	if value.IsUnknown() {
		return true
	}

	attrs, err := secureFieldValues(value)
	if err != nil {
		return false
	}

	for _, nestedValue := range attrs {
		if nestedValue == nil || nestedValue.IsNull() {
			continue
		}

		return true
	}

	return false
}

func parseInlineSecureValue(fieldName string, fieldValue attr.Value) (apicommon.InlineSecureValue, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if fieldValue == nil || fieldValue.IsNull() || fieldValue.IsUnknown() {
		return apicommon.InlineSecureValue{}, false, diags
	}

	attrs, err := secureFieldValues(fieldValue)
	if err != nil {
		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf("secure field %q has unsupported type %T; expected map/object with `name` or `create`", fieldName, fieldValue),
		)

		return apicommon.InlineSecureValue{}, false, diags
	}

	for key := range attrs {
		if key == "name" || key == "create" {
			continue
		}

		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf("secure field %q object contains unsupported key %q; only `name` and `create` are allowed", fieldName, key),
		)
	}
	if diags.HasError() {
		return apicommon.InlineSecureValue{}, false, diags
	}

	nameValue := attrs["name"]
	createValue := attrs["create"]

	name, hasConfiguredName, nameDiags := configuredSecureStringValue(fieldName, "name", nameValue)
	diags.Append(nameDiags...)
	create, hasConfiguredCreate, createDiags := configuredSecureStringValue(fieldName, "create", createValue)
	diags.Append(createDiags...)
	if diags.HasError() {
		return apicommon.InlineSecureValue{}, false, diags
	}

	switch {
	case hasConfiguredName && hasConfiguredCreate:
		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf("secure field %q object must set exactly one of `name` or `create`", fieldName),
		)
		return apicommon.InlineSecureValue{}, false, diags
	case hasConfiguredName:
		if strings.TrimSpace(name) == "" {
			diags.AddError(
				"failed to parse secure values",
				fmt.Sprintf("secure field %q object `name` must not be empty", fieldName),
			)
			return apicommon.InlineSecureValue{}, false, diags
		}
		return apicommon.InlineSecureValue{Name: name}, true, diags
	case hasConfiguredCreate:
		if create == "" {
			diags.AddError(
				"failed to parse secure values",
				fmt.Sprintf("secure field %q object `create` must not be empty", fieldName),
			)
			return apicommon.InlineSecureValue{}, false, diags
		}
		return apicommon.InlineSecureValue{Create: apicommon.NewSecretValue(create)}, true, diags
	default:
		return apicommon.InlineSecureValue{}, false, diags
	}
}

func secureFieldValues(value attr.Value) (map[string]attr.Value, error) {
	switch v := value.(type) {
	case types.Object:
		return v.Attributes(), nil
	case types.Map:
		return v.Elements(), nil
	default:
		return nil, fmt.Errorf("unsupported secure field type %T", value)
	}
}

func configuredSecureStringValue(fieldName, attributeName string, value attr.Value) (string, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value == nil || value.IsNull() || value.IsUnknown() {
		return "", false, diags
	}

	stringValue, ok := value.(types.String)
	if !ok {
		diags.AddError(
			"failed to parse secure values",
			fmt.Sprintf(
				"secure field %q object `%s` has unsupported type %T; expected string",
				fieldName,
				attributeName,
				value,
			),
		)
		return "", false, diags
	}

	return stringValue.ValueString(), true, diags
}

// genericAllowUIUpdates returns the allow_ui_updates value from the model.
// Defaults to false when not configured — Terraform-managed resources should
// not be editable in the UI unless the user explicitly opts in.
func genericAllowUIUpdates(model GenericResourceModel) bool {
	if model.AllowUIUpdates.IsNull() || model.AllowUIUpdates.IsUnknown() {
		return false
	}
	return model.AllowUIUpdates.ValueBool()
}

// readAllowUIUpdatesFromObject reads the AllowsEdits manager property from a
// live object. Returns false (the default) if manager properties are not set.
func readAllowUIUpdatesFromObject(obj sdkresource.Object) bool {
	meta, err := utils.MetaAccessor(obj)
	if err != nil {
		return false
	}

	mgr, ok := meta.GetManagerProperties()
	if !ok {
		return false
	}

	return mgr.AllowsEdits
}

// genericManagerIdentity returns the manager_identity value from the model.
// Falls back to the provider-level default when not configured.
func genericManagerIdentity(model GenericResourceModel, defaultIdentity string) string {
	if model.ManagerIdentity.IsNull() || model.ManagerIdentity.IsUnknown() {
		return defaultIdentity
	}
	return model.ManagerIdentity.ValueString()
}

// readManagerIdentityFromObject reads the manager Identity from a live object.
// Returns empty string if manager properties are not set.
func readManagerIdentityFromObject(obj sdkresource.Object) string {
	meta, err := utils.MetaAccessor(obj)
	if err != nil {
		return ""
	}

	mgr, ok := meta.GetManagerProperties()
	if !ok {
		return ""
	}

	return mgr.Identity
}

package appplatform

import (
	"fmt"
	"reflect"

	apicommon "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
)

var inlineSecureValueReflectType = reflect.TypeOf(apicommon.InlineSecureValue{})

// secureSubresourceSupport provides sdkresource.Object subresource methods for
// resources that only expose the "secure" subresource.
//
// Example:
//
//	type MyResource struct {
//		secureSubresourceSupport[myv0alpha1.MySecure]
//		// ...
//	}
type secureSubresourceSupport[T any] struct {
	Secure T `json:"secure,omitempty"`
}

func (o *secureSubresourceSupport[T]) GetSubresources() map[string]any {
	return addSecureSubresource(nil, o.Secure)
}

func (o *secureSubresourceSupport[T]) GetSubresource(name string) (any, bool) {
	return getSecureSubresource(name, o.Secure)
}

func (o *secureSubresourceSupport[T]) SetSubresource(name string, value any) error {
	handled, err := setSecureSubresource(name, value, &o.Secure)
	if handled {
		return err
	}

	return fmt.Errorf("subresource '%s' does not exist", name)
}

// addSecureSubresource merges a secure subresource payload into an existing map.
// Use this for resources that expose secure plus additional subresources.
func addSecureSubresource(subresources map[string]any, secure any) map[string]any {
	securePayload := secureSubresourcePayload(secure)
	if len(securePayload) == 0 {
		if subresources == nil {
			return map[string]any{}
		}

		return subresources
	}

	if subresources == nil {
		subresources = map[string]any{}
	}

	subresources["secure"] = securePayload
	return subresources
}

func getSecureSubresource(name string, secure any) (any, bool) {
	if name != "secure" {
		return nil, false
	}

	return secure, true
}

// setSecureSubresource handles SetSubresource for "secure".
// It returns handled=false when name is not "secure".
func setSecureSubresource[T any](name string, value any, dst *T) (handled bool, err error) {
	if name != "secure" {
		return false, nil
	}

	cast, ok := value.(T)
	if !ok {
		return true, fmt.Errorf("cannot set secure type %#v, not of type %s", value, genericTypeName[T]())
	}

	*dst = cast
	return true, nil
}

func genericTypeName[T any]() string {
	typeName := reflect.TypeOf((*T)(nil)).Elem()
	if typeName.Name() != "" {
		return typeName.Name()
	}

	return typeName.String()
}

func secureSubresourcePayload(secure any) map[string]any {
	v := reflect.ValueOf(secure)
	if !v.IsValid() {
		return map[string]any{}
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return map[string]any{}
		}

		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return structSecureSubresourcePayload(v)
	case reflect.Map:
		return mapSecureSubresourcePayload(v)
	default:
		return map[string]any{}
	}
}

func structSecureSubresourcePayload(v reflect.Value) map[string]any {
	out := map[string]any{}
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		if fieldType.PkgPath != "" {
			continue
		}

		jsonName := jsonFieldName(fieldType)
		if jsonName == "-" {
			continue
		}

		inlineValue, ok := inlineSecureValueFromReflect(v.Field(i))
		if !ok {
			continue
		}

		subresource := inlineSecureValueSubresource(inlineValue)
		if len(subresource) == 0 {
			continue
		}

		out[jsonName] = subresource
	}

	return out
}

func mapSecureSubresourcePayload(v reflect.Value) map[string]any {
	if v.Type().Key().Kind() != reflect.String {
		return map[string]any{}
	}

	out := map[string]any{}
	it := v.MapRange()
	for it.Next() {
		inlineValue, ok := inlineSecureValueFromReflect(it.Value())
		if !ok {
			continue
		}

		subresource := inlineSecureValueSubresource(inlineValue)
		if len(subresource) == 0 {
			continue
		}

		out[it.Key().String()] = subresource
	}

	return out
}

func inlineSecureValueFromReflect(v reflect.Value) (apicommon.InlineSecureValue, bool) {
	if !v.IsValid() {
		return apicommon.InlineSecureValue{}, false
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return apicommon.InlineSecureValue{}, false
		}

		v = v.Elem()
	}

	if !v.CanInterface() {
		return apicommon.InlineSecureValue{}, false
	}

	if v.Type().AssignableTo(inlineSecureValueReflectType) {
		return v.Interface().(apicommon.InlineSecureValue), true
	}

	if v.Type().ConvertibleTo(inlineSecureValueReflectType) {
		return v.Convert(inlineSecureValueReflectType).Interface().(apicommon.InlineSecureValue), true
	}

	return apicommon.InlineSecureValue{}, false
}

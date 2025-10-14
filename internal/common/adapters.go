package common

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ListToStringSlice(src []any) []string {
	dst := make([]string, 0, len(src))
	for _, s := range src {
		val, ok := s.(string)
		if !ok {
			val = ""
		}
		dst = append(dst, val)
	}
	return dst
}

func SetToStringSlice(src *schema.Set) []string {
	return ListToStringSlice(src.List())
}

func ListToIntSlice[T int | int64](src []any) []T {
	dst := make([]T, 0, len(src))
	for _, s := range src {
		val, ok := s.(int)
		if !ok {
			val = 0
		}
		dst = append(dst, T(val))
	}
	return dst
}

func SetToIntSlice[T int | int64](src *schema.Set) []T {
	return ListToIntSlice[T](src.List())
}

func StringSliceToList(list []string) []any {
	vs := make([]any, 0, len(list))
	for _, v := range list {
		vs = append(vs, v)
	}
	return vs
}

func StringSliceToSet(src []string) *schema.Set {
	return schema.NewSet(schema.HashString, StringSliceToList(src))
}

func Int32SliceToIntList(list []int32) []any {
	vs := make([]any, 0, len(list))
	for _, v := range list {
		vs = append(vs, int(v))
	}
	return vs
}

func Int32SliceToSet(src []int32) *schema.Set {
	return schema.NewSet(schema.HashInt, Int32SliceToIntList(src))
}

func ListOfSetsToStringSlice(listSet []any) [][]string {
	ret := make([][]string, 0, len(listSet))
	for _, set := range listSet {
		ret = append(ret, SetToStringSlice(set.(*schema.Set)))
	}
	return ret
}

func UnpackMap[T any](m any) map[string]T {
	ret := make(map[string]T)
	for k, v := range m.(map[string]any) {
		ret[k] = v.(T)
	}
	return ret
}

func Ref[T any](v T) *T {
	return &v
}

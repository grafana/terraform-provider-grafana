package common

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ListToStringSlice(src []interface{}) []string {
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

func ListToIntSlice(src []interface{}) []int {
	dst := make([]int, 0, len(src))
	for _, s := range src {
		val, ok := s.(int)
		if !ok {
			val = 0
		}
		dst = append(dst, val)
	}
	return dst
}

func SetToIntSlice(src *schema.Set) []int {
	return ListToIntSlice(src.List())
}

func StringSliceToList(list []string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, v)
	}
	return vs
}

func StringSliceToSet(src []string) *schema.Set {
	return schema.NewSet(schema.HashString, StringSliceToList(src))
}

func Int32SliceToIntList(list []int32) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, int(v))
	}
	return vs
}

func Int32SliceToSet(src []int32) *schema.Set {
	return schema.NewSet(schema.HashInt, Int32SliceToIntList(src))
}

func ListOfSetsToStringSlice(listSet []interface{}) [][]string {
	ret := make([][]string, 0, len(listSet))
	for _, set := range listSet {
		ret = append(ret, SetToStringSlice(set.(*schema.Set)))
	}
	return ret
}

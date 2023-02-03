package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func listToStringSlice(src []interface{}) []string {
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

func setToStringSlice(src *schema.Set) []string {
	return listToStringSlice(src.List())
}

func listToIntSlice(src []interface{}) []int {
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

func setToIntSlice(src *schema.Set) []int {
	return listToIntSlice(src.List())
}

func stringSliceToList(list []string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, v)
	}
	return vs
}

func stringSliceToSet(src []string) *schema.Set {
	return schema.NewSet(schema.HashString, stringSliceToList(src))
}

func int32SliceToIntList(list []int32) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, int(v))
	}
	return vs
}

func int32SliceToSet(src []int32) *schema.Set {
	return schema.NewSet(schema.HashInt, int32SliceToIntList(src))
}

func listOfSetsToStringSlice(listSet []interface{}) [][]string {
	ret := make([][]string, 0, len(listSet))
	for _, set := range listSet {
		ret = append(ret, setToStringSlice(set.(*schema.Set)))
	}
	return ret
}

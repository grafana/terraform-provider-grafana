package grafana

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func listToStringSlice(src []interface{}) []string {
	dst := make([]string, 0, len(src))
	for _, s := range src {
		dst = append(dst, s.(string))
	}
	return dst
}

func setToStringSlice(src *schema.Set) []string {
	return listToStringSlice(src.List())
}

func listToIntSlice(src []interface{}) []int {
	dst := make([]int, 0, len(src))
	for _, s := range src {
		dst = append(dst, s.(int))
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

func listOfSetsToStringSlice(listSet []*schema.Set) [][]string {
	ret := [][]string{}

	if listSet == nil {
		return ret
	}
	for _, set := range listSet {
		ret = append(ret, setToStringSlice(set))
	}
	return ret
}

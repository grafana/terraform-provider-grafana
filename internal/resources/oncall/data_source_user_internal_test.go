package oncall

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestUnitUserModelMatchesSchemas guards the one mistake userDataSourceModel
// invites: it backs two schemas, and the framework fails every read of a data
// source whose schema and model disagree. Adding a field to the model, or an
// attribute to either schema, without doing the same to the other fails here
// rather than in a practitioner's plan.
func TestUnitUserModelMatchesSchemas(t *testing.T) {
	ctx := context.Background()

	modelAttrs := []string{}
	modelType := reflect.TypeOf(userDataSourceModel{})
	for i := range modelType.NumField() {
		tag, ok := modelType.Field(i).Tag.Lookup("tfsdk")
		if !ok {
			t.Fatalf("field %s has no tfsdk tag", modelType.Field(i).Name)
		}
		modelAttrs = append(modelAttrs, tag)
	}

	// The singular data source: the model is its entire state.
	singularResp := &datasource.SchemaResponse{}
	(&userDataSource{}).Schema(ctx, datasource.SchemaRequest{}, singularResp)
	singularAttrs := []string{}
	for name := range singularResp.Schema.Attributes {
		singularAttrs = append(singularAttrs, name)
	}

	// The plural data source: the model is one element of its users list.
	pluralResp := &datasource.SchemaResponse{}
	(&usersDataSource{}).Schema(ctx, datasource.SchemaRequest{}, pluralResp)
	usersAttr, ok := pluralResp.Schema.Attributes["users"]
	if !ok {
		t.Fatal("grafana_oncall_users has no users attribute")
	}
	elemType, ok := usersAttr.GetType().(types.ListType).ElemType.(types.ObjectType)
	if !ok {
		t.Fatalf("users element is %T, want an object", usersAttr.GetType())
	}
	elementAttrs := []string{}
	for name := range elemType.AttrTypes {
		elementAttrs = append(elementAttrs, name)
	}

	sort.Strings(modelAttrs)
	sort.Strings(singularAttrs)
	sort.Strings(elementAttrs)

	if !reflect.DeepEqual(modelAttrs, singularAttrs) {
		t.Errorf("grafana_oncall_user schema has %v, model has %v", singularAttrs, modelAttrs)
	}
	if !reflect.DeepEqual(modelAttrs, elementAttrs) {
		t.Errorf("grafana_oncall_users element has %v, model has %v", elementAttrs, modelAttrs)
	}

	// Every attribute the model carries must also agree on type, or state
	// conversion fails even though the names line up.
	for name, elemAttrType := range elemType.AttrTypes {
		singularAttr, ok := singularResp.Schema.Attributes[name]
		if !ok {
			continue // reported above
		}
		if got := singularAttr.GetType(); !got.Equal(elemAttrType) {
			t.Errorf("attribute %s is %s in grafana_oncall_user and %s in the users element",
				name, got, elemAttrType)
		}
	}
}

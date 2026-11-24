package appplatform

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func newTestReceiverDefinition(t *testing.T) *receiverDefinition {
	t.Helper()
	def, _, err := newReceiverDefinition()
	require.NoError(t, err)
	return def
}

// receiverTestObject builds an object of typ from the given values; attributes not given are null.
func receiverTestObject(typ types.ObjectType, values map[string]attr.Value) types.Object {
	all := map[string]attr.Value{}
	for name, t := range typ.AttrTypes {
		all[name] = nullOf(t)
	}
	for k, v := range values {
		all[k] = v
	}
	return types.ObjectValueMust(typ.AttrTypes, all)
}

func receiverTestSpec(def *receiverDefinition, title string, disableProvenance bool, blocks map[string][]attr.Value) types.Object {
	values := map[string]attr.Value{
		receiverTitleAttr:             types.StringValue(title),
		receiverDisableProvenanceAttr: types.BoolValue(disableProvenance),
	}
	for _, it := range def.types {
		listType := def.specType.AttrTypes[it.TFName].(types.ListType)
		values[it.TFName] = types.ListValueMust(listType.ElemType, blocks[it.TFName])
	}
	return types.ObjectValueMust(def.specType.AttrTypes, values)
}

func TestReceiverParseAndSaveRoundTrip(t *testing.T) {
	def := newTestReceiverDefinition(t)
	ctx := context.Background()

	slackType := def.elemType(def.byTypeVersion[receiverIntegrationKey{"slack", "v1"}])
	emailType := def.elemType(def.byTypeVersion[receiverIntegrationKey{"email", "v1"}])
	webhookType := def.elemType(def.byTypeVersion[receiverIntegrationKey{"webhook", "v1"}])
	oauth2Type := webhookType.AttrTypes["http_config"].(types.ObjectType).AttrTypes["oauth2"].(types.ObjectType)
	httpConfigType := webhookType.AttrTypes["http_config"].(types.ObjectType)

	slack := receiverTestObject(slackType, map[string]attr.Value{
		"uid":                     types.StringValue("slack-primary"),
		"disable_resolve_message": types.BoolValue(true),
		"url":                     types.StringValue("https://hooks.example.com/secret"),
		"title":                   types.StringValue("Primary"),
		"mention_channel":         types.StringValue("here"),
	})
	email := receiverTestObject(emailType, map[string]attr.Value{
		"uid":       types.StringValue("email-ops"),
		"addresses": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a@example.com"), types.StringValue("b@example.com")}),
	})
	webhook := receiverTestObject(webhookType, map[string]attr.Value{
		"uid":        types.StringValue("webhook-1"),
		"url":        types.StringValue("https://example.com/hook"),
		"max_alerts": types.Int64Value(5),
		"headers":    types.MapValueMust(types.StringType, map[string]attr.Value{"X-Team": types.StringValue("ops")}),
		"http_config": receiverTestObject(httpConfigType, map[string]attr.Value{
			"oauth2": receiverTestObject(oauth2Type, map[string]attr.Value{
				"client_id":     types.StringValue("id"),
				"client_secret": types.StringValue("shh"),
				"token_url":     types.StringValue("https://example.com/token"),
				"scopes":        types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")}),
			}),
		}),
	})

	spec := receiverTestSpec(def, "Operations", false, map[string][]attr.Value{
		"slack_v1":   {slack},
		"email_v1":   {email},
		"webhook_v1": {webhook},
	})

	// Terraform -> API
	var recv v1beta1.Receiver
	diags := def.parseSpec(ctx, spec, &recv)
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, "Operations", recv.Spec.Title)
	require.Len(t, recv.Spec.Integrations, 3)

	byUID := map[string]v1beta1.ReceiverIntegration{}
	for _, in := range recv.Spec.Integrations {
		require.NotNil(t, in.Uid)
		byUID[*in.Uid] = in
	}

	s := byUID["slack-primary"]
	require.Equal(t, "slack", s.Type)
	require.Equal(t, "v1", s.Version)
	require.True(t, *s.DisableResolveMessage)
	require.Equal(t, "https://hooks.example.com/secret", s.Settings["url"], "secrets are sent from the configuration")
	require.Equal(t, "here", s.Settings["mentionChannel"])
	require.NotContains(t, s.Settings, "token", "unset fields are omitted, which clears them")
	require.Empty(t, s.SecureFields)

	require.Equal(t, "a@example.com;b@example.com", byUID["email-ops"].Settings["addresses"])

	w := byUID["webhook-1"]
	require.Equal(t, int64(5), w.Settings["maxAlerts"])
	require.Equal(t, map[string]string{"X-Team": "ops"}, w.Settings["headers"])
	oauth := w.Settings["http_config"].(map[string]any)["oauth2"].(map[string]any)
	require.Equal(t, "shh", oauth["client_secret"])
	require.Equal(t, []string{"a"}, oauth["scopes"])
	require.Equal(t, "id", oauth["client_id"])

	// API -> Terraform, as on import: the API returns no secret values.
	for _, uid := range []string{"slack-primary", "webhook-1"} {
		in := byUID[uid]
		delete(in.Settings, "url")
		if uid == "webhook-1" {
			in.Settings["http_config"].(map[string]any)["oauth2"].(map[string]any)["client_secret"] = nil
			delete(in.Settings["http_config"].(map[string]any)["oauth2"].(map[string]any), "client_secret")
			in.Settings["url"] = "https://example.com/hook"
		}
		byUID[uid] = in
	}
	imported := v1beta1.Receiver{Spec: v1beta1.ReceiverSpec{Title: "Operations"}}
	for _, uid := range []string{"slack-primary", "email-ops", "webhook-1"} {
		imported.Spec.Integrations = append(imported.Spec.Integrations, byUID[uid])
	}
	// The API response is JSON, so settings come back as map[string]any, []any and float64.
	wire, err := json.Marshal(imported.Spec)
	require.NoError(t, err)
	imported.Spec = v1beta1.ReceiverSpec{}
	require.NoError(t, json.Unmarshal(wire, &imported.Spec))
	imported.SetProvenanceStatus("")

	var model ResourceModel
	diags = def.saveSpec(ctx, &imported, &model)
	require.False(t, diags.HasError(), "%v", diags)

	attrs := model.Spec.Attributes()
	require.Equal(t, types.BoolValue(true), attrs[receiverDisableProvenanceAttr], "no provenance on the server means disable_provenance")

	gotSlack := attrs["slack_v1"].(types.List).Elements()[0].(types.Object).Attributes()
	require.Equal(t, types.StringValue("slack-primary"), gotSlack["uid"])
	require.Equal(t, types.BoolValue(true), gotSlack["disable_resolve_message"])
	require.True(t, gotSlack["url"].IsNull(), "credentials are not recovered by import")
	require.Equal(t, types.StringValue("here"), gotSlack["mention_channel"])

	gotEmail := attrs["email_v1"].(types.List).Elements()[0].(types.Object).Attributes()
	require.Equal(t, spec.Attributes()["email_v1"].(types.List).Elements()[0].(types.Object).Attributes()["addresses"], gotEmail["addresses"])

	gotWebhook := attrs["webhook_v1"].(types.List).Elements()[0].(types.Object).Attributes()
	require.Equal(t, types.Int64Value(5), gotWebhook["max_alerts"])
	require.Equal(t, types.StringValue("id"),
		gotWebhook["http_config"].(types.Object).Attributes()["oauth2"].(types.Object).Attributes()["client_id"])
	require.True(t, gotWebhook["http_config"].(types.Object).Attributes()["oauth2"].(types.Object).Attributes()["client_secret"].IsNull())
}

func TestReceiverParseProvenance(t *testing.T) {
	def := newTestReceiverDefinition(t)
	for disable, want := range map[bool]string{false: provenanceAPI, true: provenanceNone} {
		var recv v1beta1.Receiver
		diags := def.parseSpec(context.Background(), receiverTestSpec(def, "r", disable, nil), &recv)
		require.False(t, diags.HasError(), "%v", diags)
		require.Equal(t, want, recv.Annotations[v1beta1.ProvenanceStatusAnnotationKey])
	}
}

func TestReceiverSaveRejectsUnsupportedIntegrations(t *testing.T) {
	def := newTestReceiverDefinition(t)
	uid := "x"
	for name, in := range map[string]v1beta1.ReceiverIntegration{
		"unknown type":        {Uid: &uid, Type: "carrier-pigeon", Version: "v1", Settings: map[string]any{}},
		"non-creatable":       {Uid: &uid, Type: "slack", Version: "v0mimir1", Settings: map[string]any{}},
		"unknown setting key": {Uid: &uid, Type: "slack", Version: "v1", Settings: map[string]any{"nope": "x"}},
		"unknown version":     {Uid: &uid, Type: "slack", Version: "v2", Settings: map[string]any{}},
	} {
		t.Run(name, func(t *testing.T) {
			recv := v1beta1.Receiver{Spec: v1beta1.ReceiverSpec{Title: "r", Integrations: []v1beta1.ReceiverIntegration{in}}}
			var model ResourceModel
			diags := def.saveSpec(context.Background(), &recv, &model)
			require.True(t, diags.HasError(), "unsupported content must fail, not be dropped")
		})
	}
}

// Grafana accepts numbers as strings, so a number saved that way (for example in the UI) is read back
// as a string.
// Strings the API stores as JSON numbers are read back without an exponent.
func TestReceiverStringFromNumber(t *testing.T) {
	f := &integrationschemas.Field{APIName: "message_thread_id", Kind: integrationschemas.KindString}
	for raw, want := range map[float64]string{5: "5", 1000000: "1000000", 123456789: "123456789", 1.5: "1.5"} {
		v, diags := receiverValueFromAPI(f, types.StringType, raw)
		require.False(t, diags.HasError())
		require.Equal(t, types.StringValue(want), v)
	}
}

func TestReceiverIntFromAPI(t *testing.T) {
	f := &integrationschemas.Field{APIName: "retry", Kind: integrationschemas.KindInt}

	for _, raw := range []any{float64(30), int64(30), 30, "30", " 30 "} {
		v, diags := receiverValueFromAPI(f, types.Int64Type, raw)
		require.False(t, diags.HasError(), "%v", raw)
		require.Equal(t, types.Int64Value(30), v, "%v", raw)
	}

	_, diags := receiverValueFromAPI(f, types.Int64Type, "soon")
	require.True(t, diags.HasError())
}

// A setting that is stored as an explicit null is the same as one that is not set.
func TestReceiverSaveIgnoresNullSettings(t *testing.T) {
	def := newTestReceiverDefinition(t)
	uid := "x"
	in := v1beta1.ReceiverIntegration{Uid: &uid, Type: "slack", Version: "v1", Settings: map[string]any{"title": nil, "recipient": nil}}
	recv := v1beta1.Receiver{Spec: v1beta1.ReceiverSpec{Title: "r", Integrations: []v1beta1.ReceiverIntegration{in}}}
	var model ResourceModel
	diags := def.saveSpec(context.Background(), &recv, &model)
	require.False(t, diags.HasError(), "%v", diags)
}

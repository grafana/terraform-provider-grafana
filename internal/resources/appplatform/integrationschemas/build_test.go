package integrationschemas

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTFNameOf(t *testing.T) {
	for in, want := range map[string]string{
		"url":                     "url",
		"basicAuthUser":           "basic_auth_user",
		"http_config":             "http_config",
		"tlsConfig":               "tls_config",
		"insecureSkipVerify":      "insecure_skip_verify",
		"apiURL":                  "api_url",
		"apiURLValue":             "api_url_value",
		"sigv4":                   "sigv4",
		"prometheus-alertmanager": "prometheus_alertmanager",
		"sensugo":                 "sensugo",
		"disableResolveMessage":   "disable_resolve_message",
	} {
		require.Equal(t, want, tfNameOf(in), in)
	}
}

func mustLoad(t *testing.T) map[string]IntegrationType {
	t.Helper()
	types_, err := Load()
	require.NoError(t, err)
	out := map[string]IntegrationType{}
	for _, it := range types_ {
		out[it.APIType] = it
		require.Equal(t, "v1", it.Version, it.APIType)
	}
	return out
}

func field(t *testing.T, it IntegrationType, apiName string) *Field {
	t.Helper()
	for _, f := range it.Fields {
		if f.APIName == apiName {
			return f
		}
	}
	t.Fatalf("%s has no field %q", it.APIType, apiName)
	return nil
}

func TestLoad(t *testing.T) {
	byAPI := mustLoad(t)

	// Types that only have non-creatable versions are not exposed; creatable ones are.
	require.NotContains(t, byAPI, "wechat")
	require.NotContains(t, byAPI, "LINE")
	require.Contains(t, byAPI, "slack")
	require.Equal(t, "alertmanager_v1", byAPI["prometheus-alertmanager"].TFName)
	require.Equal(t, "slack_v1", byAPI["slack"].TFName)
}

func TestFieldMapping(t *testing.T) {
	slack := mustLoad(t)["slack"]

	// Fields Grafana encrypts are sensitive.
	require.True(t, field(t, slack, "token").Sensitive)
	require.True(t, field(t, slack, "url").Sensitive)
	// "required" is not unconditional when it depends on another field (recipient depends on url).
	require.False(t, field(t, slack, "recipient").Required)
	// Select values come from the snapshot.
	mention := field(t, slack, "mentionChannel")
	require.Equal(t, []string{"", "here", "channel"}, mention.AllowedValues)
	require.Equal(t, "mention_channel", mention.TFName)
	require.Equal(t, KindString, mention.Kind)
}

func TestNestedFields(t *testing.T) {
	webhook := mustLoad(t)["webhook"]

	var find func(fields []*Field, path string) *Field
	find = func(fields []*Field, path string) *Field {
		for _, f := range fields {
			if joinPath(f.Path) == path {
				return f
			}
			if got := find(f.Children, path); got != nil {
				return got
			}
		}
		return nil
	}

	secret := find(webhook.Fields, "http_config.oauth2.client_secret")
	require.NotNil(t, secret)
	require.True(t, secret.Sensitive)
	require.Equal(t, []string{"http_config", "oauth2", "client_secret"}, secret.Path)
	require.Equal(t, KindObject, field(t, webhook, "http_config").Kind)
}

func joinPath(p []string) string {
	out := ""
	for i, s := range p {
		if i > 0 {
			out += "."
		}
		out += s
	}
	return out
}

func TestOverrides(t *testing.T) {
	byAPI := mustLoad(t)

	require.Equal(t, KindInt, field(t, byAPI["webhook"], "maxAlerts").Kind)
	require.Equal(t, KindInt, field(t, byAPI["oncall"], "maxAlerts").Kind)
	require.Equal(t, KindJoinedStringList, field(t, byAPI["email"], "addresses").Kind)

	// Pushover numbers are text and select inputs in the snapshot, and numbers here. The allowed
	// values of the selects are kept.
	for _, name := range []string{"priority", "okPriority", "retry", "expire"} {
		require.Equal(t, KindInt, field(t, byAPI["pushover"], name).Kind, name)
	}
	require.Equal(t, []string{"2", "1", "0", "-1", "-2"}, field(t, byAPI["pushover"], "priority").AllowedValues)

	// Defaults Grafana applies are mentioned in the description.
	for _, c := range [][2]string{{"opsgenie", "autoClose"}, {"opsgenie", "overridePriority"}, {"pushover", "uploadImage"}} {
		require.True(t, strings.HasSuffix(field(t, byAPI[c[0]], c[1]).Description, " Defaults to `true` when not set."), c[0]+" "+c[1])
	}

	// Selects Grafana does not enforce are not restricted, and keep their options in the description.
	for _, c := range [][2]string{{"dingding", "msgType"}, {"victorops", "messageType"}, {"pushover", "sound"}, {"pushover", "okSound"}} {
		f := field(t, byAPI[c[0]], c[1])
		require.Empty(t, f.AllowedValues, c[0]+" "+c[1])
		require.Contains(t, f.Description, "Usual values are `", c[0]+" "+c[1])
	}
	// Telegram's parse_mode is checked without regard to case, as Grafana does.
	parseMode := field(t, byAPI["telegram"], "parse_mode")
	require.Equal(t, []string{"None", "HTML", "Markdown", "MarkdownV2"}, parseMode.AllowedValues)
	require.True(t, parseMode.CaseInsensitive)
	require.Contains(t, parseMode.Description, "Default is 'HTML'. Grafana ignores the case of the value.")
	// Selects Grafana does enforce keep their allowed values.
	require.Equal(t, []string{"v2", "v3"}, field(t, byAPI["kafka"], "apiVersion").AllowedValues)

	// Fields grafana_contact_point marks sensitive stay sensitive, although Grafana does not encrypt them.
	for typ, name := range map[string]string{"teams": "url", "kafka": "kafkaRestProxy"} {
		require.True(t, field(t, byAPI[typ], name).Sensitive, typ)
	}
	require.True(t, field(t, byAPI["slack"], "token").Sensitive)
	require.False(t, field(t, byAPI["slack"], "recipient").Sensitive)

	// Terraform names follow grafana_contact_point, while the Grafana key is unchanged.
	msgType := field(t, byAPI["dingding"], "msgType")
	require.Equal(t, "message_type", msgType.TFName)
	require.Equal(t, []string{"msgType"}, msgType.Path)
	require.Equal(t, "chat_id", field(t, byAPI["telegram"], "chatid").TFName)
	require.Equal(t, "basic_auth_user", field(t, byAPI["webhook"], "username").TFName)

	// opsgenie apiUrl is required in the snapshot but optional here.
	apiURL := field(t, byAPI["opsgenie"], "apiUrl")
	require.Equal(t, "url", apiURL.TFName)
	require.False(t, apiURL.Required)
}

func TestBuildRejectsOverridesThatMatchNothing(t *testing.T) {
	fieldOverrides["slack/v1/doesNotExist"] = fieldOverride{tfName: "x"}
	defer delete(fieldOverrides, "slack/v1/doesNotExist")

	snap, err := ParseSnapshot(snapshotJSON)
	require.NoError(t, err)
	_, err = build(snap)
	require.ErrorContains(t, err, "does not match a field")
}

func TestAppendSentence(t *testing.T) {
	require.Equal(t, "A. B.", appendSentence("A", "B."))
	require.Equal(t, "A. B.", appendSentence("A.", "B."))
	require.Equal(t, "B.", appendSentence("", "B."))
}

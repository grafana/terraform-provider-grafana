package integrationschemas

// valueOverride changes how a field is represented in Terraform when the snapshot's
// element kind does not capture how the API stores it.
type valueOverride int

// The zero value means no override.
const (
	_ valueOverride = iota
	// overrideInt: a Terraform number, stored as a JSON number in the API.
	overrideInt
	// overrideJoinedStringList: a Terraform list of strings, stored as one string joined with ";".
	overrideJoinedStringList
)

// fieldOverride is a Terraform-specific adjustment of one field.
type fieldOverride struct {
	// tfName replaces the generated Terraform name when set.
	tfName string
	value  valueOverride
	// optional makes the field optional even though the snapshot marks it required.
	optional bool
	// sensitive hides the field in Terraform output although Grafana does not encrypt it.
	sensitive bool
	// freeform accepts values other than the options the snapshot lists, because Grafana does not
	// restrict the field (for example it is a template). The options are mentioned in the description.
	freeform bool
	// appendDescription is added to the end of the field's description.
	appendDescription string
}

// fieldOverrides are keyed by "<integration type>/<version>/<dot-separated API field path>".
// Every key must match a field in the snapshot. The settings types follow grafana_contact_point,
// which shares Grafana's parsing of the same integration settings (see
// internal/resources/grafana/resource_alerting_contact_point_notifiers.go).
//
// opsgenie apiUrl is marked required in the snapshot although it has a default (and is optional in
// grafana_contact_point). That is a mistake in grafana/alerting; remove the Optional override once
// the snapshot is fixed.
//
// Sensitivity follows the snapshot's "secure" flag, which is what Grafana encrypts. The two fields
// grafana_contact_point marks sensitive that the snapshot does not (teams url, kafka kafkaRestProxy)
// are overridden to stay sensitive, in line with grafana_contact_point.
//
// The select options of dingding message_type, victorops message_type and pushover sound and
// ok_sound are not enforced by Grafana, and the values are templated or can be custom (a Pushover
// account can have its own sounds), so they are only suggested.
//
// Pushover priority, ok_priority, retry and expire are numbers in grafana_contact_point and are
// accepted as numbers by Grafana, although the snapshot describes them as text and select inputs.
var fieldOverrides = map[string]fieldOverride{
	"oncall/v1/maxAlerts":  {value: overrideInt},
	"webhook/v1/maxAlerts": {value: overrideInt},
	"email/v1/addresses":   {value: overrideJoinedStringList},

	"pushover/v1/priority":   {value: overrideInt},
	"pushover/v1/okPriority": {value: overrideInt},

	"dingding/v1/msgType":      {tfName: "message_type", freeform: true},
	"victorops/v1/messageType": {freeform: true},
	"pushover/v1/sound":        {freeform: true},
	"pushover/v1/okSound":      {freeform: true},
	"pushover/v1/retry":        {value: overrideInt},
	"pushover/v1/expire":       {value: overrideInt},

	// Terraform names kept from grafana_contact_point, so that migrating only renames the block.
	// The Grafana settings key (the last segment of the override key) is unchanged.
	"kafka/v1/kafkaTopic":     {tfName: "topic"},
	"kafka/v1/kafkaRestProxy": {tfName: "rest_proxy_url", sensitive: true},
	"kafka/v1/kafkaClusterId": {tfName: "cluster_id"},
	"oncall/v1/username":      {tfName: "basic_auth_user"},
	"oncall/v1/password":      {tfName: "basic_auth_password"},
	"opsgenie/v1/apiUrl":      {tfName: "url", optional: true},

	// Grafana treats these as true when they are not set. grafana_contact_point saved false.
	"opsgenie/v1/autoClose":        {appendDescription: "Defaults to `true` when not set."},
	"opsgenie/v1/overridePriority": {appendDescription: "Defaults to `true` when not set."},
	"pushover/v1/uploadImage":      {appendDescription: "Defaults to `true` when not set."},
	"sensugo/v1/apikey":            {tfName: "api_key"},
	"teams/v1/sectiontitle":        {tfName: "section_title"},
	"teams/v1/url":                 {sensitive: true},
	"telegram/v1/bottoken":         {tfName: "token"},
	"telegram/v1/chatid":           {tfName: "chat_id"},
	"webex/v1/bot_token":           {tfName: "token"},
	"webhook/v1/username":          {tfName: "basic_auth_user"},
	"webhook/v1/password":          {tfName: "basic_auth_password"},
	"wecom/v1/msgtype":             {tfName: "msg_type"},
	"wecom/v1/touser":              {tfName: "to_user"},
}

// typeNameOverrides maps an API integration type to the Terraform block name, where it is not the
// snake_case of the type. The names follow grafana_contact_point, so that migrating only renames the
// resource.
var typeNameOverrides = map[string]string{
	"prometheus-alertmanager": "alertmanager",
}

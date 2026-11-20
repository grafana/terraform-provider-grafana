package integrationschemas

// IntegrationType is an integration (Slack, email, webhook and so on) that a Terraform resource exposes.
//
// The snapshot file holds the fields that are tagged for JSON. Load fills in the others, which are
// Terraform-specific (see overrides.go).
type IntegrationType struct {
	// APIType is the integration type in the API.
	APIType string `json:"type"`
	// Name is the display name of the integration.
	Name string `json:"name"`
	// Version is the only settings version new integrations can use.
	Version string   `json:"version"`
	Fields  []*Field `json:"fields"`

	// TFName is the name to use in Terraform: the type and the version, such as slack_v1.
	TFName string `json:"-"`
}

// FieldKind is how a settings field is represented in Terraform, and how it is stored in the API.
type FieldKind int

const (
	// KindString is a string, stored as a string. Allowed values, if any, are in Field.AllowedValues.
	KindString FieldKind = iota
	// KindBool is a bool, stored as a bool.
	KindBool
	// KindInt is a number, stored as a JSON number.
	KindInt
	// KindStringList is a list of strings, stored as a list of strings.
	KindStringList
	// KindStringMap is a map of strings, stored as a map of strings.
	KindStringMap
	// KindJoinedStringList is a list of strings, stored as one string joined with ";".
	KindJoinedStringList
	// KindObject is a nested object, stored as an object. Its fields are in Field.Children.
	KindObject
	// KindObjectList is a list of nested objects, stored as a list of objects.
	KindObjectList
)

// Field is a settings field of an integration.
//
// The snapshot file holds the fields that are tagged for JSON. Load fills in the others, which are
// Terraform-specific (see overrides.go), and may change Required, Description and AllowedValues.
type Field struct {
	// APIName is the key in the integration settings.
	APIName string `json:"name"`
	// Element is the UI element kind of the field in the API, which says how it is stored.
	Element string `json:"element"`
	// Description is the API's description, or its label when it has no description.
	Description string `json:"description,omitempty"`
	// Required is whether the field must always be set. Fields that are only required in some
	// circumstances, because they depend on another field, are not Required.
	Required bool `json:"required,omitempty"`
	// Secure is whether Grafana encrypts the value.
	Secure bool `json:"secure,omitempty"`
	// AllowedValues are the values of a select field.
	AllowedValues []string `json:"options,omitempty"`
	// Children are the fields of an object or a list of objects.
	Children []*Field `json:"fields,omitempty"`

	// TFName is the name to use in Terraform.
	TFName string `json:"-"`
	// Path is the API path from the integration settings root, including APIName.
	Path []string  `json:"-"`
	Kind FieldKind `json:"-"`
	// Sensitive is whether Terraform hides the value: Grafana encrypts it, or an override marks it.
	Sensitive bool `json:"-"`
}

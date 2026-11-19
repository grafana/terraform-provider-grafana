package integrationschemas

// IntegrationType is an integration (Slack, email, webhook and so on) that a Terraform resource exposes.
type IntegrationType struct {
	// APIType is the integration type in the API.
	APIType string `json:"type"`
	// Name is the display name of the integration.
	Name string `json:"name"`
	// Version is the only settings version new integrations can use.
	Version string   `json:"version"`
	Fields  []*Field `json:"fields"`
}

// Field is a settings field of an integration.
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
}

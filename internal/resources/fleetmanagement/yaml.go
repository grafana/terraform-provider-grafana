package fleetmanagement

import (
	"bytes"

	"github.com/knadh/koanf/v2"
	"go.yaml.in/yaml/v3"
)

// YAML implements a koanf.Parser that parses YAML bytes as conf maps.
type YAML struct{}

// Ensure YAML implements koanf.Parser
var _ koanf.Parser = (*YAML)(nil)

// Parser returns a YAML parser.
func Parser() *YAML {
	return &YAML{}
}

// Marshal marshals the given config map to YAML bytes.
func (*YAML) Marshal(o map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	defer encoder.Close()

	if err := encoder.Encode(o); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal parses the given YAML bytes.
func (*YAML) Unmarshal(b []byte) (map[string]any, error) {
	var out map[string]any
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out, nil
}

package oncall

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cty/cty"
)

// Label name rules, as the OnCall API applies them when labels are written to an
// integration. A key must start and end with a letter. A value must start with a
// letter and may end with a letter or a digit.
const labelNameMaxLength = 63

var (
	labelKeyNameRegexp   = regexp.MustCompile(`^[A-Za-z]([A-Za-z0-9_]*[A-Za-z])?$`)
	labelValueNameRegexp = regexp.MustCompile(`^[A-Za-z]([A-Za-z0-9_.-]*[A-Za-z0-9])?$`)
)

func validateLabelKeyName(key string) error {
	if len(key) > labelNameMaxLength || !labelKeyNameRegexp.MatchString(key) {
		return fmt.Errorf("must be 1-%d characters, can only contain alphanumeric characters or underscores, and must start and end with a letter", labelNameMaxLength)
	}

	return nil
}

func validateLabelValueName(value string) error {
	if len(value) > labelNameMaxLength || !labelValueNameRegexp.MatchString(value) {
		return fmt.Errorf("must be 1-%d characters, can only contain alphanumeric characters, hyphens, underscores and periods, must start with a letter and must end with a letter or digit", labelNameMaxLength)
	}

	return nil
}

// validateIntegrationLabelsConfig checks the label names that an apply would
// send. The value of a dynamic label is a Jinja template which the API does not
// validate, so only its key is checked.
//
// Every violation is reported in one error, so a single plan tells the user
// about all of them.
func validateIntegrationLabelsConfig(rawConfig cty.Value) error {
	problems := labelNameProblems(rawConfig, "labels", true)
	problems = append(problems, labelNameProblems(rawConfig, "dynamic_labels", false)...)

	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("invalid label names:\n  %s", strings.Join(problems, "\n  "))
}

// labelNameProblems describes every invalid name in one label attribute, in
// config order. Names that are not known yet are skipped, Terraform validates
// again once they are resolved.
func labelNameProblems(rawConfig cty.Value, attr string, validateValues bool) []string {
	if !labelsSetInConfig(rawConfig, attr) {
		return nil
	}

	labels := rawConfig.GetAttr(attr)
	if !labels.IsKnown() || labels.LengthInt() == 0 {
		return nil
	}

	var problems []string
	for i, label := range labels.AsValueSlice() {
		if label.IsNull() || !label.IsKnown() {
			continue
		}

		entries := label.AsValueMap()

		if key, ok := knownLabelName(entries, "key"); ok {
			if err := validateLabelKeyName(key); err != nil {
				problems = append(problems, fmt.Sprintf("%s[%d].key %q: %s", attr, i, key, err))
			}
		}

		if !validateValues {
			continue
		}

		if value, ok := knownLabelName(entries, "value"); ok {
			if err := validateLabelValueName(value); err != nil {
				problems = append(problems, fmt.Sprintf("%s[%d].value %q: %s", attr, i, value, err))
			}
		}
	}

	return problems
}

func knownLabelName(entries map[string]cty.Value, name string) (string, bool) {
	value, ok := entries[name]
	if !ok || value.IsNull() || !value.IsKnown() || value.Type() != cty.String {
		return "", false
	}

	return value.AsString(), true
}

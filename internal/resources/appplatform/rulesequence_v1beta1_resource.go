package appplatform

import (
	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// RuleSequenceV1Beta1 creates a new Grafana Rule Sequence resource backed by the v1beta1 API.
// The schema matches the v0alpha1 resource, so the v0alpha1 parser and saver are reused through spec conversion.
func RuleSequenceV1Beta1() NamedResource {
	spec := ruleSequenceSpecSchema()
	spec.Description = "Manages Grafana Rule Sequences."
	spec.MarkdownDescription = `
Manages Grafana Rule Sequences.

A rule sequence groups existing alert rules and recording rules so they are evaluated in order, at a shared interval, within a single folder. All referenced rules must live in the same folder as the sequence (set ` + "`metadata.folder_uid`" + `).

This resource uses the ` + "`rules.alerting.grafana.app/v1beta1`" + ` API, which requires Grafana 13.3 or later.
`

	return NewNamedResource[*v1beta1.RuleSequence, *v1beta1.RuleSequenceList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.RuleSequence]{
			Kind:   v1beta1.RuleSequenceKind(),
			Schema: spec,
			SpecParser: specConvertingParser[*v0alpha1.RuleSequence, *v1beta1.RuleSequence, v1beta1.RuleSequenceSpec](
				func() *v0alpha1.RuleSequence { return &v0alpha1.RuleSequence{} },
				parseRuleSequenceSpec,
			),
			SpecSaver: specConvertingSaver[*v1beta1.RuleSequence, *v0alpha1.RuleSequence, v0alpha1.RuleSequenceSpec](
				func() *v0alpha1.RuleSequence { return &v0alpha1.RuleSequence{} },
				saveRuleSequenceSpec,
			),
		})
}

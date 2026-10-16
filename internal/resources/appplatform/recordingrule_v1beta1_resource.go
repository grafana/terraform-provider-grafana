package appplatform

import (
	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// RecordingRuleV1Beta1 creates a new Grafana Recording Rule resource backed by the v1beta1 API.
// The schema matches the v0alpha1 resource, so the v0alpha1 parser and saver are reused through spec conversion.
func RecordingRuleV1Beta1() NamedResource {
	spec := recordingRuleSpecSchema()
	spec.Description = "Manages Grafana Recording Rules."
	spec.MarkdownDescription = `
Manages Grafana Recording Rules.

This resource uses the ` + "`rules.alerting.grafana.app/v1beta1`" + ` API, which requires Grafana 13.3 or later.
`

	return NewNamedResource[*v1beta1.RecordingRule, *v1beta1.RecordingRuleList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.RecordingRule]{
			Kind:   v1beta1.RecordingRuleKind(),
			Schema: spec,
			SpecParser: specConvertingParser[*v0alpha1.RecordingRule, *v1beta1.RecordingRule, v1beta1.RecordingRuleSpec](
				func() *v0alpha1.RecordingRule { return &v0alpha1.RecordingRule{} },
				parseRecordingRuleSpec,
			),
			SpecSaver: specConvertingSaver[*v1beta1.RecordingRule, *v0alpha1.RecordingRule, v0alpha1.RecordingRuleSpec](
				func() *v0alpha1.RecordingRule { return &v0alpha1.RecordingRule{} },
				saveRecordingRuleSpec,
			),
		})
}

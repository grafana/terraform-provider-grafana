package cloud

import (
	"fmt"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

const (
	managedStrategyAssigned     = "assigned"
	managedStrategyMajorAligned = "major-aligned"
)

type managedPluginStatus struct {
	Enrolled      bool
	Strategy      string
	TargetVersion string
}

func managedStatusFromAPI(installation *gcom.FormattedApiInstancePlugin) *managedPluginStatus {
	if installation == nil || !installation.HasManaged() {
		return nil
	}
	managed := installation.GetManaged()
	return &managedPluginStatus{
		Enrolled:      managed.GetEnrolled(),
		Strategy:      managed.GetStrategy(),
		TargetVersion: managed.GetTargetVersion(),
	}
}

type stateVersionDecision struct {
	StateVersion string
	Warning      *diag.Diagnostic
}

func unmanagedStateVersion(priorVersion, installedVersion, catalogVersion string) string {
	if priorVersion == LatestVersion && installedVersion == catalogVersion {
		return LatestVersion
	}
	return installedVersion
}

func majorOf(raw string) (int, bool) {
	parsed, err := goversion.NewVersion(raw)
	if err != nil {
		return 0, false
	}
	return parsed.Segments()[0], true
}

func decideStateVersion(stackSlug, pluginSlug, priorVersion, installedVersion, catalogVersion string, managed *managedPluginStatus) stateVersionDecision {
	fallback := stateVersionDecision{StateVersion: unmanagedStateVersion(priorVersion, installedVersion, catalogVersion)}
	if managed == nil || !managed.Enrolled || priorVersion == "" {
		return fallback
	}
	isLatest := priorVersion == LatestVersion
	if !isLatest {
		if _, ok := majorOf(priorVersion); !ok {
			return fallback
		}
	}

	switch managed.Strategy {
	case managedStrategyAssigned:
		if managed.TargetVersion == "" || installedVersion != managed.TargetVersion {
			return fallback
		}
		decision := stateVersionDecision{StateVersion: priorVersion}
		if !isLatest && priorVersion != installedVersion {
			decision.Warning = &diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("%s on %s: version is managed by Grafana Cloud", pluginSlug, stackSlug),
				Detail: fmt.Sprintf(
					"It is kept at %s, so version %s is not applied. Set version to %s, or remove it to follow Grafana Cloud.",
					installedVersion, priorVersion, installedVersion,
				),
			}
		}
		return decision
	case managedStrategyMajorAligned:
		reference := priorVersion
		if isLatest {
			reference = catalogVersion
		}
		referenceMajor, referenceOK := majorOf(reference)
		installedMajor, installedOK := majorOf(installedVersion)
		if !referenceOK || !installedOK || referenceMajor != installedMajor {
			return fallback
		}
		decision := stateVersionDecision{StateVersion: priorVersion}
		if !isLatest && priorVersion != installedVersion {
			decision.Warning = &diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("%s on %s: minor and patch versions are managed by Grafana Cloud", pluginSlug, stackSlug),
				Detail: fmt.Sprintf(
					"It is kept at %s, so version %s is not applied. Set version to %s. To change the major version, set a version in that major.",
					installedVersion, priorVersion, installedVersion,
				),
			}
		}
		return decision
	}
	return fallback
}

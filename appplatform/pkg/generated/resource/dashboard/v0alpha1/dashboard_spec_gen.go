package v0alpha1

import (
	common "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
)

// Spec defines model for Spec.
//
// HACK: because dashboard spec is not generated using grafana-app-sdk,
// we need to copy-paste it from the grafana repo instead for now.
//
// Importing directly from grafana/grafana doesn't work,
// because the repo is not following the Go module version semantics.
//
// e.g. the necessary code is under `v11.x.x` tag and Go expects the path to be `v11/pkg/...`.
// ex. `github.com/grafana/grafana/v11 v11.4.0` would be correct,
// but `github.com/grafana/grafana v11.4.0` is not.
type Spec = common.Unstructured

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

// Submodule
// github.com/grafana/grafana/apis
// github.com/grafana/grafana/apis/dashboards/v0alpha1
// github.com/grafana/grafana/apis/dashboards/v1alpha1
// github.com/grafana/grafana/apis/dashboards/v2alpha1
// github.com/grafana/grafana/apis/playlists/v0alpha1
// github.com/grafana/grafana/apis/playlists/v1

// 1. Monorepo option
// You depend on Grafana APIs from Grafana v11.4.0
// github.com/grafana/grafana/apis => v11.4.0

// 2. Multi-module option
// github.com/grafana/grafana/apps/dashboards => v2.11.0
// github.com/grafana/grafana/apps/playlist => v1.14.0

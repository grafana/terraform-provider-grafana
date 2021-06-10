module github.com/grafana/synthetic-monitoring-api-go-client

go 1.14

require (
	github.com/google/go-cmp v0.5.6
	github.com/grafana/synthetic-monitoring-agent v0.0.22
	github.com/stretchr/testify v1.7.0
)

// Without the following replace, you get an error like
//
//     k8s.io/client-go@v12.0.0+incompatible: invalid version: +incompatible suffix not allowed: module contains a go.mod file, so semantic import versioning is required
//
// This is telling you that you cannot have a version 12.0.0 and tag
// that as "incompatible", that you should be calling the module
// something like "k8s.io/client-go/v12".
//
// 78d2af792bab is the commit tagged as v12.0.0.

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab

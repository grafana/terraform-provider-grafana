# Bundle a k6 test file with its dependencies
output "bundled_test" {
  value = provider::grafana::k6bundle("${path.module}/test.js")
}

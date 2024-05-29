// TODO(tristan): Should this be made a cloud-only test?

/*data "grafana_cloud_stack" "test" {
  slug = grafana_cloud_stack.test.slug
}*/

data "grafana_cloud_observability_aws_account" "test" {
  //stack_id = data.grafana_cloud_stack.test.id
  stack_id = "001"
  name     = "my-aws-account"
}

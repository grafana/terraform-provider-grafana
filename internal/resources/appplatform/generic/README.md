the goal of the generic app platform resource is to provide customers with something that will work even if the group/kind/version-specific resources hasn't been implemented by the corresponding team. the generic one won't have the nice strong-typing etc but it will still work

desired features of the resource
- accept a k8s-shaped resource definition (i.e. apiVersion, kind, metadata, spec)
- determine the API route based on the apiVersion and kind
- autodiscover namespace so it's not necessary to provide by user
   - always try bootdata autodiscovery first (don't cache the result). if it returns a valid stack_id, use `stacks-<id>` namespace
   - if autodiscovery doesn't find a stack_id, check for explicit provider-level `stack_id`, use `stacks-<id>`
   - if no stack_id from either source, fall back to provider-level `org_id`, use `org-<id>`
   - if none of the above, error saying you need to set either `org_id` or `stack_id`
   - this ordering means cloud users with `org_id = 1` in their config (common for legacy API compat) still get the correct cloud namespace
- accept a "secure" section with arbitrary field names (because names vary from resource to resource) 
- NO top-level overrides for api_group, version, kind, metadata, or spec. manifest alone is the single source of truth. users who need to inject Terraform variables should use HCL merge() before passing the manifest
- support only namespaced resources
- correctly determine drift of a resource when e.g. someone makes some changes via UI (moves resource to another folder, changes its spec etc). make sure the drift correction both works for `terraform import` and standard apply
  - for `metadata` fields, drift should be based on only config fields. don't cherry-pick fields — just send whatever is configured. for `metadata`, don't use a list of "supported" `metadata` fields; supported fields are only OK for the top-level manifest. updates should preserve unconfigured metadata fields (also nested)
  - for `spec`, drift should be based on whatever is in spec. anything added compared to config should cause drift
- both `metadata.name` and `metadata.uid` are accepted inside the manifest as the object identifier (they're aliases for the K8s object name)
- no `plural` field - use discovery to figure out `plural`, if that fails then you fail. input `plural` override is out of scope here
- `allow_ui_updates` should be false by default

the manifest shape 
```
  resource "grafana_apps_generic_resource" "x" {
    manifest = yamldecode(file("${path.module}/thing.yaml"))
  }

  # if you need to inject variables, use HCL merge:
  resource "grafana_apps_generic_resource" "y" {
    manifest = merge(yamldecode(file("${path.module}/thing.yaml")), {
      spec = merge(yamldecode(file("${path.module}/thing.yaml")).spec, {
        title = var.title
      })
    })
  }
```


the secure

```
secure = {
   api_token = { create = var.token }
   webhook   = { name = "existing-secret-name" }
}
secure_version = 2

```
no need for `remove =` since it doesn't make much sense from a usability PoV

## difference from typed resources (`resource.go`)

the typed app platform resources (`resource.go`) do NOT refresh spec from the API on Read — they preserve the plan state. the generic resource is more aggressive: it refreshes spec from the server on every Read so that any server-side changes (UI edits, API mutations) are detected as drift. this is intentional — the generic resource has no schema knowledge, so it can't distinguish "server default" from "user intent" and errs on the side of detecting all changes.
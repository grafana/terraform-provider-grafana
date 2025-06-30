function(catalog, schema)
  // Get resources from catalog
  local catalogParsed = std.parseYaml(catalog);
  local components = std.filter(function(obj) obj.kind == 'Component', catalogParsed);
  local resources =
    std.sort(
      std.filterMap(
        function(obj) obj.spec.type == 'terraform-resource',
        function(obj)
          // Strip 'resource-' prefix to match schema names
          if std.startsWith(obj.metadata.name, 'resource-')
          then std.substr(obj.metadata.name, 9, std.length(obj.metadata.name) - 9)
          else obj.metadata.name,
        components
      )
    );

  // Get resources from provider schema
  local s = schema.provider_schemas['registry.terraform.io/grafana/grafana'];
  local resourcesInSchema = std.objectFields(s.resource_schemas);

  // Validate
  assert resources == resourcesInSchema :
         '\nMissing in catalog: ' + std.setDiff(resourcesInSchema, resources)
         + '\nMissing in schema: ' + std.setDiff(resources, resourcesInSchema);
  'Valid!'

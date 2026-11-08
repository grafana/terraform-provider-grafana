function(catalog, schema)
  local catalogParsed = std.parseYaml(catalog);
  local components = std.filter(function(obj) obj.kind == 'Component', catalogParsed);
  local s = schema.provider_schemas['registry.terraform.io/grafana/grafana'];

  // Get resources from catalog
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
  local resourcesInSchema = std.objectFields(s.resource_schemas);

  // Validate resources
  assert resources == resourcesInSchema :
         '\nResources missing in catalog: ' + std.setDiff(resourcesInSchema, resources)
         + '\nResources missing in schema: ' + std.setDiff(resources, resourcesInSchema);

  // Get data sources from catalog
  local dataSources =
    std.sort(
      std.filterMap(
        function(obj) obj.spec.type == 'terraform-data-source',
        function(obj)
          // Strip 'datasource-' prefix to match schema names
          if std.startsWith(obj.metadata.name, 'datasource-')
          then std.substr(obj.metadata.name, 11, std.length(obj.metadata.name) - 11)
          else obj.metadata.name,
        components
      )
    );

  // Get data sources from provider schema
  local dataSourcesInSchema = std.objectFields(s.data_source_schemas);

  // Validate data sources
  assert dataSources == dataSourcesInSchema :
         '\nData sources missing in catalog: ' + std.setDiff(dataSourcesInSchema, dataSources)
         + '\nData sources missing in schema: ' + std.setDiff(dataSources, dataSourcesInSchema);

  'Valid!'

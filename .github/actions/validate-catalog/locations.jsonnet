function(catalog, pathPrefix='')
  // Get resources from catalog
  local catalogParsed = std.parseYaml(catalog);
  local locations = std.filter(function(obj) obj.kind == 'Location', catalogParsed);
  local targets =
    std.flatMap(
      function(obj) obj.spec.targets,
      locations
    );

  std.lines(
    std.map(
      function(target) pathPrefix + target,
      targets
    )
  )

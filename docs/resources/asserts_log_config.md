# grafana_asserts_log_config

Manages Asserts Log Configuration through Grafana API.

## Example Usage

### Basic Configuration

```hcl
resource "grafana_asserts_log_config" "production" {
  name = "production"
  config = <<-EOT
    name: production
    logConfig:
      enabled: true
      retention: "30d"
      maxLogSize: "1GB"
      compression: true
      filters:
        - level: "ERROR"
        - level: "WARN"
        - service: "api"
  EOT
}
```

### Minimal Configuration

```hcl
resource "grafana_asserts_log_config" "development" {
  name = "development"
  config = <<-EOT
    name: development
    logConfig:
      enabled: true
  EOT
}
```

### Advanced Configuration

```hcl
resource "grafana_asserts_log_config" "staging" {
  name = "staging"
  config = <<-EOT
    name: staging
    logConfig:
      enabled: true
      retention: "7d"
      maxLogSize: "500MB"
      compression: false
      filters:
        - level: "ERROR"
        - level: "WARN"
        - level: "INFO"
        - service: "web"
        - service: "api"
      sampling:
        rate: 0.1
        maxTracesPerSecond: 100
  EOT
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, Forces new resource) The name of the log configuration environment.
* `config` - (Required) The log configuration in YAML format.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `name` - The name of the log configuration environment.
* `config` - The log configuration in YAML format.

## Import

Log configurations can be imported using the environment name:

```bash
terraform import grafana_asserts_log_config.production production
```

## Configuration Schema

The `config` field accepts YAML configuration with the following structure:

```yaml
name: <environment-name>
logConfig:
  enabled: <boolean>
  retention: <duration>  # e.g., "7d", "30d", "1h"
  maxLogSize: <size>     # e.g., "100MB", "1GB"
  compression: <boolean>
  filters:
    - level: <log-level>    # e.g., "ERROR", "WARN", "INFO", "DEBUG"
    - service: <service-name>
  sampling:
    rate: <float>           # 0.0 to 1.0
    maxTracesPerSecond: <integer>
```

### Configuration Options

- **enabled**: Whether log collection is enabled for this environment
- **retention**: How long to retain logs (e.g., "7d", "30d", "1h")
- **maxLogSize**: Maximum size for log files before rotation
- **compression**: Whether to compress log files
- **filters**: Array of filters to apply to log collection
- **sampling**: Optional sampling configuration for high-volume environments

## Notes

- The `name` field is used as the resource identifier and cannot be changed after creation
- Configuration changes will trigger an update operation
- The resource supports eventual consistency - changes may take a few seconds to propagate
- All configuration is validated by the Asserts API before being applied
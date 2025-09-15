#!/bin/bash

# Import existing log configurations
# Replace the stack ID and environment names with your actual values

# Import production environment
terraform import grafana_asserts_log_config.production production

# Import development environment  
terraform import grafana_asserts_log_config.development development

# Import staging environment
terraform import grafana_asserts_log_config.staging staging

# Import test environment
terraform import grafana_asserts_log_config.test test

echo "Import completed. Run 'terraform plan' to verify the import."